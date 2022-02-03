// This file contains RISC-V 64 assembly generating code for conditionals such as WHILE and IF-ELSE statements.

package riscv

import (
	"errors"
	"fmt"
	"vslc/src/ir"
	"vslc/src/util"
)

// ----------------------------
// ----- Type definitions -----
// ----------------------------

// ---------------------
// ----- Constants -----
// ---------------------

// -------------------
// ----- Globals -----
// -------------------

// ---------------------
// ----- Functions -----
// ---------------------
// genIf generates an IF or IF-ELSE statement. An error is returned if something went wrong.
func genIf(n *ir.Node, f *ir.Symbol, wr *util.Writer, st, ls *util.Stack, rf *registerFile) error {
	rel := n.Children[0]   // Relation.
	c1 := n.Children[1]    // Body of IF-THEN.
	var rs1, rs2 *register // Source registers for comparison.
	var err error

	// Move operands into registers.
	if rs1, rs2, err = genRel(rel, f, wr, st, rf); err != nil {
		return err
	}

	if len(n.Children) == 2 {
		// IF-THEN.
		lifend := util.NewLabel(util.LabelIfEnd) // The end of the IF-THEN statement.

		// Generate compare and jump.
		if err = genJump(rel, rs1, rs2, wr, lifend); err != nil {
			return err
		}

		// Generate IF-THEN part.
		if err = genAsm(c1, f, wr, st, ls, rf); err != nil {
			return err
		}

		// Jump here IF NOT taken.
		wr.Label(lifend)
	} else {
		// IF-THEN-ELSE.
		lels := util.NewLabel(util.LabelIfElse)       // The beginning of the ELSE part.
		lelsend := util.NewLabel(util.LabelIfElseEnd) // Then end of the IF-ELSE statement.
		c2 := n.Children[2]

		// Generate compare and jump.
		if err = genJump(rel, rs1, rs2, wr, lels); err != nil {
			return err
		}

		// Generate IF-THEN part.
		if err = genAsm(c1, f, wr, st, ls, rf); err != nil {
			return err
		}

		// When IF-THEN is finished; jump unconditionally to end of IF-THEN-ELSE statement.
		wr.Write("\tjal\t%s, %s\n", regi[zero], lelsend)

		// Jump here if ELSE was taken.
		wr.Label(lels)

		// Generate ELSE part.
		if err = genAsm(c2, f, wr, st, ls, rf); err != nil {
			return err
		}

		wr.Label(lelsend)
	}
	return nil
}

// genWhile generates a WHILE statement. An error is returned if something went wrong.
func genWhile(n *ir.Node, f *ir.Symbol, wr *util.Writer, st, ls *util.Stack, rf *registerFile) error {
	var rs1, rs2 *register // Source registers for comparison.
	var err error
	c1 := n.Children[0] // Relation.
	c2 := n.Children[1] // WHILE body.

	head := util.NewLabel(util.LabelWhileHead)
	end := util.NewLabel(util.LabelWhileEnd)

	// Append head label to label stack for continue statements.
	ls.Push(head)

	// Loop label.
	wr.Label(head)

	// Generate compare and jump.
	if rs1, rs2, err = genRel(c1, f, wr, st, rf); err != nil {
		return err
	}

	if err = genJump(c1, rs1, rs2, wr, end); err != nil {
		return err
	}

	// Generate WHILE body.
	if err = genAsm(c2, f, wr, st, ls, rf); err != nil {
		return err
	}

	// Unconditional jump to loop head.
	wr.Write("\tjal\t%s, %s\n", regi[zero], head)

	// Break label.
	wr.Label(end)
	ls.Pop()
	return nil
}

// genContinue generates a continue statement. An error is returned if something went wrong.
func genContinue(wr *util.Writer, ls *util.Stack) error {
	if l := ls.Peek(); l != nil {
		wr.Write("\tjal\t%s, %s\n", regi[zero], l.(string))
	} else {
		return errors.New("label stack is empty")
	}
	return nil
}

// genRel generates a relation by moving both operands to some registers.
// An error is returned if something went wrong.
func genRel(n *ir.Node, f *ir.Symbol, wr *util.Writer, st *util.Stack, rf *registerFile) (rs1, rs2 *register, err error) {
	c1 := n.Children[0]
	c2 := n.Children[1]
	var op1, op2 string

	// Move operand 1.
	switch c1.Typ {
	case ir.IDENTIFIER_DATA:
		rs1 = rf.loadIdentifierToReg(c1.Data.(string), f, wr, st)
		op1 = rs1.String()
	case ir.FLOAT_DATA:
		// Move constant float to register.
		rs1 = rf.lruF()
		wr.Write("\tlui\t%s, %%hi(%s%d)\n", rs1.String(), labelFloat, n.Data.(int))                    // Move high 20 bits.
		wr.Write("\taddi\t%s, %s, %%lo(%s%d)\n", rs1.String(), rs1.String(), labelFloat, n.Data.(int)) // Append low 12 bits.
		op1 = rs1.String()
	case ir.INTEGER_DATA:
		// Move integer constant to integer register.
		rs1 = rf.lruI()
		wr.Write("\tli\t%s, %d\n", rs1.String(), c1.Data.(int))
		op1 = rs1.String()
	case ir.EXPRESSION:
		if rs1, err = genExpression(c1, f, wr, st, rf); err != nil {
			return nil, nil, err
		} else {
			if rs1.typ == float {
				// Result of expression is float.
				op1 = rs1.String()
			} else {
				// Result of expression is integer.
				r := rf.lruF()
				op1 = r.String()
				wr.Write("\tfcvt.s.w\t%s, %s\n", op1, rs1.String()) // Convert int to float and move to float register.
			}
		}
	}

	// Move operand 2.
	switch c2.Typ {
	case ir.IDENTIFIER_DATA:
		rs2 = rf.loadIdentifierToReg(c2.Data.(string), f, wr, st)
		op2 = rs2.String()
	case ir.FLOAT_DATA:
		// Move constant float to register.
		rs2 = rf.lruF()
		wr.Write("\tlui\t%s, %%hi(%s%d)\n", rs2.String(), labelFloat, n.Data.(int))                    // Move high 20 bits.
		wr.Write("\taddi\t%s, %s, %%lo(%s%d)\n", rs2.String(), rs2.String(), labelFloat, n.Data.(int)) // Append low 12 bits.
		op2 = rs2.String()
	case ir.INTEGER_DATA:
		// Move integer constant to integer register.
		rs2 = rf.lruI()
		wr.Write("\tli\t%s, %d\n", rs2.String(), c2.Data.(int))
		// Move integer register to float register.
		op2 = rs2.String()
	case ir.EXPRESSION:
		if rs2, err = genExpression(c2, f, wr, st, rf); err != nil {
			return nil, nil, err
		} else {
			if rs2.typ == float {
				// Result of expression is float.
				r := rf.lruF()
				op2 = r.String()
				wr.Write("\tfcvt.s.w\t%s, %s\n", op2, rs2.String()) // Convert int to float and move to float register.
			} else {
				// Result of expression is integer.
				op2 = rs2.String()
			}
		}
	}
	return rs1, rs2, nil
}

// genJump generates a jump instruction based on the number of labels provided. 1 label means IF-THEN statement,
// 2 labels mean IF-THEN-ELSE statement. An error is returned if something went wrong.
func genJump(n *ir.Node, rs1, rs2 *register, wr *util.Writer, label string) error {
	var op string
	switch n.Data.(string) {
	case "=":
		op = "bne"
	case "<":
		op = "bge"
	case ">":
		op = "blt"
	default:
		return fmt.Errorf("line %d:%d: undefined relation operator %q", n.Line, n.Pos, n.Data.(string))
	}

	wr.Ins3(op, rs1.String(), rs2.String(), label)
	return nil
}
