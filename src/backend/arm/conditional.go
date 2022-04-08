package arm

import (
	"errors"
	"fmt"
	"vslc/src/ir"
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

// genRelation generates aarch64 assembler comparison of two values. An error is returned if something went wrong.
// A pointer to the result register is returned if everything went ok.
func genRelation(n *ir.Node, rf *registerFile, wr *util.Writer, st *util.Stack) error {
	if n == nil {
		return errors.New("compiler error: relation node is <nil>")
	}
	if n.Typ != ir.RELATION {
		return fmt.Errorf("line %d:%d: compiler error: expected node of type RELATION, got %s",
			n.Line, n.Pos, n.Type())
	}
	if len(n.Children) != 2 {
		return fmt.Errorf("line %d:%d: compiler error: relation node expected 2 children, got %d",
			n.Line, n.Pos, len(n.Children))
	}

	c1 := n.Children[0]
	c2 := n.Children[1]
	var op1, op2 *register

	// Load operand 1.
	switch c1.Typ {
	case ir.INTEGER_DATA:
		op1 = rf.lruI()
		wr.Write("\tmov\t%s, #%d\n", op1.String(), c1.Data.(int))
	case ir.FLOAT_DATA:
		label := floatToGlobalString(c1.Data.(float64))
		op1 = rf.lruF()
		wr.Write("\tldr\t%s, =%s\n", op1.String(), label)
	case ir.EXPRESSION:
		var err error
		op1, err = genExpression(c1, rf, wr, st)
		if err != nil {
			return err
		}
	case ir.IDENTIFIER_DATA:
		var err error
		op1, err = loadIdentifier(c1.Data.(string), rf, wr, st)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("line %d:%d: unexpected node type, expected INTEGER_DATA, FLOAT_DATA, EXPRESSION or IDENTIFIER, got %s",
			c1.Line, c1.Pos, c1.Type())
	}

	// Load operand 2.
	switch c2.Typ {
	case ir.INTEGER_DATA:
		op2 = rf.lruI()
		wr.Write("\tmov\t%s, #%d\n", op2.String(), c2.Data.(int))
	case ir.FLOAT_DATA:
		label := floatToGlobalString(c2.Data.(float64))
		op2 = rf.lruF()
		wr.Write("\tldr\t%s, =%s\n", op2.String(), label)
	case ir.EXPRESSION:
		var err error
		op2, err = genExpression(c2, rf, wr, st)
		if err != nil {
			return err
		}
	case ir.IDENTIFIER_DATA:
		var err error
		op2, err = loadIdentifier(c2.Data.(string), rf, wr, st)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("line %d:%d: unexpected node type, expected INTEGER_DATA, FLOAT_DATA, EXPRESSION or IDENTIFIER, got %s",
			c1.Line, c1.Pos, c1.Type())
	}

	// Check operand types.
	if op1.typ != op2.typ {
		// Cast operands.
		if op1.typ == i {
			// op1 is integer, op2 is float. Cast op1 to float.
			tmp := rf.lruF()
			wr.Write("\tscvtf\t%s, %s\n", tmp.String(), op1.String())
			op1 = tmp
		} else {
			// op1 is float, op2 is integer. Cast op2 to float.
			tmp := rf.lruF()
			wr.Write("\tscvtf\t%s, %s\n", tmp.String(), op2.String())
			op2 = tmp
		}
	}

	wr.Write("\tcmp\t%s, %s\n", op1.String(), op2.String())
	return nil
}

// genIf generates aarch64 assembler for an if statement, and recursively generates the THEN and ELSE bodies. Returns
// an error if something wrong happens.
func genIf(n *ir.Node, fun *ir.Symbol, rf *registerFile, wr *util.Writer, st, ls *util.Stack) error {
	if n == nil {
		return errors.New("compiler error: if node is <nil>")
	}
	if n.Typ != ir.IF_STATEMENT {
		return fmt.Errorf("line %d:%d: compiler error: expected node of type IF_STATEMENT, got %s",
			n.Line, n.Pos, n.Type())
	}
	if len(n.Children) < 2 {
		return fmt.Errorf("line %d:%d: compiler error: if node expected at least 2 children, got %d",
			n.Line, n.Pos, len(n.Children))
	}

	if err := genRelation(n.Children[0], rf, wr, st); err != nil {
		return err
	}

	if len(n.Children) == 2 {
		return genIfThen(n, fun, rf, wr, st, ls)
	} else {
		return genIfThenElse(n, fun, rf, wr, st, ls)
	}
}

// genIfThen generates an IF-THEN conditional and body.
func genIfThen(n *ir.Node, fun *ir.Symbol, rf *registerFile, wr *util.Writer, st, ls *util.Stack) error {
	converge := util.NewLabel(util.LabelIfEnd)
	switch n.Children[0].Data.(string) {
	case "=":
		wr.Write("\tb.ne\t%s\n", converge)
	case ">":
		wr.Write("\tb.le\t%s\n", converge)
	case "<":
		wr.Write("\tb.ge\t%s\n", converge)
	}

	// Generate THEN body.
	if err := gen(n.Children[1], fun, rf, wr, st, ls); err != nil {
		return err
	}

	// Jump here if false.
	wr.Label(converge)
	return nil
}

// genIfThenElse is like genIfThen, but it generates an additional ELSE body.
func genIfThenElse(n *ir.Node, fun *ir.Symbol, rf *registerFile, wr *util.Writer, st, ls *util.Stack) error {
	converge := util.NewLabel(util.LabelIfElseEnd)
	els := util.NewLabel(util.LabelIfElse)
	switch n.Children[0].Data.(string) {
	case "=":
		wr.Write("\tb.ne\t%s\n", els)
	case ">":
		wr.Write("\tb.le\t%s\n", els)
	case "<":
		wr.Write("\tb.ge\t%s\n", els)
	}

	// Begin THEN.
	if err := gen(n.Children[1], fun, rf, wr, st, ls); err != nil {
		return err
	}

	wr.Write("\tb\t%s\n", converge)
	// End THEN.
	wr.Label(els)
	// Begin ELSE.
	if err := gen(n.Children[2], fun, rf, wr, st, ls); err != nil {
		return err
	}

	// End ELSE.
	wr.Label(converge)
	return nil
}

// genWhile generates aarch64 assembler for a while loop. The while body is generated recursively. An error is returned
// if something went wrong.
func genWhile(n *ir.Node, fun *ir.Symbol, rf *registerFile, wr *util.Writer, st, ls *util.Stack) error {
	if n == nil {
		return errors.New("compiler error: if node is <nil>")
	}
	if n.Typ != ir.WHILE_STATEMENT {
		return fmt.Errorf("line %d:%d: compiler error: expected node of type WHILE_STATEMENT, got %s",
			n.Line, n.Pos, n.Type())
	}
	if len(n.Children) < 2 {
		return fmt.Errorf("line %d:%d: compiler error: if node expected 2 children, got %d",
			n.Line, n.Pos, len(n.Children))
	}

	head := util.NewLabel(util.LabelWhileHead)
	end := util.NewLabel(util.LabelWhileEnd)

	wr.Label(head)

	// Check for branch.
	if err := genRelation(n.Children[0], rf, wr, st); err != nil {
		return err
	}

	switch n.Children[0].Data.(string) {
	case "=":
		wr.Write("\tb.ne\t%s\n", end)
	case ">":
		wr.Write("\tb.le\t%s\n", end)
	case "<":
		wr.Write("\tb.ge\t%s\n", end)
	}

	// While body.
	if err := gen(n.Children[1], fun, rf, wr, st, ls); err != nil {
		return err
	}

	// Jump back to loop head.
	wr.Write("\tbr\t%s\n", head)

	// Converge.
	wr.Label(end)
	return errors.New("while statement not implemented yet")
}

// genContinue generates aarch64 unconditional branch instruction to jump to the last inserted label in the label stack.
func genContinue(wr *util.Writer, ls *util.Stack) error {
	l := ls.Peek()
	if l == nil {
		return errors.New("compiler error: label stack is empty")
	}
	if len(l.(string)) < 1 {
		return errors.New("compiler error: empty label found on label stack")
	}
	wr.Label(l.(string))
	return nil
}
