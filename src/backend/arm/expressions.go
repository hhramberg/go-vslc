package arm

import (
	"fmt"
	"vslc/src/backend/regfile"
	"vslc/src/ir/lir"
	"vslc/src/ir/lir/types"
)

import (
	"vslc/src/util"
)

// -----------------------------
// ----- Type definitions ------
// -----------------------------

// ---------------------
// ----- Constants -----
// ---------------------

// -------------------
// ----- globals -----
// -------------------

// --------------------
// ----- Function -----
// --------------------

// genExpression generates aarch64 assembler for arithmetic expressions. An error is returned if something went wrong.
func genExpression(v *lir.DataInstruction, wr *util.Writer) error {
	op1 := v.Operand1()
	op2 := v.Operand2()
	dst := v.GetHW().(*lir.LiveNode).Reg.(regfile.Register)
	reg1 := op1.GetHW().(*lir.LiveNode).Reg.(regfile.Register)

	if op2 != nil {
		// Binary expression.
		reg2 := op2.GetHW().(*lir.LiveNode).Reg.(regfile.Register)

		// Choose instruction from operator.
		if dst.Type() == int(types.Int) {
			// Integer operations.
			switch v.Operator() {
			case types.Add:
				wr.Write("\tadd\t%s, %s, %s\n", dst.String(), reg1.String(), reg2.String())
			case types.Sub:
				wr.Write("\tsub\t%s, %s, %s\n", dst.String(), reg1.String(), reg2.String())
			case types.Mul:
				wr.Write("\tmul\t%s, %s, %s\n", dst.String(), reg1.String(), reg2.String())
			case types.Div:
				// Signed division. Division by zero caught in validate.
				wr.Write("\tsdiv\t%s, %s, %s\n", dst.String(), reg1.String(), reg2.String())
			case types.Rem:
				// From: https://stackoverflow.com/questions/35351470/obtaining-remainder-using-single-aarch64-instruction
				// Also division by zero is caught in validate.
				wr.Write("\tudiv\t%s, %s, %s\n", dst.String(), reg1.String(), reg2.String())
				wr.Write("\tmsub\t%s, %s, %s, %s\n", dst.String(), dst.String(), reg2.String(), reg1.String())
			case types.And:
				wr.Write("\tand\t%s, %s, %s\n", dst.String(), reg1.String(), reg2.String())
			case types.Xor:
				wr.Write("\teor\t%s, %s, %s\n", dst.String(), reg1.String(), reg2.String())
			case types.Or:
				wr.Write("\torr\t%s, %s, %s\n", dst.String(), reg1.String(), reg2.String())
			case types.RShift:
				wr.Write("\tlsr\t%s, %s, %s\n", dst.String(), reg1.String(), reg2.String())
			case types.LShift:
				wr.Write("\tlsl\t%s, %s, %s\n", dst.String(), reg1.String(), reg2.String())
			default:
				return fmt.Errorf("unexpected binary operator %q", v.Operator().String())
			}
		} else {
			switch v.Operator() {
			case types.Add:
				wr.Write("\tfadd\t%s, %s, %s\n",
					v.GetHW().(*lir.LiveNode).Reg.(regfile.Register).String(),
					op1.GetHW().(*lir.LiveNode).Reg.(regfile.Register).String(),
					op2.GetHW().(*lir.LiveNode).Reg.(regfile.Register).String())
			case types.Sub:
				wr.Write("\tfsub\t%s, %s, %s\n", dst.String(), reg1.String(), reg2.String())
			case types.Mul:
				wr.Write("\tfmul\t%s, %s, %s\n", dst.String(), reg1.String(), reg2.String())
			case types.Div:
				wr.Write("\tfdiv\t%s, %s, %s\n", dst.String(), reg1.String(), reg2.String())
			default:
				return fmt.Errorf("unexpected binary operator %q", v.Operator().String())
			}
		}
	} else {
		// Unary expression.
		switch v.Operator() {
		case types.Sub:
			wr.Write("\tneg\t%s, %s\n", dst.String(), reg1.String())
		case types.Not:
			wr.Write("\tmvn\t%s, %s\n", dst.String(), reg1.String())
		default:
			return fmt.Errorf("unexpected unary operator %q", v.Operator().String())
		}
	}
	return nil
}

// genFunctionCall generates aarch64 assembler for a function call. An error is returned if something went wrong. The
// result of the function call is put in register a0 for integers or v0 for floating point functions.
func genFunctionCall(v *lir.FunctionCallInstruction, rf regfile.RegisterFile, wr *util.Writer) error {

	// Check if we need to pass arguments on stack.
	nargs := len(v.Arguments()) // Number of arguments.
	stack := 0
	if nargs > paramReg {
		// Worst case is everything is same datatype.
		stack = nargs - paramReg
		spill := stack % stackAlign
		if spill != 0 {
			stack += stackAlign - spill
		}
		wr.Write("\tsub\t%s, %s, #%d\n", rf.SP().String(), rf.SP().String(), stack)
	}

	ii := 1 // Number of integer arguments.
	fi := 1 // Number of float arguments.

	// Move arguments[1:] first, because a0 and v0 are used for intermediate operations during operand casting.
	if len(v.Arguments()[1:]) > 0 {
		for i1, e1 := range v.Arguments() {
			// Expression list.
			param := v.Target().Params()[i1]
			reg := e1.GetHW().(*lir.LiveNode).Reg.(regfile.Register)

			// TODO: How does this go with strings?
			if param.DataType() == i {
				// Require integer. Cast float to integer.
				switch e1.DataType() {
				case types.Int, types.String:
					if ii < paramReg {
						// Put constant in register.
						wr.Write("\tmov\t%s, %s\n", rf.GetI(r0+ii), reg.String())
					} else {
						// Put constant on stack.
						pos := ii - paramReg
						if fi >= paramReg {
							pos += fi - paramReg
						}
						wr.Write("\tstr\t%s, [%s, #%d]\n", reg.String(), rf.SP().String(), pos)
					}
					ii++
				case types.Float:
					// Cast float to integer.
					if ii < paramReg {
						// Put argument in register.
						wr.Write("\tfcvtzs\t%s, %s\n", rf.GetI(r0+ii), reg.String())
					} else {
						// Store immediate on stack.
						pos := ii - paramReg
						if fi >= paramReg {
							pos += fi - paramReg
						}
						wr.Write("\tfcvtzs\t%s, %s\n", rf.GetI(a0), reg.String()) // Use a0 as temporary register.
						wr.Write("\tstr\t%s, [%s, #%d]\n", rf.GetI(a0), rf.SP().String(), pos)
					}
					ii++
				default:
					return fmt.Errorf("unexpected function argument, got data type %s",
						e1.DataType().String())
				}
			} else if param.DataType() == types.Float{
				// Require float. Cast integer to float.
				switch e1.DataType() {
				case types.Int:
					// Cast integer to float.
					if fi < paramReg {
						// Put in register.
						wr.Write("\tsvctf\t%s, %s\n", rf.GetF(v0+fi).String(), reg.String())
					} else {
						// Put on stack.
						pos := fi - paramReg
						if fi >= paramReg {
							pos += fi - paramReg
						}
						wr.Write("\tsvctf\t%s, %s\n", rf.GetF(v0).String(), reg.String())
						wr.Write("\tstr\t%s, [%s, #%d]\n", rf.GetF(v0), rf.SP().String(), pos)
					}
					fi++
				case types.Float:
					if fi < paramReg {
						wr.Write("\tmov\t%s, %s\n", rf.GetF(v0+fi), reg.String())
					} else {
						// Put constant on stack.
						pos := ii - paramReg
						if fi >= paramReg {
							pos += fi - paramReg
						}
						wr.Write("\tstr\t%s, [%s, #%d]\n", reg.String(), rf.SP().String(), pos)
					}
					fi++
				default:
					return fmt.Errorf("unexpected function argument, got data type %s",
						e1.DataType().String())
				}
			}else{
				// Require string.
				// TODO: Handle.
			}
		}

		// Move argument r0 and v0 last, because they were used as temporaries during operand casting.
		param := v.Target().Params()[0]
		reg := v.Arguments()[0].GetHW().(*lir.LiveNode).Reg.(regfile.Register)
		if param.DataType() == types.Int {
			// Require integer.
			switch v.Arguments()[0].DataType() {
			case types.Int, types.String:
				wr.Write("\tmov\t%s, %s\n", rf.GetI(a0), reg.String())
			case types.Float:
				// Cast float to int and move.
				wr.Write("\tfcvtzs\t%s, %s\n", rf.GetI(a0), reg.String()) // Use a0 as temporary register.
			}
		} else {
			// Require float.
			switch v.Arguments()[0].DataType() {
			case types.Int, types.String:
				// Cast int to float and move.
				wr.Write("\tscvtf\t%s, %s\n", rf.GetF(v0), reg.String()) // Use a0 as temporary register.
			case types.Float:
				wr.Write("\tmov\t%s, %s\n", rf.GetF(v0), reg.String())
			}
		}
	}

	// Call function.
	wr.Write("\tbl\t%s\n", v.Target().Name())

	// De-allocate stack for arguments, if any.
	if nargs > paramReg {
		stack = nargs - paramReg
		wr.Write("\tadd\t%s, %s, #%d\n", rf.SP().String(), rf.SP().String(), stack)
	}
	return nil
}
