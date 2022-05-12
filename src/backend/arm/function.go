package arm

import (
	"fmt"
	"vslc/src/backend/regfile"
	"vslc/src/ir/lir"
	"vslc/src/ir/lir/types"
	"vslc/src/util"
)

// ----------------------------
// ----- Type definitions -----
// ----------------------------

// ---------------------
// ----- Constants -----
// ---------------------

// -------------------
// ----- globals -----
// -------------------

// ---------------------
// ----- functions -----
// ---------------------

// genFunction generates aarch64 assembler code for an integer or floating point return type function.
//
// General steps:
//
// - Grow stack with 8 * arguments + sp and lr. Align with stack alignment.
// - Store all arguments on stack to maximise available registers.
// - Used register file LRU to assign registers.
// - Generate function body.
// - De-allocate stack.
// - Return x0 for integer functions, use v0 for floating point functions.
func genFunction(fun *lir.Function, wr *util.Writer) error {
	if len(fun.Blocks()) < 1 {
		return nil
	}
	rf := CreateRegisterFile()

	// Write function name label.
	wr.Write("\n")
	wr.Label(fun.Name())

	// Calculate new stack size.
	sa := wordSize * (len(fun.Params()) + len(fun.Locals()) + 2) // Stack adjust. Accommodate all local variables, params and FP + LR.
	spill := sa % stackAlign
	if spill != 0 {
		sa += stackAlign - spill
	}

	// Adjust stack and set stack frame pointer.
	wr.Write("\tsub\t%s, %s, #%d\n", rf.SP(), rf.SP(), sa)

	// Save old frame pointer and link register.
	wr.Write("\tstp\t%s, %s, [%s, #%d]\n", rf.FP(), rf.LR(), rf.SP(), sa-(wordSize<<1))

	// Set frame pointer to old stack  pointer.
	wr.Write("\tadd\t%s, %s, #%d\n", rf.FP(), rf.SP(), sa)

	ii := 0 // Number of integer parameters.
	fi := 0 // Number of float parameters.

	// Put arguments on stack.
	offset := -(wordSize * 3) // Offset by 3: 2 for skipping old SP and LR, one to align with current word.
	for i1, e1 := range fun.Params() {
		if e1.DataType() == i {
			// Integer parameter.
			if ii > paramReg {
				// Load from stack, store on stack. Reuse x0, because argument passed in x0 is stored on stack by this point.
				wr.Write("\tldr\t%s, [%s, #%d]\n", regi[r0], rf.FP(), wordSize*i1)
				wr.Write("\tstr\t%s, [%s, #%d]\n", regi[r0], rf.FP(), offset)
			} else {
				// Store directly on stack from register.
				wr.Write("\tstr\t%s, [%s, #%d]\n", regi[r0+ii], rf.FP(), offset)
			}
			ii++
		} else {
			// Float parameter.
			if fi > paramReg {
				// Load from stack, store on stack. Reuse v0, because argument passed in v0 is stored on stack by this point.
				wr.Write("\tldr\t%s, [%s, #%d]\n", rf.GetF(v0), rf.FP(), wordSize*i1)
				wr.Write("\tstr\t%s, [%s, #%d]\n", rf.GetF(v0), rf.FP(), offset)
			} else {
				// Store directly on stack from register.
				wr.Write("\tstr\t%s, [%s, #%d]\n", rf.GetF(v0+fi), rf.FP(), offset)
			}
			fi++
		}
		offset -= wordSize
	}

	ls := util.Stack{}

	// Generate function body.
	for _, e1 := range fun.Blocks() {
		// Write label for basic block.
		wr.Label(e1.Name())
		for _, e2 := range e1.Instructions() {
			switch e2.Type() {
			case types.DataInstruction:
				if e2.DataType() == types.VaList {
					// VaList is handled already by genExpression.
					break
				}
				if err := genExpression(e2.(*lir.DataInstruction), wr); err != nil {
					return err
				}
			case types.LoadInstruction:
				dst := e2.GetHW().(*lir.LiveNode).Reg.(regfile.Register)
				if e2.DataType() == types.String {
					wr.Write("\tadrp\t%s, %s\n",
						dst.String(), e2.Operand1().Name())
					wr.Write("\tadd\t%s, %s, :lo12:%s\n", dst.String(), dst.String(), e2.Operand1().Name())
					break
				}
				switch e2.Operand1().Type() {
				case types.DeclareInstruction:
					// Add 3 to offset: 1 to align for bottom-down, 2 for skipping stack saved SP and LR.
					src := e2.Operand1().(*lir.DeclareInstruction)
					wr.Write("\t%s\t%s, [%s, #%d]\n",
						load, dst.String(),
						rf.FP(), -wordSize*(src.Seq()+3+len(fun.Params()))) // Locals are stored after parameters.
				case types.Param:
					// Add 3 to offset: 1 to align for bottom-down, 2 for skipping stack saved SP and LR.
					src := e2.Operand1().(*lir.Param)
					wr.Write("\t%s\t%s, [%s, #%d]\n",
						load, dst.String(),
						rf.FP(), -wordSize*(src.Id()+3)) // Params go first on stack.
				case types.Global:
					src := e2.Operand1().(*lir.Global)

					// Used x0 for storing the temporary value that is &GLOBAL_VARIABLE. Load cannot happen after return.
					wr.Write("\tadrp\t%s, %s\n", rf.GetI(r0).String(), src.Name())
					wr.Write("\t%s\t%s, [%s, :lo12:%s]\n",
						load, dst.String(), rf.GetI(r0).String(), src.Name())
				default:
					panic(fmt.Sprintf("compiler error: unexpected load source type %s", e2.Operand1().Type().String()))
				}
			case types.StoreInstruction:
				src := e2.Operand1().GetHW().(*lir.LiveNode).Reg.(regfile.Register)
				switch e2.Operand2().Type() {
				case types.DeclareInstruction:
					// Add 3 to offset: 1 to align for bottom-down, 2 for skipping stack saved SP and LR.
					dst := e2.Operand2().(*lir.DeclareInstruction)
					wr.Write("\t%s\t%s, [%s, #%d]\n",
						store, src.String(),
						rf.FP(), -wordSize*(dst.Seq()+3+len(fun.Params()))) // Locals are stored after parameters.
				case types.Param:
					// Add 3 to offset: 1 to align for bottom-down, 2 for skipping stack saved SP and LR.
					dst := e2.Operand2().(*lir.Param)
					wr.Write("\t%s\t%s, [%s, #%d]\n",
						store, src.String(),
						rf.FP(), -wordSize*(dst.Id()+3)) // Params go first on stack.
				case types.Global:
					dst := e2.Operand2().(*lir.Global)

					// Used x28 for storing the temporary value that is &GLOBAL_VARIABLE. Load cannot happen after return.
					wr.Write("\tadrp\t%s, %s\n", rf.GetI(r28).String(), dst.Name())
					wr.Write("\t%s\t%s, [%s, :lo12:%s]\n",
						store, src.String(), rf.GetI(r28).String(), dst.Name())
				default:
					panic(fmt.Sprintf("compiler error: unexpected store destination type %d", e2.Operand2().Type()))
				}
			case types.Constant:
				r := e2.GetHW().(*lir.LiveNode).Reg.(regfile.Register) // Assigned hardware register.
				if e2.DataType() == types.Int {
					val := e2.(*lir.Constant).Value().(int)
					if minImm <= val && val <= maxImm {
						// Used immediate instruction.
						wr.Write("\tmov\t%s, #%d\n", r.String(), val)
					} else {
						// Load hex string representation of integer and load. Use x28 as temporary register.
						cnst := e2.(*lir.Constant)
						istr := fmt.Sprintf("%s%d", labelConstant, cnst.GlobalSeq())
						wr.Write("\tadrp\t%s, %s\t\t//Load constant %d\n",
							rf.GetI(r28).String(), istr, cnst.Value().(int))
						wr.Write("\tldr\t%s, [%s, :lo12:%s]\n", r.String(), rf.GetI(r28).String(), istr)
						cnst.Use()
					}
				} else {
					// Load hex string representation of float into destination register. Use x28 as temporary register.
					cnst := e2.(*lir.Constant)
					fstr := fmt.Sprintf("%s%d", labelConstant, cnst.GlobalSeq())
					wr.Write("\tadrp\t%s, %s\t\t//Load constant %f\n",
						rf.GetI(r28).String(), fstr, cnst.Value().(float64))
					wr.Write("\tldr\t%s, [%s, :lo12:%s]\n", r.String(), rf.GetI(r28).String(), fstr)
					cnst.Use()
				}
			case types.CastInstruction:
				if e2.DataType() == types.Int {
					// Cast float to int.
					wr.Write("\tfcvtns\t%s, %s\n",
						e2.GetHW().(*lir.LiveNode).Reg.(regfile.Register).String(),
						e2.Operand1().GetHW().(*lir.LiveNode).Reg.(regfile.Register).String()) // Convert to nearest.
				} else {
					// Cast int to float.
					wr.Write("\tscvtf\t%s, %s\n",
						e2.GetHW().(*lir.LiveNode).Reg.(regfile.Register).String(),
						e2.Operand1().GetHW().(*lir.LiveNode).Reg.(regfile.Register).String())
				}
			case types.BranchInstruction:
				if err := genBranch(e2.(*lir.BranchInstruction), rf, wr, &ls); err != nil {
					return err
				}
			case types.ReturnInstruction:
				if err := genReturn(e2.(*lir.ReturnInstruction), fun, &rf, wr); err != nil {
					return err
				}
			case types.FunctionCallInstruction:
				if err := genFunctionCall(e2.(*lir.FunctionCallInstruction), rf, wr); err != nil {
					return err
				}
			case types.PreserveInstruction:
				// Preserves x0 or d0 from function calls.
				dst := e2.GetHW().(*lir.LiveNode).Reg.(regfile.Register)
				src := e2.Operand1().GetHW().(*lir.LiveNode).Reg.(regfile.Register)
				if e2.DataType() == types.Int {
					wr.Write("\tmov\t%s, %s\n", dst.String(), src.String())
				}else{
					wr.Write("\tfmov\t%s, %s\n", dst.String(), src.String())
				}
			case types.PrintInstruction, types.Global, types.Param, types.DeclareInstruction:
				// Ignore, because they've been handled during LIR construction.
				continue
			default:
				return fmt.Errorf("unexpected LIR instruction type %d", e2.Type())
			}
		}
	}
	return nil
}

// genReturn generates a function return statement. An error is returned if something went wrong.
func genReturn(v *lir.ReturnInstruction, fun *lir.Function, rf *RegisterFile, wr *util.Writer) error {
	r := v.Operand1().GetHW().(*lir.LiveNode).Reg.(regfile.Register)

	// Check if correct register index was assigned.
	if r.Id() != r0 {
		if r.Type() == int(i) {
			wr.Write("\tmov\t%s, %s\n", rf.regi[r0].String(), r.String())
		} else {
			wr.Write("\tfmov\t%s, %s\n", rf.regf[v0].String(), r.String())
		}
	}

	// Check if return value is of correct type.
	if r.Type() != int(fun.DataType()) {
		if r.Type() == int(i) {
			// Cast integer to float.
			wr.Write("\tscvtf\t%s, %s\n", rf.GetF(v0).String(), r.String())
		} else {
			// Cast float to integer.
			wr.Write("\tfcvtns\t%s, %s\n", rf.GetI(r0).String(), r.String()) // Convert to nearest.
		}
	}

	// Calculate allocated stack size.
	sa := wordSize * (len(fun.Params()) + len(fun.Locals()) + 2) // Stack adjust.
	spill := sa % stackAlign
	if spill != 0 {
		sa += stackAlign - spill
	}

	// Restore FP and LR.
	wr.Write("\tldp\t%s, %s, [%s, #%d]\n", rf.FP().String(), rf.LR().String(), rf.SP().String(), sa-(wordSize<<1))

	// De-allocate stack.
	wr.Write("\tadd\t%s, %s, #%d\n", rf.SP().String(), rf.SP().String(), sa)
	wr.Write("\tret\n")
	return nil
}
