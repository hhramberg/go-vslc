// Package arm provides means to generate aarch64 assembly code from the intermediate syntax tree representation.
package arm

import (
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"vslc/src/backend/regfile"
)

import (
	"vslc/src/ir"
	"vslc/src/util"
)

// ----------------------------
// ----- Type definitions -----
// ----------------------------

// register defines a physical register, of either type integer or floating point, and an index (r0-30 or v0-v31).
type register struct {
	typ  int // Type of register (integer or floating point).
	size int // Size of register in bits (64 or 32).
	idx  int // Index of register (0 = x0, 1 = x1, 4 = v4 etc.).
	use  int // Counter of when this register was used. Lower value means a longer time has passed since last this register was used.
}

// registerFile defines a virtual register file during compilation time. It holds 32 integer and 32 floating point
// registers per aarch64 ABI.
type registerFile struct {
	regi []register // General purpose integer registers of register file.
	regf []register // Floating point registers of register file.
	usei int        // Number of integer registers used since last function call. Used for detecting when to push registers to stack.
	usef int        // Number of floating point registers used since last function call. Used for detecting when to push registers to stack.
	seqi int        // Sequence number of integer register uses, for the LRU algorithm.
	seqf int        // Sequence number of floating point register uses, for the LRU algorithm.
}

// RegisterFile defines a virtual register file during compilation time. It holds 32 integer and 32 floating point
// registers per aarch64 ABI.
type RegisterFile struct {
	regi []regfile.Register
	regf []regfile.Register
}

// ---------------------
// ----- Constants -----
// ---------------------

const labelFloat = "_LFLOAT_"
const labelString = "_STR_"
const labelMain = "main"                         // String literal of name of main function as defined in the output assembler.
const labelPrintfInt = "_printf_fmt_int"         // Used by printf to print integers.
const labelPrintfFloat = "_printf_fmt_float"     // Used by printf to print floats.
const labelPrintfString = "_printf_fmt_string"   // Used by printf to print strings.
const labelPrintfNewline = "_printf_fmt_newline" // Used by printf to print strings.

const (
	i = ir.DataInteger // i indicates integer type. // TODO: Change to LIR Int?
	f = ir.DataFloat   // f indicates floating point type. // TODO: Change to LIR Float?
)

const (
	bitSize64  = 64 // Number of bits in 64-bit architecture.
	bitSize32  = 32 // Number of bits in 32-bit architecture.
	wordSize64 = 8  // Word size in bytes for 64-bit architecture.
	wordSize32 = 4  // Word size in bytes for 32-bit architecture.
)

// stackAlign defines the stack alignment of the aarch64 stack. If the stack grows or shrinks, it must do so in
// multiples of the stackAlign value.
const stackAlign = 16 // Per chapter 5.2.2.1 of https://documentation-service.arm.com/static/5fa43415b1a7c5445f292563?token=

// paramReg defines the maximum number of arguments that can go in parameters.
const paramReg = 8

const minImm = -2048 // minImm defines the minimum 12-bit signed immediate value.
const maxImm = 2047  // maxImm defines the maximum 12-bit signed immediate value.

// Integer general purpose registers.
const (
	r0 = iota
	r1
	r2
	r3
	r4
	r5
	r6
	r7
	r8
	r9
	r10
	r11
	r12
	r13
	r14
	r15
	r16
	r17
	r18
	r19
	r20
	r21
	r22
	r23
	r24
	r25
	r26
	r27
	r28
	r29
	r30
)

// Floating point general purpose registers.
const (
	v0 = iota
	v1
	v2
	v3
	v4
	v5
	v6
	v7
	v8
	v9
	v10
	v11
	v12
	v13
	v14
	v15
	v16
	v17
	v18
	v19
	v20
	v21
	v22
	v23
	v24
	v25
	v26
	v27
	v28
	v29
	v30
	v31
)

// From: https://documentation-service.arm.com/static/5fa43415b1a7c5445f292563?token=
//
// General purpose integer registers.
//
// r19-28	Callee saved registers.
// r18		Do not use for platform independent code.
// r9-r17	Temporary registers (caller saved).
// r8		Indirect result LOCATION register.
// r0-r7	Parameter and result registers. MAY use as temporary, calle saved, registers.
//
// Floating point registers.
//
// v0-v7	Parameter and result registers. MAY use as temporary, calle saved, registers.
// v8-v15	Callee saved registers.
// v16-v31	Temporary registers.

const (
	lr = r30     // Link register.
	fp = r29     // Frame pointer (top of stack frame).
	sp = r30 + 1 // Stack pointer (bottom of stack frame).
)

// -------------------
// ----- globals -----
// -------------------

// regi defines print friendly string representations of the general purpose integer registers.
var regi = [...]string{
	"x0",
	"x1",
	"x2",
	"x3",
	"x4",
	"x5",
	"x6",
	"x7",
	"x8",
	"x9",
	"x10",
	"x11",
	"x12",
	"x13",
	"x14",
	"x15",
	"x16",
	"x17",
	"x18",
	"x19",
	"x20",
	"x21",
	"x22",
	"x23",
	"x24",
	"x25",
	"x26",
	"x27",
	"x28",
	"x29",
	"x30",
	"SP",
}

// regf defines print friendly string representations of the floating point registers.
var regf = [...]string{
	"v0",
	"v1",
	"v2",
	"v3",
	"v4",
	"v5",
	"v6",
	"v7",
	"v8",
	"v9",
	"v10",
	"v12",
	"v13",
	"v14",
	"v15",
	"v16",
	"v17",
	"v18",
	"v19",
	"v20",
	"v21",
	"v22",
	"v23",
	"v24",
	"v25",
	"v26",
	"v27",
	"v28",
	"v29",
	"v30",
}

// wordSize defines the word size of the aarch64 architecture to generate.
var wordSize = wordSize64 // Default to 64-bit architecture.

// bitSize defines the bit size of the aarch64 architecture to generate.
var bitSize = bitSize64 // default to 64-bit architecture.

// floatStrings stores statically, compile time generated, floating point values, written to the data segment.
var floatStrings = struct {
	s []string
	sync.Mutex
}{
	s: make([]string, 0, 16), // Expect no more than 16 static floats.
}

// ---------------------
// ----- functions -----
// ---------------------

// GenArm recursively generates ARM v8 (aarch64) assembler code from the intermediate representation.
func GenArm(opt util.Options) error {
	// Generate .text section.
	wr := util.NewWriter()
	defer wr.Close()
	wr.Write("\t.arch\tarmv8-a\n")
	wr.Write("\t.file\t%q\n", filepath.Base(opt.Src))
	wr.Write("\t.text\n")

	wr.Write("\t.global\t%s\n", labelMain)
	wr.Write("\t.type\t%s, %%function\n", labelMain)
	wr.Flush() // Write to top of output.

	// Generate functions.
	if opt.Threads > 1 {
		// Parallel.
		t := opt.Threads
		l := len(ir.Funcs.F)
		if t > l {
			t = l
		}
		n := l / t   // jobs per worker go routine.
		res := l % t // Residual jobs.

		start := 0
		end := n

		wg := sync.WaitGroup{}
		wg.Add(t)
		cerr := make(chan error)

		for i1 := 0; i1 < t; i1++ {
			// Launch t go routines.
			if i1 < res {
				// Worker should do one extra residual job.
				end++
			}

			// Spawn worker go routine.
			go func(start, end int, wg *sync.WaitGroup, cerr chan error) {
				defer wg.Done()
				wr := util.NewWriter()
				defer wr.Close()

				for _, e1 := range ir.Funcs.F[start:end] {
					if err := genFunction(e1, &wr, opt); err != nil {
						cerr <- err
					}
				}
			}(start, end, &wg, cerr)
			start = end
			end += n
		}

		wg.Wait()
	} else {
		// Sequential.
		for _, e1 := range ir.Funcs.F {
			if err := genFunction(e1, &wr, opt); err != nil {
				return err
			}
		}
	}

	// Generate main function.
	var callee *ir.Symbol
	rf := CreateRegisterFile()

	// Find first defined function, which will be called implicitly from main.
	for _, e1 := range ir.Root.Children {
		if e1.Typ == ir.FUNCTION {
			callee = e1.Entry
			break
		}
	}

	// Generate implicit main function for program entry.
	if err := genMain(&rf, callee, &wr); err != nil {
		return err
	}

	// Generate global data.
	wr.Write("\t.data\n")
	for _, v := range ir.Global.HT {
		if v.Typ == ir.SymGlobal {
			wr.Label(v.Name)
			// Write globals with initial values 0. VSL doesn't support variable initialisation on declaration.
			wr.Write("\t.xword\t0\n")
		}
	}

	// Generate string data.
	for i1, e1 := range ir.Strings.St {
		wr.Label(fmt.Sprintf("%s%d", labelString, i1))
		wr.Write("\t.asciz\t%q\n", e1)
	}

	// Generate float constant data.
	for i1, e1 := range floatStrings.s {
		wr.Label(fmt.Sprintf("%s%d", labelFloat, i1))
		wr.Write("\t.xword\t%s\n", e1)
	}

	// Generate printf format strings.
	wr.Label(labelPrintfInt)
	wr.Write("\t.asciz\t\"%%d\"\n")
	wr.Label(labelPrintfFloat)
	wr.Write("\t.asciz\t\"%%f\"\n")
	wr.Label(labelPrintfString)
	wr.Write("\t.asciz\t\"%%s\"\n")
	wr.Label(labelPrintfNewline)
	wr.Write("\t.asciz\t\"\\n\"\n")

	return nil
}

// gen generates aarch64 assembler recursively. An error is returned if something went wrong.
func gen(n *ir.Node, fun *ir.Symbol, rf *registerFile, wr *util.Writer, st, ls *util.Stack) error {
	switch n.Typ {
	case ir.BLOCK:
		scope := ir.SymTab{HT: make(map[string]*ir.Symbol, 16)}
		st.Push(&scope.HT)
		for _, e1 := range n.Children {
			if err := gen(e1, fun, rf, wr, st, ls); err != nil {
				st.Pop()
				return err
			}
		}
		st.Pop()
	case ir.EXPRESSION:
		if _, err := genExpression(n, rf, wr, st); err != nil {
			return err
		}
	case ir.IF_STATEMENT:
		if err := genIf(n, fun, rf, wr, st, ls); err != nil {
			return err
		}
	case ir.WHILE_STATEMENT:
		if err := genWhile(n, fun, rf, wr, st, ls); err != nil {
			return err
		}
	//case ir.DECLARATION: // Allocated during function head and globals generation.
	case ir.NULL_STATEMENT:
		if err := genContinue(wr, ls); err != nil {
			return err
		}
	case ir.ASSIGNMENT_STATEMENT:
		if err := genAssignment(n, rf, wr, st); err != nil {
			return err
		}
	case ir.PRINT_STATEMENT:
		if err := genPrint(n, rf, wr, st); err != nil {
			return err
		}
	case ir.RETURN_STATEMENT:
		if err := genReturn(n, fun, rf, wr, st); err != nil {
			return err
		}
	default:
		for _, e1 := range n.Children {
			if err := gen(e1, fun, rf, wr, st, ls); err != nil {
				return err
			}
		}
	}
	return nil
}

// genAssignment generates aarch64 assembler for assigning a value to a named variable.
func genAssignment(n *ir.Node, rf *registerFile, wr *util.Writer, st *util.Stack) error {
	name := n.Children[0].Data.(string)
	c2 := n.Children[1]

	switch c2.Typ {
	case ir.INTEGER_DATA:
		r := rf.lruI()
		if err := genLoadImmToRegister(c2.Data.(int), r, wr); err != nil {
			return err
		}
		if err := storeIdentifier(r, name, rf, wr, st); err != nil {
			return err
		}
	case ir.FLOAT_DATA:
		floatStrings.Lock()
		idx := len(floatStrings.s)
		floatStrings.s = append(floatStrings.s, fmt.Sprintf("%x", c2.Data.(float64)))
		floatStrings.Unlock()
		r := rf.lruF()
		wr.Write("\tldr\t%s, %s%d\n", r.String(), labelFloat, idx)
		if err := storeIdentifier(r, name, rf, wr, st); err != nil {
			return err
		}
	case ir.EXPRESSION:
		r, err := genExpression(c2, rf, wr, st)
		if err != nil {
			return err
		}
		if err = storeIdentifier(r, name, rf, wr, st); err != nil {
			return err
		}
	case ir.IDENTIFIER_DATA:
		r, err := loadIdentifier(c2.Data.(string), rf, wr, st)
		if err != nil {
			return err
		}
		if err = storeIdentifier(r, name, rf, wr, st); err != nil {
			return err
		}
	default:
		return fmt.Errorf("compiler error: expected node type INTEGER_DATA, FLOAT_DATA, EXPRESSIO or IDENTIFIER, got %s",
			c2.Type())
	}
	return nil
}

// loadIdentifier loads an identifier from local scopes, function parameters or global scope. If the identifier isn't
// found, an error is returned. The identifier value is loaded into the register pointed to by the returned register
// pointer.
func loadIdentifier(name string, rf *registerFile, wr *util.Writer, st *util.Stack) (*register, error) {
	// Check local scopes and function parameters.
	for i1 := 1; i1 < st.Size(); i1++ {
		e1 := st.Get(i1)
		if e1 == nil {
			return nil, errors.New("compiler error: scope stack is malformed")
		}
		scope := e1.(*map[string]*ir.Symbol)
		ident, ok := (*scope)[name]
		if !ok {
			continue
		}

		// Found identifier, calculate offset from FP.
		offset := -(wordSize * 2)            // FP and LR
		offset -= wordSize * (ident.Seq + 1) // Offset within function's defined variables.

		// Check variables datatype.
		var r *register
		if ident.DataTyp == i {
			// Integer variable.
			r = rf.lruI()
		} else {
			// Floating point variable.
			r = rf.lruF()
		}

		// Load from stack to register r.
		wr.Write("\tldr\t%s, [%s, #%d]\n", r.String(), regi[fp], offset)
		return r, nil
	}

	// Check globals.
	g := st.Get(st.Size())
	if g == nil {
		return nil, errors.New("compiler error: scope stack is malformed")
	}
	glob := g.(*map[string]*ir.Symbol)
	ident, ok := (*glob)[name]
	if !ok {
		return nil, fmt.Errorf("undeclared identifier %s", name)
	}

	var r *register
	if ident.DataTyp == i {
		// Integer variable.
		r = rf.lruI()
	} else {
		// Floating point variable.
		r = rf.lruF()
	}

	// Load from data segment to register r.
	wr.Write("\tldr\t%s, =%s\n", r.String(), ident.Name)
	return r, nil
}

// storeIdentifier stores the contents of register r to the identifier with the given name. An error is returned if
// something went wrong. Floats being stored to integer variables are cast to integer before store.
func storeIdentifier(r *register, name string, rf *registerFile, wr *util.Writer, st *util.Stack) error {
	// Check local scopes and function parameters.
	for i1 := 1; i1 < st.Size(); i1++ {
		e1 := st.Get(i1)
		if e1 == nil {
			return errors.New("compiler error: scope stack is malformed")
		}
		scope := e1.(*map[string]*ir.Symbol)
		ident, ok := (*scope)[name]
		if !ok {
			continue
		}

		if r.typ == f && ident.DataTyp != r.typ {
			// Cast float to integer.
			tmp := rf.lruI()
			wr.Write("\tscvtf\t%s, %s\n", tmp.String(), r.String())
			r = tmp
		}

		// Found identifier, calculate offset from FP.
		offset := -(wordSize * 2)            // FP and LR
		offset -= wordSize * (ident.Seq + 1) // Offset within function's defined variables.

		// Store register r to stack.
		wr.Write("\tstr\t%s, [%s, #%d]\n", r.String(), regi[fp], offset)
		return nil
	}

	// Check globals.
	g := st.Get(st.Size())
	if g == nil {
		return errors.New("compiler error: scope stack is malformed")
	}
	glob := g.(map[string]*ir.Symbol)
	ident, ok := glob[name]
	if !ok {
		return fmt.Errorf("undeclared identifier %s", name)
	}

	// Store register r in data segment.
	wr.Write("\tstr\t%s, =%s\n", r.String(), ident.Name)
	return nil
}

// genLoadImmToRegister generates a load of a signed integer imm to the destination register r.
func genLoadImmToRegister(imm int, r *register, wr *util.Writer) error {
	if imm >= minImm && imm <= maxImm {
		wr.Write("\tmov\t%s, #%d\n", r.String(), imm)
		return nil
	}
	if imm&0xFFFF == imm {
		// Can do with 16-bit move.
		wr.Write("\tmovz\t%s, #%d\n", r.String(), imm&0x000000000000FFFF)
		return nil
	}
	if imm&0xFFFFFFFF == imm {
		// Can do with 32-bit move.
		wr.Write("\tmovz\t%s, #%d\n", r.String(), imm&0x000000000000FFFF)
		wr.Write("\tmovk\t%s, #%d, lsl 16\n", r.String(), imm&0x00000000FFFF0000)
		return nil
	}
	if r.size != bitSize64 {
		return errors.New("trying to load a 64-bit value into a 32-bit register")
	}
	if imm&0xFFFFFFFFFFFF == imm {
		// Can do with 48-bit move.
		wr.Write("\tmovz\t%s, #%d\n", r.String(), imm&0x000000000000FFFF)
		wr.Write("\tmovk\t%s, #%d, lsl 16\n", r.String(), imm&0x00000000FFFF0000)
		wr.Write("\tmovk\t%s, #%d, lsl 32\n", r.String(), imm&0x0000FFFF00000000)
		return nil
	}
	// Must do 64-bit move.
	wr.Write("\tmovz\t%s, #%d\n", r.String(), imm&0x000000000000FFFF)
	wr.Write("\tmovk\t%s, #%d, lsl 16\n", r.String(), imm&0x00000000FFFF0000)
	wr.Write("\tmovk\t%s, #%d, lsl 32\n", r.String(), imm&0x0000FFFF00000000)
	wr.Write("\tmovk\t%s, #%d, lsl 48\n", r.String(), uint(imm)&0xFFFF000000000000)
	return nil
}

// genMain generates an implicit main function that checks input command-line arguments and calls the function callee.
// After the function callee returns the main function exits the program with the return value of the call to callee.
// If the return value of callee is a floating point value, the value is cast to integer.
func genMain(rf *registerFile, callee *ir.Symbol, wr *util.Writer) error {
	wr.Write("\n")
	wr.Label(labelMain)

	nf, ni := 0, 0 // Number of floating point and integer parameters respectively.
	for _, e1 := range callee.Params {
		if e1.DataTyp == i {
			ni++
		} else {
			nf++
		}
	}

	// Set up stack.
	sa := wordSize << 1 // argc and argv.
	if nf > paramReg {
		// Add stack for integer arguments being passed to callee.
		sa += nf - paramReg
	}
	if ni > paramReg {
		// Add stack for floating point arguments being passed to callee.
		sa += ni - paramReg
	}

	spill := 0 // Needed for adjusting where to start storing arguments, such that last argument hits FP of callee.
	sa *= wordSize
	res := sa % stackAlign
	if res != 0 {
		spill += stackAlign - res
		sa += stackAlign - res
	}
	wr.Write("\tsub\t%s, %s, #%d\n", rf.SP().String(), rf.SP().String(), sa)
	wr.Write("\tstr\t%s, [%s, #%d]\n", rf.regi[r0].String(), rf.SP().String(), sa-wordSize)    // argc.
	wr.Write("\tstr\t%s, [%s, #%d]\n", rf.regi[r1].String(), rf.SP().String(), sa-wordSize<<1) // argv.

	// Jump labels for error checking.
	largcok := util.NewLabel(util.LabelIfEnd) // Jump to label if argc matches parameter count of callee.
	largverr := util.NewLabel(util.LabelIf)   // Jump to label if parameter is not integer or float.
	lcall := util.NewLabel(util.LabelJump)    // Jump to label when all parameters are ok.

	// Check parameter count and argc.
	wr.Write("\tstr\t%s, [%s, #%d]\n", rf.regi[r0].String(), rf.SP().String(), sa-wordSize) // This is bloated, but it's idiomatic to load argc from the stack.
	wr.Write("\tcmp\t%s, #%d\n", rf.regi[r0].String(), callee.Nparams+1)                    // First argument is application path.
	wr.Write("\tb.eq\t%s\n", largcok)

	// argc is not ok.
	idx := len(ir.Strings.St)
	ir.Strings.St = append(
		ir.Strings.St,
		fmt.Sprintf("Argument error: expected %d arguments, got %%d\n", callee.Nparams),
	)

	// Load argc - 1 into r1. It's safe to overwrite r1, because we're not going to use it.
	wr.Write("\tsub\t%s, %s, #%d\n", rf.regi[r1].String(), rf.regi[r0].String(), 1)

	// Load format string and call printf.
	wr.Write("\tadrp\t%s, %s%d\n", rf.regi[r0].String(), labelString, idx)
	wr.Write("\tadd\t%s, %s, :lo12:%s%d\n", rf.regi[r0].String(), rf.regi[r0].String(), labelString, idx)
	wr.Write("\tbl\tprintf\n")

	// Set return code and return.
	wr.Write("\tmov\t%s, #%d\n", rf.regi[r0].String(), 1)
	wr.Write("\tadd\t%s, %s, #%d\n", rf.SP().String(), rf.SP().String(), sa)
	wr.Write("\tret\n")

	// argc is ok.
	wr.Label(largcok)

	ii := 0           // Number of integer arguments.
	fi := 0           // Number of floating point arguments.
	fzero := false    // Set to true if the label for floating point zero has been generated.
	var flabel string // Label for loading the float constant 0.0.
	var argvreg *register
	if ni > 0 {
		// Use r8 instead of r0 for argv pointer. r0 is already assigned a parameter.
		argvreg = &rf.regi[r8]
	} else {
		argvreg = &rf.regi[r0]
	}
	for i1, e1 := range callee.Params {
		// Move char pointer to r0. Increment r1 to next string.
		wr.Write("\tldr\t%s, [%s, #%d]\n", argvreg.String(), rf.SP().String(), sa-wordSize<<1) // Load argv into r8.

		if e1.DataTyp == i {
			// Parse argv[i1] as int.

			// Run atoi.
			wr.Write("\tbl\tatoi\n")

			// Check return value of atoi. Zero means string isn't an integer. Cannot parse "0", because atoi is stupid.
			wr.Write("\tcmp\t%s, #%d\n", rf.regi[r0].String(), 0)
			wr.Write("\tmov\t%s, #%d\n", rf.regi[r1].String(), i1+1) // Hack the printf by moving parameter index to r1. Wastes one cycle.
			wr.Write("\tb.eq\t%s\n", largverr)                       // Got error from atof, branch to argverror.
			if ii < paramReg {
				// Store parameter in register.
				wr.Write("\tmov\t%s, %s\n", rf.regi[r0].String(), rf.regi[ii].String())
			} else {
				// Store parameter on stack.
				wr.Write("\tstr\t%s, [%s, #%d]\n", rf.regi[r0].String(), rf.LR(), -(spill + wordSize*e1.Seq))
			}
			ii++
		} else {
			// Parse argv[i1] as float.

			if !fzero {
				flabel = floatToGlobalString(0.0)
				fzero = true
			}

			// Run atoi.
			wr.Write("\tbl\tatof\n")

			// Check return value of atof. 0.0 means string isn't a float. Cannot parse "0.0", because atof is stupid.
			loadFloatToRegister(flabel, &rf.regf[v1], wr) // Move 0.0 into v1.
			wr.Write("\tfcmp\t%s, %s\n", rf.regf[v0].String(), rf.regf[v1].String())
			wr.Write("\tmov\t%s, #%d\n", rf.regi[r1].String(), i1+1) // Hack the printf by moving parameter index to r1. Wastes one cycle.
			wr.Write("\tb.eq\t%s\n", largverr)                       // Got error from atof, branch to argverror.
			if fi < paramReg {
				// Store parameter in register.
				wr.Write("\tmov\t%s, %s\n", rf.regf[v0].String(), rf.regf[fi].String())
			} else {
				// Store parameter on stack.
				wr.Write("\tstr\t%s, [%s, #%d]\n", rf.regf[v0].String(), rf.LR(), -(spill + wordSize*e1.Seq))
			}
			fi++
		}

		// Increment argv and store on stack.
		wr.Write("\tadd\t%s, %s, #%d\n", argvreg.String(), rf.regi[r8].String(), wordSize)     // Increment argv.
		wr.Write("\tstr\t%s, [%s, #%d]\n", argvreg.String(), rf.SP().String(), sa-wordSize<<1) // Store argv++.
	}

	// When done with parameters, cause program to jump to call function under the argv error handling logic.
	wr.Write("\tb\t%s\n", lcall)

	// argv errors jump here.
	wr.Label(largverr)
	idx = len(ir.Strings.St)
	ir.Strings.St = append(
		ir.Strings.St,
		"Argument error: argument %d is either not int or float\n",
	)

	// Load format string and call printf.
	wr.Write("\tadrp\t%s, %s%d\n", rf.regi[r0].String(), labelString, idx)
	wr.Write("\tadd\t%s, %s, :lo12:%s%d\n", rf.regi[r0].String(), rf.regi[r0].String(), labelString, idx)
	wr.Write("\tbl\tprintf\n")

	// Set return code and return.
	wr.Write("\tmov\t%s, #%d\n", rf.regi[r0].String(), 1)
	wr.Write("\tadd\t%s, %s, #%d\n", rf.SP().String(), rf.SP().String(), sa)
	wr.Write("\tret\n")

	// Go here when ready to call callee.
	wr.Label(lcall)
	wr.Write("\tbl\t%s\n", callee.Name)

	// Move float result from v0 to r0 if necessary.
	if callee.DataTyp == f {
		wr.Write("\tfcvts\t%s, %s\n", rf.regf[v0].String(), rf.regi[r0].String())
	}

	// De-allocate stack and return, result from callee is already in r0.
	wr.Write("\tadd\t%s, %s, #%d\n", rf.SP().String(), rf.SP().String(), sa)
	wr.Write("\tret\n")
	return nil
}

// CreateRegisterFile creates an aarch64 register file with 32 general purpose integer registers and 32 floating point
// register of configuration specific width.
// TODO: Delete.
func CreateRegisterFile() registerFile {
	rf := registerFile{
		regi: make([]register, 32),
		regf: make([]register, 32),
	}

	// Initiate registers.
	for i1 := range rf.regi {
		rf.regi[i1] = register{
			typ:  i,
			size: bitSize,
			idx:  i1,
		}
		rf.regf[i1] = register{
			typ:  f,
			size: bitSize,
			idx:  i1,
		}
	}
	return rf
}

func CreateRegisterFile2() RegisterFile {
	rf := RegisterFile{
		regi: make([]regfile.Register, 32),
		regf: make([]regfile.Register, 32),
	}

	// Initiate registers.
	for i1 := range rf.regi {
		rf.regi[i1] = &register{
			typ:  i,
			size: bitSize,
			idx:  i1,
		}
		rf.regf[i1] = &register{
			typ:  f,
			size: bitSize,
			idx:  i1,
		}
	}
	return rf
}

// floatToGlobalString takes a float immediate and generates a compile time float constant for the data segment.
// The label is returned.
func floatToGlobalString(fl float64) string {
	floatStrings.Lock()
	idx := len(floatStrings.s)
	label := fmt.Sprintf("%s%d", labelString, idx)
	floatStrings.s = append(floatStrings.s, fmt.Sprintf("%x", fl))
	floatStrings.Unlock()
	return label
}

// loadFloatToRegisters generates aarch64 assembler for loading an existing float label into register r.
func loadFloatToRegister(label string, r *register, wr *util.Writer) {
	wr.Write("\tadrp\t%s, %s\n", r.String(), label)
	wr.Write("\tadd\t%s, %s, :lo12:%s\n", r.String(), r.String(), label)
}

// ----------------------------
// ----- Register methods -----
// ----------------------------

// String returns the assembler string of the register.
func (r register) String() string {
	if r.typ == i {
		return regi[r.idx]
	}
	return regf[r.idx]
}

// Id returns the index of the register r.
func (r register) Id() int {
	return r.idx
}

// Type returns the register type, 0 = integer and 1 = floating point.
func (r register) Type() int {
	return r.typ
}

// Used returns true if the register has been allocated (is in use).
func (r register) Used() bool {
	return r.use == 1
}

// Use sets the use flag of the register r to true.
func (r register) Use() {
	r.use = 1
}

// Free sets the use flag of register r to false.
func (r register) Free() {
	r.use = 0
}

// ---------------------------------
// ----- Register file methods -----
// ---------------------------------

// lruI uses the least recently used first algorithm to select a temporary integer register for use.
func (rf *registerFile) lruI() *register {
	res := &rf.regi[0]

	// TODO: Add stack buffering for saturated register file.

	// Check argument registers.
	for _, e1 := range rf.regi[:r8] {
		if e1.use < res.use {
			res = &e1
		}
	}

	// Check block 1 temporary registers.
	for _, e1 := range rf.regi[r9:r18] {
		if e1.use < res.use {
			res = &e1
		}
	}

	// Check block 2 temporary registers.
	for _, e1 := range rf.regi[r19:] {
		if e1.use < res.use {
			res = &e1
		}
	}
	rf.usei++
	res.use = rf.seqi
	rf.seqi++
	return res
}

// lruINoArg is the same as lruI, but doesn't return argument registers r0-r7.
func (rf *registerFile) lruINoArg() *register {
	res := &rf.regi[r9]

	// TODO: Add stack buffering for saturated register file.

	// Check block 1 temporary registers.
	for _, e1 := range rf.regi[r9:r18] {
		if e1.use < res.use {
			res = &e1
		}
	}

	// Check block 2 temporary registers.
	for _, e1 := range rf.regi[r19:] {
		if e1.use < res.use {
			res = &e1
		}
	}
	rf.usei++
	res.use = rf.seqi
	rf.seqi++
	return res
}

// lruF uses the least recently used first algorithm to select a temporary floating point register for use.
func (rf *registerFile) lruF() *register {
	res := &rf.regf[0]

	// TODO: Add stack buffering for saturated register file.

	// Check all registers. Floating point registers are all general purpose.
	for _, e1 := range rf.regf {
		if e1.use < res.use {
			res = &e1
		}
	}
	rf.usef++
	res.use = rf.seqf
	rf.seqf++
	return res
}

// lruFNoArg is the same as lruF, but doesn't return argument registers v0-v7.
func (rf *registerFile) lruFNoArg() *register {
	res := &rf.regf[v8]

	// TODO: Add stack buffering for saturated register file.

	// Check all registers. Floating point registers are all general purpose.
	for _, e1 := range rf.regf[v9:] {
		if e1.use < res.use {
			res = &e1
		}
	}
	rf.usef++
	res.use = rf.seqf
	rf.seqf++
	return res
}

// GetI returns integer register with index i.
func (rf registerFile) GetI(i int) *register {
	if i < 0 || i > len(rf.regi) {
		return nil
	}
	return &rf.regi[i]
}

// GetF returns floating point register with index i.
func (rf registerFile) GetF(i int) *register {
	if i < 0 || i > len(rf.regf) {
		return nil
	}
	return &rf.regf[i]
}

// GetNextTempI returns the next available integer register that hasn't been allocated yet.
// If no registers are vacant, <nil> is returned.
func (rf registerFile) GetNextTempI() *register {
	for i1, e1 := range rf.regi {
		if !e1.Used() {
			return &rf.regi[i1]
		}
	}
	return nil
}

// GetNextTempF returns the next available floating point register that hasn't been allocated yet.
// If no registers are vacant, <nil> is returned.
func (rf registerFile) GetNextTempF() *register {
	for i1, e1 := range rf.regf {
		if !e1.Used() {
			return &rf.regf[i1]
		}
	}
	return nil
}

// SP returns a pointer to the register file's stack pointer.
func (rf registerFile) SP() *register {
	return &rf.regi[sp]
}

// FP returns a pointer to the register file's frame pointer.
func (rf registerFile) FP() *register {
	return &rf.regi[fp]
}

// LR returns a pointer to the register file's link register.
func (rf registerFile) LR() *register {
	return &rf.regi[lr]
}

// GetI returns integer register with index i.
func (rf RegisterFile) GetI(i int) regfile.Register {
	if i < 0 || i > len(rf.regi) {
		return nil
	}
	return rf.regi[i]
}

// GetF returns floating point register with index i.
func (rf RegisterFile) GetF(i int) regfile.Register {
	if i < 0 || i > len(rf.regf) {
		return nil
	}
	return rf.regf[i]
}

// GetNextTempI returns the next available integer register that hasn't been allocated yet.
// If no registers are vacant, <nil> is returned.
func (rf RegisterFile) GetNextTempI() regfile.Register {
	// Use r8-28. Registers r19-28 are callee-saved.
	for i1, e1 := range rf.regi[r8:r29] {
		if e1.(*register).use == 0 {
			rf.regi[i1+r8].(*register).use = 1
			return rf.regi[i1+r8]
		}
	}
	return nil
}

// GetNextTempF returns the next available floating point register that hasn't been allocated yet.
// If no registers are vacant, <nil> is returned.
func (rf RegisterFile) GetNextTempF() regfile.Register {
	// Use v8-31. Registers v8-15 are callee-saved.
	for i1, e1 := range rf.regf[v8:] {
		if e1.(*register).use == 0 {
			rf.regf[i1+v8].(*register).use = 1
			return rf.regf[i1+v8]
		}
	}
	return nil
}

// GetNextTempIExclude returns the next available integer register that hasn't been allocated yet and is
// not in the exclusion list. If no registers are vacant, <nil> is returned.
func (rf RegisterFile) GetNextTempIExclude(exc []regfile.Register) regfile.Register {
	// Use r8-28. Registers r19-28 are callee-saved.
	for i1, e1 := range rf.regi[r8:r29] {
		for _, e2 := range exc {
			if e2.Id() == e1.(*register).Id() && e2.Type() == ir.DataInteger {
				// Register already in use by neighbour.
				goto els
			}
		}
		return rf.regi[i1+r8]

	els:
	}
	return nil
}

// GetNextTempF returns the next available floating point register that hasn't been allocated yet and is
// not in the exclusion list. If no registers are vacant, <nil> is returned.
func (rf RegisterFile) GetNextTempFExclude(exc []regfile.Register) regfile.Register {
	// Use v8-31. Registers v8-15 are callee-saved.
	for i1, e1 := range rf.regf[v8:] {
		for _, e2 := range exc {
			if e2.Id() == e1.(*register).Id() && e2.Type() == ir.DataFloat {
				// Register already in use by neighbour.
				goto els
			}
		}
		return rf.regf[i1+v8]
	els:
	}
	return nil
}

// FreeI frees integer register with index i.
func (rf RegisterFile) FreeI(i int) {
	if i < 0 || i >= len(rf.regi) {
		return
	}
	rf.regi[i].(*register).use = 0
}

// FreeF frees integer register with index i.
func (rf RegisterFile) FreeF(i int) {
	if i < 0 || i >= len(rf.regf) {
		return
	}
	rf.regf[i].(*register).use = 0
}

// SP returns a pointer to the register file's stack pointer.
func (rf RegisterFile) SP() regfile.Register {
	return rf.regi[sp]
}

// FP returns a pointer to the register file's frame pointer.
func (rf RegisterFile) FP() regfile.Register {
	return rf.regi[fp]
}

// LR returns a pointer to the register file's link register.
func (rf RegisterFile) LR() regfile.Register {
	return rf.regi[lr]
}

// Ki returns the number of usable temporary integer registers.
func (rf RegisterFile) Ki() int {
	return 20
}

// Kf returns the number of usable temporary floating point registers.
func (rf RegisterFile) Kf() int {
	return 24
}
