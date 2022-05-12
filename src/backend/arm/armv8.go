// Package arm provides means to generate aarch64 assembly code from the intermediate syntax tree representation.
package arm

import (
	"errors"
	"fmt"
	"math"
	"path/filepath"
	"sync"
)

import (
	"vslc/src/backend/regfile"
	"vslc/src/ir"
	"vslc/src/ir/lir"
	"vslc/src/ir/lir/types"
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

// RegisterFile defines a virtual register file during compilation time. It holds 32 integer and 32 floating point
// registers per aarch64 ABI.
type RegisterFile struct {
	regi []regfile.Register
	regf []regfile.Register
}

// ---------------------
// ----- Constants -----
// ---------------------

const labelMain = "main"          // String literal of name of main function as defined in the output assembler.
const labelConstant = "_L_CONST_" // String literal for all constants.

const (
	i = types.Int   // i indicates integer type.
	f = types.Float // f indicates floating point type.
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

const (
	a0 = r0 + iota // a0 defines argument register 0 and return value register.
	a1             // a1 defines argument register 1.
	a2             // a2 defines argument register 2.
	a3             // a3 defines argument register 3.
	a4             // a4 defines argument register 4.
	a5             // a5 defines argument register 5.
	a6             // a6 defines argument register 6.
	a7             // a7 defines argument register 7.
)

const (
	load  = "ldr"
	store = "str"
)

// -------------------
// ----- Globals -----
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
	"fp",
	"lr",
	"sp",
}

// regf defines print friendly string representations of the floating point registers.
var regf = [...]string{
	"d0",
	"d1",
	"d2",
	"d3",
	"d4",
	"d5",
	"d6",
	"d7",
	"d8",
	"d9",
	"d10",
	"d11",
	"d12",
	"d13",
	"d14",
	"d15",
	"d16",
	"d17",
	"d18",
	"d19",
	"d20",
	"d21",
	"d22",
	"d23",
	"d24",
	"d25",
	"d26",
	"d27",
	"d28",
	"d29",
	"d30",
}

// wordSize defines the word size of the aarch64 architecture to generate.
var wordSize = wordSize64 // Default to 64-bit architecture.

// bitSize defines the bit size of the aarch64 architecture to generate.
var bitSize = bitSize64 // default to 64-bit architecture.

// wordLabel defines the size of the architecture word. xword for 64-bit, word for 32-bit.
var wordLabel = "xword"

// ---------------------
// ----- functions -----
// ---------------------

// GenArm recursively generates ARM v8 (aarch64) assembler code from the intermediate representation.
func GenArm(opt util.Options, m *lir.Module, root *ir.Node) error {
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
		l := len(m.Functions())
		if t > l {
			t = l
		}
		n := l / t   // Jobs per worker go routine.
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
				w := util.NewWriter()
				defer wg.Done()
				defer w.Close()

				for _, e1 := range m.Functions()[start:end] {
					if err := genFunction(e1, &w); err != nil {
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
		for _, e1 := range m.Functions() {
			if err := genFunction(e1, &wr); err != nil {
				return err
			}
		}
	}

	// Generate main function.
	// Find first defined function, which will be called implicitly from main.
	var callee *lir.Function
	for _, e1 := range root.Children {
		if e1.Typ == ir.FUNCTION {
			if callee = m.GetFunction(e1.Children[0].Data.(string)); callee == nil {
				return errors.New("no functions defined for module")
			}
			break
		}
	}
	rf := CreateRegisterFile()

	// Generate implicit main function for program entry.
	if err := genMain(rf, callee, &wr); err != nil {
		return err
	}
	wr.Flush()

	// Generate global data.
	wr.Write("\n\t.data\n")
	for _, e1 := range m.Globals() {
		wr.Label(e1.Name())
		// Write globals with initial values 0. VSL doesn't support variable initialisation on declaration.
		wr.Write("\t.%s\t0x0\n", wordLabel)
	}

	// Generate constant data.
	for _, e1 := range m.Constants() {
		// Only write constants that have been used. This avoids double storing small constants such as integer immediates.
		if e1.Used() {
			wr.Label(fmt.Sprintf("%s%d", labelConstant, e1.GlobalSeq()))
			if e1.DataType() == types.Int {
				wr.Write("\t.%s\t0x%x\t// %d\n", wordLabel, e1.Value().(int), e1.Value().(int))
			} else {
				fl := math.Float64bits(e1.Value().(float64))
				wr.Write("\t.%s\t0x%x\t// %f\n", wordLabel, fl, e1.Value().(float64))
			}
		}
	}

	// Generate string data.
	for _, e1 := range m.Strings() {
		wr.Label(e1.Name())
		wr.Write("\t.asciz\t%q\n", e1.Value())
	}
	return nil
}

// genMain generates an implicit main function that checks input command-line arguments and calls the function callee.
// After the function callee returns the main function exits the program with the return value of the call to callee.
// If the return value of callee is a floating point value, the value is cast to integer.
func genMain(rf RegisterFile, callee *lir.Function, wr *util.Writer) error {
	wr.Write("\n")
	wr.Label(labelMain)

	nf, ni := 0, 0 // Number of floating point and integer parameters respectively.
	for _, e1 := range callee.Params() {
		if e1.DataType() == i {
			ni++
		} else {
			nf++
		}
	}

	// Set up stack.
	sa := 4 + ni + nf // FP, LR, argc and argv plus all arguments required by callee.

	spill := 0 // Needed for adjusting where to start storing arguments, such that last argument hits FP of callee.
	sa *= wordSize
	res := sa % stackAlign
	if res != 0 {
		spill += stackAlign - res
		sa += res
	}

	// Stack from top to bottom before PASSING ARGUMENTS over stack.
	// This is just for storing parsed arguments before calling VSL callee function.
	//
	// TOP
	// <--- FP
	// LR
	// FP
	// argc
	// **argv
	// [spill]
	// argv[1]
	// argv[2]
	// argv[3]
	// ...
	// argv[argc-1]
	// <--- SP
	//
	// BOTTOM

	fpOffsetArgc := wordSize * 3 // Offset of argc on stack from FP.
	fpOffsetArgv := wordSize << 2
	wr.Write("\tsub\t%s, %s, #%d\n", rf.SP().String(), rf.SP().String(), sa) // Adjust SP.
	wr.Write("\tstp\t%s, %s, [%s, #%d]\n",
		rf.FP().String(), rf.LR().String(), rf.SP().String(), sa-(wordSize<<1)) // Store FP and LR on top of stack.
	wr.Write("\tadd\t%s, %s, #%d\n", rf.FP().String(), rf.SP().String(), sa)                  // Set new FP to old SP.
	wr.Write("\tstr\t%s, [%s, #%d]\n", rf.regi[r0].String(), rf.FP().String(), -fpOffsetArgc) // argc.
	wr.Write("\tstr\t%s, [%s, #%d]\n", rf.regi[r1].String(), rf.FP().String(), -fpOffsetArgv) // argv.

	// Jump labels for error checking.
	largcok := "_L_argc_ok"     // Jump to label if argc matches parameter count of callee.
	largverr := "_L_argv_error" // Jump to label if parameter is not integer or float.
	lcall := "_L_call"          // Jump to label when all parameters are ok.

	// Check parameter count and argc.
	wr.Write("\tldr\t%s, [%s, #%d]\n", rf.GetI(r1).String(), rf.FP().String(), -fpOffsetArgc) // This is bloated, but it's idiomatic to load argc from the stack.
	wr.Write("\tsub\t%s, %s, #%d\n", rf.GetI(r1).String(), rf.GetI(r1).String(), 1)
	wr.Write("\tcmp\t%s, #%d\n", rf.GetI(r1).String(), len(callee.Params())) // First argument is application path.
	wr.Write("\tb.eq\t%s\n", largcok)

	// argc is not ok.
	var errstr *lir.String
	if len(callee.Params()) == 1 {
		errstr = callee.CreateGlobalString("Argument error: expected 1 argument, got %d\n")
	} else {
		errstr = callee.CreateGlobalString(fmt.Sprintf("Argument error: expected %d arguments, got %%d\n", len(callee.Params())))
	}

	// Load format string and call printf.
	wr.Write("\tadrp\t%s, %s\n", rf.GetI(r0).String(), errstr.Name())
	wr.Write("\tadd\t%s, %s, :lo12:%s\n", rf.GetI(r0).String(), rf.GetI(r0).String(), errstr.Name())
	wr.Write("\tbl\tprintf\n")

	// Set return code and return.
	wr.Write("\tmov\t%s, #%d\n", rf.GetI(r0).String(), 1)
	wr.Write("\tldp\t%s, %s, [%s, #%d]\n",
		rf.FP().String(), rf.LR().String(), rf.SP().String(), sa-(wordSize<<1)) // Restore FP and LR before returning.
	wr.Write("\tadd\t%s, %s, #%d\n", rf.SP().String(), rf.SP().String(), sa)
	wr.Write("\tret\n")

	// argc is ok.
	wr.Label(largcok)

	if len(callee.Params()) > 0 {
		ii := 0 // Number of integer arguments provided.
		fi := 0 // Number of floating point arguments provided.

		// Generate arguments. Parse and store on stack to avoid overwriting during atoi/atof calls.
		// Retrieve when calling VSL callee function.
		for i1, e1 := range callee.Params() {
			// Move argv pointer into register x8.
			wr.Write("\tldr\t%s, [%s, #%d]\t// Load argv\n",
				rf.GetI(r8).String(), rf.FP().String(), -fpOffsetArgv)

			// Put the i'th element of argv into x0 for atoi and/or atof.
			wr.Write("\tldr\t%s, [%s, #%d]\t// Load argv[%d]\n",
				rf.GetI(r0).String(), rf.GetI(r8).String(), wordSize*(i1+1), i1+1)

			// Save current argv index in x19 for error reporting.
			wr.Write("\tmov\t%s, #%d\n", rf.GetI(r19).String(), i1+1)

			if e1.DataType() == types.Int {
				// Parse argv[i1+1] as int using atoi.
				wr.Write("\tbl\tatoi\n")

				// Verify that argument was an integer != 0.
				wr.Write("\tcbz\tw0, %s\n", largverr) // atoi returns 32-bit int in w0.

				// Store on stack for later.
				wr.Write("\tstr\t%s, [%s, #%d]\n",
					rf.GetI(r0), rf.FP().String(), -fpOffsetArgv-spill-wordSize*(1+i1)) // Adjust for spill.
				ii++
			} else {
				// Parse argv[i1+1] as float using atof.

				// Call atof.
				wr.Write("\tbl\tatof\n")

				// Verify that argument was a float != 0.0.
				wr.Write("\tfcmp\t%s, #0.0\n", rf.GetF(v0).String())
				wr.Write("\tb.eq\t%s\n", largverr)

				// Store on stack for later.
				wr.Write("\tstr\t%s, [%s, #%d]\n",
					rf.GetF(v0), rf.FP().String(), -fpOffsetArgv-spill-wordSize*(1+i1)) // Adsjust for spill.
				fi++
			}
		}

		// Generate arguments from back to front, such that the first argument can be put into x0/d0 without collision.
		//for i1 := len(callee.Params()) - 1; i1 >= 0; i1-- {
		//	// Move argv pointer into register x8.
		//	wr.Write("\tldr\t%s, [%s, #%d]\n",
		//		rf.GetI(r8).String(), rf.FP().String(), -fpOffsetArgv)
		//
		//	wr.Write("\tmov\t%s, #%d\n", rf.GetI(r19).String(), len(callee.Params())-ii-fi) // Save current argv index in x19 for error reporting.
		//	e1 := callee.Params()[i1]
		//
		//	if e1.DataType() == types.Int {
		//		// Parameter is integer. Used atoi to parse argument.
		//
		//		// Put the i'th element of argv into x0 for atoi and/or atof.
		//		wr.Write("\tldr\t%s, [%s, #%d]\t// Load argv[%d]\n",
		//			rf.GetI(r0).String(), rf.GetI(r8).String(), wordSize*(i1+1), len(callee.Params())-ii-fi)
		//
		//		// Call atoi.
		//		wr.Write("\tbl\tatoi\n")
		//
		//		// Verify that argument was an integer != 0.
		//		wr.Write("\tcbz\tw0, %s\n", largverr) // atoi returns 32-bit int in w0.
		//
		//		// Argument is good: move result integer to correct register or pass on stack.
		//		if ii < paramReg {
		//			// Pass in register.
		//			dst := rf.GetI(ni%paramReg - ii - 1)
		//			//if dst.Id() != r0 {
		//			// Argument is already in correct register.
		//			wr.Write("\tmov\t%s, %s\n", dst.String(), rf.GetI(r0).String())
		//			//}
		//		} else {
		//			// Pass on stack.
		//			wr.Write("\tstr\t%s, [%s, #%d]\n", rf.GetI(r0).String(), rf.SP().String(), (ii-paramReg)*wordSize)
		//		}
		//		ii++
		//	} else {
		//		// Parameter is floating point type. Parse argument as float using atof.
		//
		//		// Put the i'th element of argv into x0 for atof.
		//		wr.Write("\tldr\t%s, [%s, #%d]\n", rf.GetI(r0).String(), rf.GetI(r8).String(), wordSize*i1)
		//
		//		// Call atof.
		//		wr.Write("\tbl\tatof\n")
		//
		//		// Verify that argument was a float != 0.0.
		//		wr.Write("\tfcmp\t%s, #0.0\n", rf.GetF(v0).String(), largverr)
		//		wr.Write("\tb.eq\t%s\n", largverr)
		//
		//		// Pass floating point argument either in register or on stack.
		//		if fi < paramReg {
		//			// Pass in register.
		//			dst := rf.GetF(fi%paramReg - fi - 1)
		//			//if dst.Id() != v0 {
		//			// Argument is already in correct register.
		//			wr.Write("\tmov\t%s, %s\n", dst.String(), rf.GetF(v0).String())
		//			//}
		//		} else {
		//			// Pass on stack.
		//			wr.Write("\tstr\t%s, [%s, #%d]\n", rf.GetI(v0).String(), rf.SP().String(), (fi-paramReg)*wordSize)
		//		}
		//		fi++
		//	}
		//}
	}

	// When done with parameters, cause program to jump to call function under the argv error handling logic.
	wr.Write("\tb\t%s\n", lcall)

	// Go here when ready to call callee.
	wr.Label(lcall)

	idx := 0 // Number of int arguments moved.
	fdx := 0 // Number of float arguments moved.

	// Check if we need to pass arguments over stack.
	argsa := sa // Argument passing stack.
	if ni > paramReg {
		argsa += (ni - paramReg) * wordSize
	}
	if nf > paramReg {
		argsa += (nf - paramReg) * wordSize
	}
	res = argsa % stackAlign
	if res != 0 {
		argsa += stackAlign - res
	}

	// Need to adjust stack.
	if argsa > sa {
		wr.Write("\tsub\t%s, %s, #%d\n", rf.SP().String(), rf.SP().String(), argsa-sa)
		sa = argsa

		// Stack from top to bottom when passing arguments over stack.
		//
		// TOP
		// <--- FP
		// LR
		// FP
		// argc
		// **argv
		// [spill]
		// argv[1]
		// argv[2]
		// argv[3]
		// ...
		// argv[argc-1]
		// ------------- only needed if more arguments than argument registers --------------
		// [spill]
		// parsed argument 0
		// parsed argument 1
		// parsed argument 2
		// ...
		// parsed argument n
		// ------------- only needed if more arguments than argument registers --------------
		// <--- SP
		//
		// BOTTOM
	}

	// Load arguments from stack into registers or push to stack.
	for i1, e1 := range callee.Params() {
		if e1.DataType() == types.Int {
			if idx < paramReg {
				wr.Write("\tldr\t%s, [%s, #%d]\t// Load parsed argv[%d] into register %s\n",
					rf.GetI(idx).String(), rf.FP().String(), -fpOffsetArgv-spill-wordSize*(i1+1), i1+1, rf.GetI(idx).String())
			} else {
				// Store to stack.
				sdx := 1 + i1 - paramReg
				tmp := rf.GetI(r20) // Used r20 as temporary register.
				wr.Write("\tldr\t%s, [%s, #%d]\n",
					tmp.String(), rf.FP().String(), -fpOffsetArgv-spill-wordSize*(idx+1))
				wr.Write("\tstr\t%s, [%s, #%d]\n", tmp.String(), rf.SP().String(), wordSize*sdx)
			}
			idx++
		} else {
			if fdx < paramReg {
				wr.Write("\tldr\t%s, [%s, #%d]\t// Load parsed argv[%d] into register %s\n",
					rf.GetF(fdx).String(), rf.FP().String(), -fpOffsetArgv-spill-wordSize*(i1+1), i1+1, rf.GetF(fdx).String())
			} else {
				// Store to stack.
				sdx := 1 + i1 - paramReg
				tmp := rf.GetF(v20) // Used r20 as temporary register.
				wr.Write("\tldr\t%s, [%s, #%d]\n",
					tmp.String(), rf.FP().String(), -fpOffsetArgv-spill-wordSize*(idx+1))
				wr.Write("\tstr\t%s, [%s, #%d]\n", tmp.String(), rf.SP().String(), wordSize*sdx)
			}
			fdx++
		}
	}

	// Call VSL callee function.
	wr.Write("\tbl\t%s\n", callee.Name())

	// Move float result from v0 to r0 if necessary.
	if callee.DataType() == f {
		wr.Write("\tfcvtns\t%s, %s\n", rf.regf[v0].String(), rf.regi[r0].String()) // Round to nearest.
	}

	// De-allocate stack and return, result from callee is already in r0.
	wr.Write("\tldp\t%s, %s, [%s, #%d]\n",
		rf.FP().String(), rf.LR().String(), rf.SP().String(), sa-(wordSize<<1)) // Restore FP and LR before returning.
	wr.Write("\tadd\t%s, %s, #%d\n", rf.SP().String(), rf.SP().String(), sa)
	wr.Write("\tret\n")

	if len(callee.Params()) > 0 {

		// argv errors jump here.
		wr.Label(largverr)
		errstr = callee.CreateGlobalString("Argument error: argument %ld is neither int nor float\n")

		// Load format string and call printf.
		wr.Write("\tadrp\t%s, %s\n", rf.regi[r0].String(), errstr.Name())
		wr.Write("\tadd\t%s, %s, :lo12:%s\n", rf.regi[r0].String(), rf.regi[r0].String(), errstr.Name())
		wr.Write("\tmov\t%s, %s\n", rf.GetI(r1).String(), rf.GetI(r19).String()) // Move saved argument index into x1.
		wr.Write("\tbl\tprintf\n")

		// Set return code and return.
		wr.Write("\tmov\t%s, #%d\n", rf.GetI(r0).String(), 1)
		wr.Write("\tldp\t%s, %s, [%s, #%d]\n",
			rf.FP().String(), rf.LR().String(), rf.SP().String(), sa-(wordSize<<1)) // Restore FP and LR before returning.
		wr.Write("\tadd\t%s, %s, #%d\n", rf.SP().String(), rf.SP().String(), sa)
		wr.Write("\tret\n")
	}
	return nil
}

func CreateRegisterFile() RegisterFile {
	rf := RegisterFile{
		regi: make([]regfile.Register, 32),
		regf: make([]regfile.Register, 32),
	}

	// Initiate registers.
	for i1 := range rf.regi {
		rf.regi[i1] = &register{
			typ:  int(types.Int),
			size: bitSize,
			idx:  i1,
		}
		rf.regf[i1] = &register{
			typ:  int(types.Float),
			size: bitSize,
			idx:  i1,
		}
	}
	return rf
}

// ----------------------------
// ----- Register methods -----
// ----------------------------

// String returns the assembler string of the register.
func (r register) String() string {
	if r.typ == int(i) {
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
	// Used r8-28. Registers r19-28 are callee-saved.
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
	// Used v8-31. Registers v8-15 are callee-saved.
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
	// Used r8-28. Registers r19-28 are callee-saved.
	// Exclude r28, because it may be used for register spilling or other temporaries.
	for i1, e1 := range rf.regi[r8:r28] {
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

// GetNextTempFExclude returns the next available floating point register that hasn't been allocated yet and is
// not in the exclusion list. If no registers are vacant, <nil> is returned.
func (rf RegisterFile) GetNextTempFExclude(exc []regfile.Register) regfile.Register {
	// Used v8-30. Registers v8-15 are callee-saved.
	// Exclude v30 because of saving one register for register spilling or other temporaries.
	for i1, e1 := range rf.regf[v8:v30] {
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
	return len(rf.regi[r8:r29])
}

// Kf returns the number of usable temporary floating point registers.
func (rf RegisterFile) Kf() int {
	return len(rf.regf[v8:v30])
}
