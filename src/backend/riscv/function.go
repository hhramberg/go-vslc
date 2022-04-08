package riscv

import (
	"fmt"
	"vslc/src/ir"
	"vslc/src/util"
)

// genFunction generates a function. An error is returned if something went wrong.
func genFunction(n *ir.Node, wr *util.Writer, st, ls *util.Stack, rf *RegisterFile) error {
	name := n.Children[0].Data.(string)
	//returnType := n.Children[1].Data.(string)
	params := n.Children[2].Children // slice of typed variable lists.
	body := n.Children[3]            // Function body.

	fun := n.Entry // Symbol table entry for function.

	// Calculate stack required by called function.
	np := 0
	if fun.Nparams > 8 {
		np += fun.Nparams - 8
	}
	N := (fun.Nparams + fun.Nlocals) * wordSize // Number of bytes of data elements required by function.
	if res := N % stackAlign; res != 0 {
		N += res // Adjust stack alignment.
	}

	// Allocate stack.
	wr.Write("\n")
	wr.Label(name)
	wr.Ins2imm("addi", regi[sp], regi[sp], -(N + (wordSize << 1)))        // Grow stack downwards.
	wr.Write("\t%s\t%s, %d(%s)\n", store, regi[ra], N+wordSize, regi[sp]) // Store old return address.
	wr.Write("\t%s\t%s, %d(%s)\n", store, regi[fp], N, regi[sp])          // Store old frame pointer.
	wr.Ins2imm("addi", regi[fp], regi[sp], N+(wordSize<<1))               // Set fp to be frame pointer of this function's stack.

	// Put arguments from registers on stack.
	rp := fun.Nparams
	if rp > argsReg {
		rp = argsReg
	}
	for i1 := 0; i1 < rp; {
		for _, e2 := range params {
			if i1 >= rp {
				break
			}
			for _, e3 := range e2.Children {
				if i1 >= rp {
					break
				}
				idx := wordSize << 1 // First two words are ra and fp of caller.
				idx += wordSize * (i1 + 1)
				param, _ := fun.Locals.Get(e3.Data.(string)) // Safe; we know it exists, else it would have called error in validation.
				if param.DataTyp == ir.DataInteger {
					wr.Write("\t%s\t%s, -%d(%s)\n", store, regi[a0+i1], idx, regi[fp])
				} else {
					wr.Write("\tf%s\t%s, -%d(%s)\n", store, regf[fa0+i1], idx, regi[fp])
				}
				i1++
			}
		}
	}

	// Arguments with sequence numbers >= argsReg are already on stack.

	// Local definitions are allocated space on stack, but they have not yet been assigned a value.
	// VSL doesn't support assignment on declaration.

	// Generate function body.
	if err := genAsm(body, fun, wr, st, ls, rf); err != nil {
		return err
	}

	// Deallocate stack.
	wr.Write("\t%s\t%s, %d(%s)\n", load, regi[ra], N+wordSize, regi[sp])
	wr.Write("\t%s\t%s, %d(%s)\n", load, regi[fp], N, regi[sp])
	wr.Ins2imm("addi", regi[sp], regi[sp], N+(wordSize<<1))
	wr.Write("\tret\n")

	return nil
}

// genFunctionCall generates a call to a function and returns a pointer to the register where the result is put.
// An error is returned if the function call is not valid.
func genFunctionCall(n *ir.Node, f *ir.Symbol, wr *util.Writer, st *util.Stack, rf *RegisterFile) (*register, error) {
	// At this point we know that this call is valid and parameters match, because it has been previously verified.
	name := n.Children[0].Data.(string)
	args := n.Children[1].Children[0].Children // Arguments.
	fun, _ := ir.GetEntry(name, st)            // Symbol table entry of function to call.

	wr.Write("# calling function %q\n", name)
	// Save volatile register t0-t6 and ft0-ft11 to stack before calling function.
	adj := 19 * wordSize
	if adj%stackAlign != 0 {
		// Align the stack adjustment to comply with 16-byte aligned stack.
		adj += stackAlign - (adj % stackAlign)
	}
	wr.Ins2imm("addi", regi[sp], regi[sp], adj) // Size of registers t0-t6 and ft0-ft11.

	// Save t0-t2 to stack.
	idx := 0
	for i1 := t0; i1 <= t2; i1++ {
		wr.Ins2(store, regi[i1], fmt.Sprintf("%d(%s)", idx, regi[sp]))
		idx -= wordSize
	}
	// Save t3-t6 to stack.
	for i1 := t3; i1 <= t6; i1++ {
		wr.Ins2(store, regi[i1], fmt.Sprintf("%d(%s)", idx, regi[sp]))
		idx -= wordSize
	}
	// Save ft0-ft7 to stack.
	for i1 := ft0; i1 <= ft7; i1++ {
		wr.Ins2(fmt.Sprintf("f%s", store), regf[i1], fmt.Sprintf("%d(%s)", idx, regi[sp]))
		idx -= wordSize
	}
	// Save ft8-ft11 to stack.
	for i1 := ft8; i1 <= ft11; i1++ {
		wr.Ins2(fmt.Sprintf("f%s", store), regf[i1], fmt.Sprintf("%d(%s)", idx, regi[sp]))
		idx -= wordSize
	}

	// Put arguments on stack.
	if fun.Nparams > argsReg {
		// Make room in stack for arguments.
		m := fun.Nparams % argsReg
		adj := m * wordSize
		res := (m * wordSize) % stackAlign
		if res != 0 {
			adj += res
		}
		wr.Ins2imm("addi", regi[sp], regi[sp], -adj)
	}

	for i1, e1 := range args {
		if i1 < argsReg {
			// Put argument in register.
			switch e1.Typ {
			case ir.INTEGER_DATA:
				wr.Write("\tli\t%s, %d\n", regi[a0+i1], e1.Data.(int))
			case ir.FLOAT_DATA:
				wr.Write("\tlui\t%s, %%hi(%s)\n", regf[fa0+i1], ir.Floats.Ft[e1.Data.(int)])
				wr.Write("\tf%s\t%s, %%lo(%s)(%s)\n", load, regf[fa0+i1], ir.Floats.Ft[e1.Data.(int)], regf[fa0+i1])
			case ir.IDENTIFIER_DATA:
				// TODO: Could edit loadIdentifierToReg to specify destination register on call.
				reg := rf.loadIdentifierToReg(e1.Data.(string), f, wr, st)
				if reg.typ == integer {
					wr.Ins2("mv", regi[a0+i1], reg.String())
				} else {
					wr.Ins2("mv", regf[fa0+i1], reg.String())
				}
			case ir.EXPRESSION:
				if reg, err := genExpression(e1, f, wr, st, rf); err != nil {
					return nil, err
				} else {
					if reg.typ == integer {
						wr.Ins2("mv", regi[a0+i1], reg.String())
					} else {
						wr.Ins2("mv", regf[fa0+i1], reg.String())
					}
				}
			}
		} else {
			// Put argument on stack.
			idx = (i1 + 1 - argsReg) * wordSize
			switch e1.Typ {
			case ir.INTEGER_DATA:
				reg := rf.lruI()
				wr.Write("\tli\t%s, %d\n", reg.String(), e1.Data.(int))
				wr.Write("\t%s\t%s, %d(%s)\n", store, reg.String(), idx, regi[sp])
			case ir.FLOAT_DATA:
				reg := rf.lruF()
				wr.Write("\tlui\t%s, %%hi(%s)\n", reg.String(), ir.Floats.Ft[e1.Data.(int)])
				wr.Write("\tf%s\t%s, %%lo(%s)(%s)\n", store, reg.String(), ir.Floats.Ft[e1.Data.(int)], regf[fa0+i1])
				wr.Write("\tf%s\t%s, %d(%s)\n", store, reg.String(), idx, regi[sp])
			case ir.IDENTIFIER_DATA:
				reg := rf.loadIdentifierToReg(e1.Data.(string), f, wr, st)
				if reg.typ == integer {
					wr.Write("\t%s\t%s, %d(%s)\n", store, reg.String(), idx, regi[sp])
				} else {
					wr.Write("\tf%s\t%s, %d(%s)\n", store, reg.String(), idx, regi[sp])
				}
			case ir.EXPRESSION:
				if reg, err := genExpression(e1, f, wr, st, rf); err != nil {
					return nil, err
				} else {
					if reg.typ == integer {
						wr.Write("\t%s\t%s, %d(%s)\n", store, reg.String(), idx, regi[sp])
					} else {
						wr.Write("\tf%s\t%s, %d(%s)\n", store, reg.String(), idx, regi[sp])
					}
				}
			}
		}
	}

	// Jump and link.
	wr.Ins1("call", name)

	// Deallocate stack for arguments.
	if fun.Nparams > argsReg {
		m := fun.Nparams % argsReg
		adj := m * wordSize
		res := (m * wordSize) % stackAlign
		if res != 0 {
			adj += res
		}
		wr.Ins2imm("addi", regi[sp], regi[sp], adj)
	}

	// Restore saved  temporary registers.
	// Load t0-t2 from stack.
	idx = 0
	for i1 := t0; i1 <= t2; i1++ {
		wr.Ins2(load, regi[i1], fmt.Sprintf("%d(%s)", idx, regi[sp]))
		idx -= wordSize
	}
	// Load t3-t6 from stack.
	for i1 := t3; i1 <= t6; i1++ {
		wr.Ins2(load, regi[i1], fmt.Sprintf("%d(%s)", idx, regi[sp]))
		idx -= wordSize
	}
	// Load ft0-ft7 from stack.
	for i1 := ft0; i1 <= ft7; i1++ {
		wr.Ins2(fmt.Sprintf("f%s", load), regf[i1], fmt.Sprintf("%d(%s)", idx, regi[sp]))
		idx -= wordSize
	}
	// Load ft8-ft11 from stack.
	for i1 := ft8; i1 <= ft11; i1++ {
		wr.Ins2(fmt.Sprintf("f%s", load), regf[i1], fmt.Sprintf("%d(%s)", idx, regi[sp]))
		idx -= wordSize
	}

	// Restore stack pointer.
	wr.Ins2imm("addi", regi[sp], regi[sp], adj) // Size of registers t0-t6 and ft0-ft11.

	// Return a0 or f0.
	if fun.DataTyp == ir.DataInteger {
		return &(rf.i[a0]), nil
	} else {
		return &(rf.f[fa0]), nil
	}
}
