// Package arm provides means to generate aarch64 assembly code from the intermediate syntax tree representation.
package arm

import (
	"errors"
	"fmt"
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

const labelFloat = "_LFLOAT_"
const labelString = "_STR_"
const labelMain = "main"                         // String literal of name of main function as defined in the output assembler.
const labelPrintfInt = "_printf_fmt_int"         // Used by printf to print integers.
const labelPrintfFloat = "_printf_fmt_float"     // Used by printf to print floats.
const labelPrintfString = "_printf_fmt_string"   // Used by printf to print strings.
const labelPrintfNewline = "_printf_fmt_newline" // Used by printf to print strings.

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

				for _, e1 := range m.Functions()[start:end] {
					if err := genFunction(e1, &wr); err != nil {
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
	if err := genMain(rf, m, callee, &wr); err != nil {
		return err
	}

	// Generate global data.
	wr.Write("\n\t.data\n")
	for _, e1 := range m.Globals() {
		wr.Label(e1.Name())
		// Write globals with initial values 0. VSL doesn't support variable initialisation on declaration.
		wr.Write("\t.xword\t0x0\n")
	}

	// Generate string data.
	for _, e1 := range m.Strings() {
		wr.Label(e1.Name())
		wr.Write("\t.asciz\t%q\n", e1.Value())
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

// genMain generates an implicit main function that checks input command-line arguments and calls the function callee.
// After the function callee returns the main function exits the program with the return value of the call to callee.
// If the return value of callee is a floating point value, the value is cast to integer.
func genMain(rf RegisterFile, m *lir.Module, callee *lir.Function, wr *util.Writer) error {
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
	wr.Write("\tcmp\t%s, #%d\n", rf.regi[r0].String(), len(callee.Params())+1)              // First argument is application path.
	wr.Write("\tb.eq\t%s\n", largcok)

	// argc is not ok.
	errstr := callee.CreateGlobalString(fmt.Sprintf("Argument error: expected %d arguments, got %%d\n", len(callee.Params())))

	// Load argc - 1 into r1. It's safe to overwrite r1, because we're not going to use it.
	wr.Write("\tsub\t%s, %s, #%d\n", rf.regi[r1].String(), rf.regi[r0].String(), 1)

	// Load format string and call printf.
	wr.Write("\tadrp\t%s, %s\n", rf.regi[r0].String(), errstr.Name())
	wr.Write("\tadd\t%s, %s, :lo12:%s\n", rf.regi[r0].String(), rf.regi[r0].String(), errstr.Name())
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
	var argvreg regfile.Register
	if ni > 0 {
		// Use r8 instead of r0 for argv pointer. r0 is already assigned a parameter.
		argvreg = rf.GetI(r8)
	} else {
		argvreg = rf.GetI(r0)
	}
	for i1, e1 := range callee.Params() {
		// Move char pointer to r0. Increment r1 to next string.
		wr.Write("\tldr\t%s, [%s, #%d]\n", argvreg.String(), rf.SP().String(), sa-wordSize<<1) // Load argv into r8.

		if e1.DataType() == i {
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
				wr.Write("\tstr\t%s, [%s, #%d]\n", rf.regi[r0].String(), rf.LR(), -(spill + wordSize*i1))
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
			loadFloatToRegister(flabel, rf.GetF(v1), wr) // Move 0.0 into v1.
			wr.Write("\tfcmp\t%s, %s\n", rf.regf[v0].String(), rf.regf[v1].String())
			wr.Write("\tmov\t%s, #%d\n", rf.regi[r1].String(), i1+1) // Hack the printf by moving parameter index to r1. Wastes one cycle.
			wr.Write("\tb.eq\t%s\n", largverr)                       // Got error from atof, branch to argverror.
			if fi < paramReg {
				// Store parameter in register.
				wr.Write("\tmov\t%s, %s\n", rf.regf[v0].String(), rf.regf[fi].String())
			} else {
				// Store parameter on stack.
				wr.Write("\tstr\t%s, [%s, #%d]\n", rf.regf[v0].String(), rf.LR(), -(spill + wordSize*i1))
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
	idx := len(m.Strings())
	errstr = callee.CreateGlobalString(fmt.Sprintf("Argument error: argument %d is either not int or float\n", idx))

	// Load format string and call printf.
	wr.Write("\tadrp\t%s, %s%d\n", rf.regi[r0].String(), errstr.Name())
	wr.Write("\tadd\t%s, %s, :lo12:%s%d\n", rf.regi[r0].String(), rf.regi[r0].String(), errstr.Name())
	wr.Write("\tbl\tprintf\n")

	// Set return code and return.
	wr.Write("\tmov\t%s, #%d\n", rf.regi[r0].String(), 1)
	wr.Write("\tadd\t%s, %s, #%d\n", rf.SP().String(), rf.SP().String(), sa)
	wr.Write("\tret\n")

	// Go here when ready to call callee.
	wr.Label(lcall)
	wr.Write("\tbl\t%s\n", callee.Name())

	// Move float result from v0 to r0 if necessary.
	if callee.DataType() == f {
		wr.Write("\tfcvts\t%s, %s\n", rf.regf[v0].String(), rf.regi[r0].String())
	}

	// De-allocate stack and return, result from callee is already in r0.
	wr.Write("\tadd\t%s, %s, #%d\n", rf.SP().String(), rf.SP().String(), sa)
	wr.Write("\tret\n")
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
func loadFloatToRegister(label string, r regfile.Register, wr *util.Writer) {
	wr.Write("\tadrp\t%s, %s\n", r.String(), label)
	wr.Write("\tadd\t%s, %s, :lo12:%s\n", r.String(), r.String(), label)
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
	// Exclude r28, because it may be used for register spilling.
	// TODO: Confirm the use of excluding register 28.
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
	// Use v8-31. Registers v8-15 are callee-saved.
	// Exclude v31 because of saving one register for register spilling.
	// TODO: Confirm use of v31 for register spilling.
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
