// This file contains RISC-V 64 assembly generating code for expressions.

package riscv

import (
	"fmt"
	"vslc/src/ir"
	"vslc/src/util"
)

// ---------------------
// ----- Functions -----
// ---------------------

// TODO: set result of expression in a0, and operands for binary a0 and a1.

// genExpression generates a binary or unary expression and returns a pointer to the register where the result is put.
// An error is returned if something goes wrong.
func genExpression(n *ir.Node, f *ir.Symbol, wr *util.Writer, st *util.Stack, rf *registerFile) (*register, error) {
	if n.Data == nil {
		// FUNCTION call.
		return genFunctionCall(n, f, wr, st, rf)
	}

	if len(n.Children) == 2 {
		// Binary expression.
		var op1t, op2t int
		c1 := n.Children[0]
		c2 := n.Children[1]

		// Check data type of operand 1.
		switch c1.Typ {
		case ir.INTEGER_DATA:
			op1t = int(ir.DataInteger)
		case ir.FLOAT_DATA:
			return genExpressionFloat(n, f, wr, st, rf)
		case ir.IDENTIFIER_DATA:
			reg := rf.loadIdentifierToReg(c1.Data.(string), f, wr, st)
			op1t = reg.typ
		case ir.EXPRESSION:
			if r, err := genExpression(c1, f, wr, st, rf); err != nil {
				return nil, err
			} else {
				op1t = r.typ
			}
		}

		// Float overrides int.
		if op1t == float {
			return genExpressionFloat(n, f, wr, st, rf)
		}

		// Check data type of operand 2.
		switch c2.Typ {
		case ir.INTEGER_DATA:
			if op1t != float {
				return genExpressionInt(n, f, wr, st, rf)
			}
		case ir.FLOAT_DATA:
			return genExpressionFloat(n, f, wr, st, rf)
		case ir.IDENTIFIER_DATA:
			reg := rf.loadIdentifierToReg(c2.Data.(string), f, wr, st)
			op2t = reg.typ
		case ir.EXPRESSION:
			if r, err := genExpression(c2, f, wr, st, rf); err != nil {
				return nil, err
			} else {
				op2t = r.typ
			}
		}

		// Float overrides int.
		if op2t == float {
			return genExpressionFloat(n, f, wr, st, rf)
		} else {
			return genExpressionInt(n, f, wr, st, rf)
		}
	} else {
		// Unary expression.
		c1 := n.Children[0]
		var op1t int
		switch c1.Typ {
		case ir.INTEGER_DATA:
			return genExpressionInt(n, f, wr, st, rf)
		case ir.FLOAT_DATA:
			return genExpressionFloat(n, f, wr, st, rf)
		case ir.IDENTIFIER_DATA:
			reg := rf.loadIdentifierToReg(c1.Data.(string), f, wr, st)
			op1t = reg.typ
		case ir.EXPRESSION:
			if r, err := genExpression(c1, f, wr, st, rf); err != nil {
				return nil, err
			} else {
				op1t = r.typ
			}
		}

		// Float overrides int.
		if op1t == float {
			return genExpressionFloat(n, f, wr, st, rf)
		} else {
			return genExpressionInt(n, f, wr, st, rf)
		}
	}
}

// genExpressionInt generates an integer expression and returns a pointer to the register where the result is put.
// An error is returned if the expression is not valid.
func genExpressionInt(n *ir.Node, f *ir.Symbol, wr *util.Writer, st *util.Stack, rf *registerFile) (*register, error) {
	if len(n.Children) == 2 {
		// Binary operator.
		c1 := n.Children[0]
		c2 := n.Children[1]
		var op, op1, op2 string
		var imm1, imm2 bool // True if either operand is an immediate.
		var imm bool        // True if the operator has an immediate instruction equivalent.

		// At this point we know that neither operator is float.

		// Check if operator allows immediate operand.
		switch n.Data.(string) {
		case "+", "^", "|", "&", "<<", ">>":
			imm = true
		}

		// Move operand two first, because not all instructions are commutative.
		// Move operand one.
		switch c2.Typ {
		case ir.IDENTIFIER_DATA:
			r := rf.loadIdentifierToReg(c2.Data.(string), f, wr, st)
			op2 = r.String()
		case ir.INTEGER_DATA:
			i := c2.Data.(int)
			if imm && i >= minImm && i <= maxImm {
				// This constant will fit into RISC-V 12 bit immediate.
				imm2 = true
			} else {
				// Move constant to register.
				reg := rf.lruI()
				wr.Write("\tlui\t%s, %%hi(%d)\n", reg.String(), i)
				wr.Write("\taddi\t%s, %s, %%lo(%d)\n", reg.String(), reg.String(), i)
				op2 = reg.String()
			}
		case ir.EXPRESSION:
			if r, err := genExpression(c2, f, wr, st, rf); err != nil {
				return nil, err
			} else {
				if r.typ != integer {
					return nil, fmt.Errorf("line %d:%d: cannot assign float to int", c2.Line, c2.Pos)
				} else {
					op2 = r.String()
				}
			}
		}

		// Move operand one.
		switch c1.Typ {
		case ir.IDENTIFIER_DATA:
			r := rf.loadIdentifierToReg(c1.Data.(string), f, wr, st)
			op1 = r.String()
		case ir.INTEGER_DATA:
			i := c1.Data.(int)
			if imm && !imm2 && i >= minImm && i <= maxImm {
				// This constant will fit into RISC-V 12 bit immediate.
				imm1 = true
			} else {
				// Move constant to register.
				reg := rf.lruI()
				wr.Write("\tlui\t%s, %%hi(%d)\n", reg.String(), i)
				wr.Write("\taddi\t%s, %s, %%lo(%d)\n", reg.String(), reg.String(), i)
				op1 = reg.String()
			}
		case ir.EXPRESSION:
			if r, err := genExpression(c1, f, wr, st, rf); err != nil {
				return nil, err
			} else {
				if r.typ != integer {
					return nil, fmt.Errorf("line %d:%d: cannot assign float to int", c1.Line, c1.Pos)
				} else {
					op1 = r.String()
				}
			}
		}

		// Set operator.
		switch n.Data.(string) {
		case "+":
			// Commutative.
			if imm1 || imm2 {
				op = "addi"
			} else {
				op = "add"
			}
		case "-":
			// Not commutative.
			op = "sub"
		case "*":
			// Commutative.
			op = "mul"
		case "/":
			// Not commutative.
			// Division by zero constant is caught in earlier compiler stage.
			op = "div"
		case "%":
			// Not commutative.
			op = "rem"
		case "^":
			// Commutative.
			if imm1 || imm2 {
				op = "xori"
			} else {
				op = "xor"
			}
		case "|":
			// Commutative.
			if imm1 || imm2 {
				op = "ori"
			} else {
				op = "or"
			}
		case "&":
			// Commutative.
			if imm1 || imm2 {
				op = "andi"
			} else {
				op = "and"
			}
		case "<<":
			// Not commutative.
			if imm1 || imm2 {
				op = "slli"
			} else {
				op = "sll"
			}
		case ">>":
			// Not commutative.
			if imm1 || imm2 {
				op = "srli"
			} else {
				op = "srl"
			}
		}

		res := rf.lruI()
		if imm {
			// Operator allows immediate operand.
			if imm1 {
				wr.Ins2imm(op, res.String(), op2, c1.Data.(int))
			} else if imm2 {
				wr.Ins2imm(op, res.String(), op1, c2.Data.(int))
			} else {
				wr.Ins3(op, res.String(), op1, op2)
			}
		} else {
			// Operator does not allow immediate operand.
			wr.Ins3(op, res.String(), op1, op2)
		}
		return res, nil
	} else {
		// Unary operator.
		c1 := n.Children[0]
		var op1 string

		// By this point we know that the operator is an integer.

		// Move operand.
		switch c1.Typ {
		case ir.IDENTIFIER_DATA:
			// Move from memory to register.
			reg := rf.loadIdentifierToReg(c1.Data.(string), f, wr, st)
			op1 = regi[reg.id]
		case ir.INTEGER_DATA:
			// Move constant to register.
			reg := rf.lruI()
			wr.Write("\tli\t%s, %d\n", reg.String(), c1.Data.(int))
		case ir.EXPRESSION:
			// Generate expression and check if result is integer.
			if r, err := genExpression(c1, f, wr, st, rf); err != nil {
				return nil, err
			} else {
				if r.typ == integer {
					// Result of expression is float.
					op1 = regf[r.id]
				} else {
					return nil, fmt.Errorf("line %d:%d: cannot assign float to integer", c1.Line, c1.Pos)
				}
			}
		}

		// Set result register.
		res := rf.lruI()

		// Set operator.
		switch n.Data.(string) {
		case "-":
			// Move -1 to register and multiply by operand.
			wr.Write("\tli\t%s, %d\n", res.String(), -1)
			wr.Ins3("mul", res.String(), res.String(), op1)
		case "~":
			// TODO: make sure below mask is correct if this compiler should work for RISC-V 64-bit. Add to asm memory?
			wr.Write("\tlui\t%s, %%hi(%d)\n", res.String(), 0xFFFFFFFF)
			wr.Write("\taddi\t%s, %s, %%lo(%d)\n", res.String(), res.String(), 0xFFFFFFFF)
			wr.Ins3("xor", res.String(), res.String(), op1)
		default:
			return nil, fmt.Errorf("operator %q not defined for integer", n.Data.(string))
		}
		return res, nil
	}
}

// genExpressionFloat generates a float expression and returns a pointer to the register where the result is put.
// An error is returned if the expression is not valid.
func genExpressionFloat(n *ir.Node, f *ir.Symbol, wr *util.Writer, st *util.Stack, rf *registerFile) (*register, error) {
	if len(n.Children) == 2 {
		c1 := n.Children[0]
		c2 := n.Children[1]
		var op, op1, op2 string

		// Load operand one.
		switch c1.Typ {
		case ir.IDENTIFIER_DATA:
			reg := rf.loadIdentifierToReg(c1.Data.(string), f, wr, st)
			op1 = regf[reg.id]
		case ir.INTEGER_DATA:
			// Move integer constant to integer register.
			reg := rf.lruI()
			wr.Write("\tli\t%s, %d\n", reg.String(), c1.Data.(int))
			// Move integer register to float register.
			r := rf.lruF()
			wr.Write("\tfcvt.s.w\t%s, %s\n", r.String(), reg.String()) // Convert int to float and move to float register.
			op1 = reg.String()
		case ir.FLOAT_DATA:
			// Move constant float to register.
			reg := rf.lruF()
			wr.Write("\tlui\t%s, %%hi(%s%d)\n", reg.String(), labelFloat, n.Data.(int))                    // Move high 20 bits.
			wr.Write("\taddi\t%s, %s, %%lo(%s%d)\n", reg.String(), reg.String(), labelFloat, n.Data.(int)) // Append low 12 bits.
			op1 = reg.String()
		case ir.EXPRESSION:
			// Generate expression and check if result is float.
			if r, err := genExpression(c1, f, wr, st, rf); err != nil {
				return nil, err
			} else {
				if r.typ == float {
					// Result of expression is float.
					rs1 := rf.lruF()
					op1 = rs1.String()
					wr.Write("\tfcvt.s.w\t%s, %s\n", op1, r.String()) // Convert int to float and move to float register.
				} else {
					// Result of expression is integer.
					op1 = r.String()
				}
			}
		}

		// Load operand one.
		switch c2.Typ {
		case ir.IDENTIFIER_DATA:
			reg := rf.loadIdentifierToReg(c2.Data.(string), f, wr, st)
			op2 = regf[reg.id]
		case ir.INTEGER_DATA:
			// Move integer constant to integer register.
			reg := rf.lruI()
			wr.Write("\tli\t%s, %d\n", reg.String(), c2.Data.(int))
			// Move integer register to float register.
			r := rf.lruF()
			wr.Write("\tfcvt.s.w\t%s, %s\n", r.String(), reg.String()) // Convert int to float and move to float register.
			op2 = reg.String()
		case ir.FLOAT_DATA:
			// Move constant float to register.
			reg := rf.lruF()
			wr.Write("\tlui\t%s, %%hi(%s%d)\n", reg.String(), labelFloat, n.Data.(int))                    // Move high 20 bits.
			wr.Write("\taddi\t%s, %s, %%lo(%s%d)\n", reg.String(), reg.String(), labelFloat, n.Data.(int)) // Append low 12 bits.
			op2 = reg.String()
		case ir.EXPRESSION:
			// Generate expression and check if result is float.
			if r, err := genExpression(c2, f, wr, st, rf); err != nil {
				return nil, err
			} else {
				if r.typ == float {
					// Result of expression is float.
					rs1 := rf.lruF()
					op2 = rs1.String()
					wr.Write("\tfcvt.s.w\t%s, %s\n", op2, r.String()) // Convert int to float and move to float register.
				} else {
					// Result of expression is integer.
					op2 = r.String()
				}
			}
		}

		// Set correct operator.
		switch n.Data.(string) {
		case "+":
			op = "fadd.s"
		case "-":
			op = "fsub.s"
		case "*":
			op = "fmul.s"
		case "/":
			op = "fdiv.s"
		default:
			return nil, fmt.Errorf("operator %q not defined for float", n.Data.(string))
		}

		res := rf.lruF()
		wr.Ins3(op, res.String(), op1, op2)
		return res, nil
	} else {
		if n.Data.(string) != "-" {
			return nil, fmt.Errorf("operator %q not defined for float", n.Data.(string))
		}
		c1 := n.Children[0]
		var op1 string

		switch c1.Typ {
		case ir.IDENTIFIER_DATA:
			reg := rf.loadIdentifierToReg(c1.Data.(string), f, wr, st)
			op1 = regf[reg.id]
		case ir.INTEGER_DATA:
			// Move integer constant to integer register.
			reg := rf.lruI()
			wr.Write("\tmov\t%s, %d\n", reg.String(), c1.Data.(int))
			// Move integer register to float register.
			r := rf.lruF()
			wr.Write("\tfcvt.s.w\t%s, %s\n", r.String(), reg.String()) // Convert int to float and move to float register.
			op1 = r.String()
		case ir.FLOAT_DATA:
			// Move constant float to register.
			reg := rf.lruF()
			wr.Write("\tlui\t%s, %%hi(%s%d)\n", reg.String(), labelFloat, n.Data.(int))                    // Move high 20 bits.
			wr.Write("\taddi\t%s, %s, %%lo(%s%d)\n", reg.String(), reg.String(), labelFloat, n.Data.(int)) // Append low 12 bits.
			op1 = reg.String()
		case ir.EXPRESSION:
			// Generate expression and check if result is float.
			if r, err := genExpression(c1, f, wr, st, rf); err != nil {
				return nil, err
			} else {
				if r.typ == float {
					// Result of expression is float.
					rs1 := rf.lruF()
					op1 = rs1.String()
					wr.Write("\tfcvt.s.w\t%s, %s\n", op1, r.String()) // Convert int to float and move to float register.
				} else {
					// Result of expression is integer.
					op1 = r.String()
				}
			}
		}

		// Move -1 to int register, convert and move to float register.
		reg := rf.lruI()
		wr.Write("\tmovi\t%s, -1\n", reg.String())
		res := rf.lruF()
		wr.Write("\tfcvt.s.w\t%s, %s\n", res.String(), reg.String())        // Convert int to float and move to float register.
		wr.Write("\tfmul.s\t%s, %s, %s\n", res.String(), res.String(), op1) // Multiply by -1.
		return res, nil
	}
}
