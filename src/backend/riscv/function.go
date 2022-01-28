package riscv

import (
	"fmt"
	"vslc/src/ir"
	"vslc/src/util"
)

// genFunction generates a function. An error is returned if something went wrong.
func genFunction(n *ir.Node, wr *util.Writer, st *util.Stack, rf *registerFile) error {
	name := n.Children[0].Data.(string)
	//returnType := n.Children[1].Data.(string)
	//params := n.Children[2].Children // slice of typed variable lists.
	body := n.Children[3] // Function body.

	fun := n.Entry // Symbol table entry for function.

	// Calculate stack required by called function.
	np := 0
	if fun.Nparams > 8 {
		np += fun.Nparams - 8
	}
	N := (np + fun.Nlocals) << 2 // Number of bytes of data elements required by function.
	if res := N % stackAlign; res != 0 {
		N += res // Adjust stack alignment.
	}

	// Allocate stack.
	wr.Label(name)
	wr.Ins2imm("addi", regi[sp], regi[sp], -(N + 16))            // Grow stack downwards.
	wr.Write("\tsw\t%s, %d(%s)\n", regi[ra], (N+16)-4, regi[sp]) // Store old sp to return address.
	wr.Write("\tsw\t%s, %d(%s)\n", regi[fp], (N+16)-8, regi[sp]) // Store old sp to frame pointer.
	wr.Ins2imm("addi", regi[fp], regi[sp], N+16)                 // Set fp to be frame pointer.

	// Check for floating point

	// From: https://riscv.org/wp-content/uploads/2015/01/riscv-calling.pdf
	// The stack pointer sp points to the first argument not passed in a register.

	// Generate function body.
	if err := genAsm(body, fun, wr, st, rf); err != nil {
		return err
	}

	// Deallocate stack.
	wr.Write("\tlw\t%s, %d(%s)\n", regi[ra], 12, regi[sp])
	wr.Write("\tlw\t%s, %d(%s)\n", regi[fp], 8, regi[sp])
	wr.Ins2imm("addi", regi[sp], regi[sp], N+16)
	wr.Write("\tret\n")

	return nil
}

// genFunctionCall generates a call to a function and returns a pointer to the register where the result is put.
// An error is returned if the function call is not valid.
func genFunctionCall(n *ir.Node, f *ir.Symbol, wr *util.Writer, st *util.Stack, rf *registerFile) (*register, error) {
	// At this point we know that this call is valid and parameters match, because it has been previously verified.
	name := n.Children[0].Data.(string)
	args := n.Children[1].Children[0].Children // Arguments.
	fun, _ := ir.GetEntry(name, st)            // Symbol table entry of called function.

	// Put arguments in registers and on stack.
	// From: https://riscv.org/wp-content/uploads/2015/01/riscv-calling.pdf
	// The stack pointer sp points to the first argument not passed in a register.
	//pi := 0
	//pf := 0
	idx := 0
	for _, e1 := range args {
		switch e1.Typ {
		case ir.IDENTIFIER_DATA:
		case ir.FLOAT_DATA:
		case ir.INTEGER_DATA:
		case ir.EXPRESSION:
		}
		fmt.Println(e1.String())
	}

	// Save volatile register t0-t6 and ft0-ft11 to stack before calling function.
	wr.Ins2imm("addi", regi[sp], regi[sp], -76) // Size of registers t0-t6 and ft0-ft11.

	// Save t0-t2 to stack.
	idx = 0
	for i1 := t0; i1 <= t2; i1++ {
		wr.Ins2("sw", regi[i1], fmt.Sprintf("%d(%s)", idx, regi[sp]))
		idx -= wordSize
	}
	// Save t3-t6 to stack.
	for i1 := t3; i1 <= t6; i1++ {
		wr.Ins2("sw", regi[i1], fmt.Sprintf("%d(%s)", idx, regi[sp]))
		idx -= wordSize
	}
	// Save ft0-ft7 to stack.
	for i1 := ft0; i1 <= ft7; i1++ {
		wr.Ins2("fsw", regf[i1], fmt.Sprintf("%d(%s)", idx, regi[sp]))
		idx -= wordSize
	}
	// Save ft8-ft11 to stack.
	for i1 := ft8; i1 <= ft11; i1++ {
		wr.Ins2("fsw", regf[i1], fmt.Sprintf("%d(%s)", idx, regi[sp]))
		idx -= wordSize
	}

	// Jump and link.
	wr.Ins1("call", name)

	// Restore saved  temporary registers.
	// Load t0-t2 from stack.
	idx = 0
	for i1 := t0; i1 <= t2; i1++ {
		wr.Ins2("lw", regi[i1], fmt.Sprintf("%d(%s)", idx, regi[sp]))
		idx -= wordSize
	}
	// Load t3-t6 from stack.
	for i1 := t3; i1 <= t6; i1++ {
		wr.Ins2("lw", regi[i1], fmt.Sprintf("%d(%s)", idx, regi[sp]))
		idx -= wordSize
	}
	// Load ft0-ft7 from stack.
	for i1 := ft0; i1 <= ft7; i1++ {
		wr.Ins2("flw", regf[i1], fmt.Sprintf("%d(%s)", idx, regi[sp]))
		idx -= wordSize
	}
	// Load ft8-ft11 from stack.
	for i1 := ft8; i1 <= ft11; i1++ {
		wr.Ins2("flw", regf[i1], fmt.Sprintf("%d(%s)", idx, regi[sp]))
		idx -= wordSize
	}

	// Restore stack pointer.
	wr.Ins2imm("addi", regi[sp], regi[sp], 76) // Size of registers t0-t6 and ft0-ft11.

	// Return a0 or f0.
	if fun.DataTyp == ir.DataInteger {
		return &(rf.i[a0]), nil
	} else {
		return &(rf.f[fa0]), nil
	}
}
