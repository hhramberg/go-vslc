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
	nargs := 0
	ni := 0 // Number of integer arguments.
	nf := 0 // Number of float arguments.

	for _, e1 := range v.Arguments() {
		if e1.DataType() == types.VaList {
			nargs += len(e1.(*lir.VaList).Values())
			for _, e2 := range e1.(*lir.VaList).Values() {
				if e2.DataType() == types.String || e2.DataType() == types.Int {
					ni++
				} else {
					nf++
				}
			}
		} else {
			nargs++
			if e1.DataType() == types.Int || e1.DataType() == types.String {
				ni++
			} else {
				nf++
			}
		}
	}
	stack := 0
	if ni > paramReg {
		stack += paramReg - ni
	}
	if nf > paramReg {
		stack += paramReg - nf
	}
	if stack > 0 {
		stack *= wordSize
		res := stack % stackAlign
		if res != 0 {
			stack += stackAlign - res
		}
		wr.Write("\tsub\t%s, %s, #%d\n", rf.SP().String(), rf.SP().String(), stack)
	}

	if len(v.Arguments()) > 0 {
		ii := 0 // Index of current or last integer argument.
		fi := 0 // Index of current or last float argument.

		// Generate argument passing.
		for i1, e1 := range v.Arguments() {
			arg := e1
			param := v.Target().Params()[i1]

			if param.DataType() == types.Int || param.DataType() == types.String {
				if ii < paramReg {
					// Use integer registers.
					wr.Write("\tmov\t%s, %s\n",
						rf.GetI(ii), arg.GetHW().(*lir.LiveNode).Reg.(regfile.Register).String())
				} else {
					// Put on stack.
					wr.Write("\tstr\t%s, [%s, #%d]\n",
						arg.GetHW().(*lir.LiveNode).Reg.(regfile.Register).String(), rf.SP().String(), nargs-1)
				}
				ii++
				nargs--
			} else if arg.DataType() == types.Float {
				if fi < paramReg {
					// Use float registers.
					wr.Write("\tmov\t%s, %s\n",
						rf.GetF(fi), arg.GetHW().(*lir.LiveNode).Reg.(regfile.Register).String())
				} else {
					// Put on stack.
					wr.Write("\tstr\t%s, [%s, #%d]\n",
						arg.GetHW().(*lir.LiveNode).Reg.(regfile.Register).String(), rf.SP().String(), nargs-1)
				}
				fi++
				nargs--
			} else if arg.DataType() == types.VaList {
				// VaList is used exclusively by calls to printf.
				for _, e2 := range arg.(*lir.VaList).Values() {
					varg := e2.GetHW().(*lir.LiveNode).Reg.(regfile.Register)
					if e2.DataType() == types.Int || e2.DataType() == types.String {
						// Int or strings.
						if fi < paramReg {
							// Move to register.
							wr.Write("\tmov\t%s, %s\n", rf.GetI(ii).String(), varg.String())
						} else {
							// Pass on stack.
							wr.Write("\tstr\t%s, [%s, #%d]\n", varg.String(), rf.SP().String(), nargs-1)
						}
						ii++
						nargs--
					} else {
						// Float.
						if fi < paramReg {
							// Move to register.
							wr.Write("\tmov\t%s, %s\n", rf.GetF(fi).String(), varg.String())
						} else {
							// Pass on stack.
							wr.Write("\tstr\t%s, [%s, #%d]\n", varg.String(), rf.SP().String(), nargs-1)
						}
						fi++
						nargs--
					}
				}
			} else {
				return fmt.Errorf("cannot create function call assembler: unexpected data type: %s",
					arg.DataType().String())
			}

		}
	}

	// Call function.
	wr.Write("\tbl\t%s\n", v.Target().Name())

	// De-allocate stack for arguments, if any.
	if stack > 0 {
		wr.Write("\tadd\t%s, %s, #%d\n", rf.SP().String(), rf.SP().String(), stack)
	}
	return nil
}
