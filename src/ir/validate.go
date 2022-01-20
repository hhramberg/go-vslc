package ir

import (
	"fmt"
	"vslc/src/util"
)

// ----------------------------
// ----- Type definitions -----
// ----------------------------

// ---------------------
// ----- Constants -----
// ---------------------

// Binary operators.
const (
	opPlus  = iota // Binary plus.
	opMinus        // Binary minus.
	opMul          // Binary multiply.
	opDiv          // Binary division.
	opMod          // Binary modulo operator.
	opOr           // Binary OR.
	opAnd          // Binary AND.
	opXor          // Binary XOR.
	opLsh          // Binary left shift.
	opRsh          // Binary right shift .
	opGt           // Binary greater than.
	opLt           // Binary less than.
	opEq           // Binary equals.
)

// lut is the lookup table for binary expressions and type compatability.
// Dimensions
// 1 Datatype of operand 1.
// 2 Datatype of operand 2.
// 3 Operator.
var lut = [2][2][opEq + 1]bool{
	{
		// OP1 is integer.
		{
			// OP2 is integer.
			opPlus:  true,
			opMinus: true,
			opMul:   true,
			opDiv:   true,
			opMod:   true,
			opOr:    true,
			opAnd:   true,
			opXor:   true,
			opLsh:   true,
			opRsh:   true,
			opGt:    true,
			opLt:    true,
			opEq:    true,
		},
		{
			// OP2 is float.
			opPlus:  true,
			opMinus: true,
			opMul:   true,
			opDiv:   true,
			opMod:   false,
			opOr:    false,
			opAnd:   false,
			opXor:   false,
			opLsh:   false,
			opRsh:   false,
			opGt:    false,
			opLt:    false,
			opEq:    false,
		},
	},
	{
		// OP1 is float.
		{
			// OP2 is integer.
			opPlus:  true,
			opMinus: true,
			opMul:   true,
			opDiv:   true,
			opMod:   false,
			opOr:    false,
			opAnd:   false,
			opXor:   false,
			opLsh:   false,
			opRsh:   false,
			opGt:    false,
			opLt:    false,
			opEq:    false,
		},
		{
			// OP2 is float.
			opPlus:  true,
			opMinus: true,
			opMul:   true,
			opDiv:   true,
			opMod:   false,
			opOr:    false,
			opAnd:   false,
			opXor:   false,
			opLsh:   false,
			opRsh:   false,
			opGt:    false,
			opLt:    false,
			opEq:    false,
		},
	},
}

// -------------------
// ----- Globals -----
// -------------------

// ----------------------
// ----- Functions ------
// ----------------------

// ValidateTree validates types for expressions and assignments.
func ValidateTree(opt util.Options) error {
	if opt.Threads > 1 {
		// Parallel.
		// TODO: implement.
	} else {
		// Sequential.
		for _, e1 := range Root.Children[0].Children {
			fmt.Println(e1.String()) // TODO: fix!
		}
	}
	return nil
}

// validate recursively checks expressions and assignments for type validation.
func (n *Node) validate(target dataType) error {
	switch n.Typ {
	case ASSIGNMENT_STATEMENT:
		// Check left and right-hand side data types.
		c0 := n.Children[0]
		c1 := n.Children[1]

		switch c0.Entry.DataTyp {
		case DataInteger:
			// Accept only integer.
			switch c1.Typ {
			case INTEGER_DATA:
				return nil
			case FLOAT_DATA:
				return fmt.Errorf("cannot assign %s to %q of type %s",
					dTyp[DataFloat], c0.Data.(string), dTyp[c0.Entry.DataTyp])
			case IDENTIFIER_DATA:
				if c1.Entry.DataTyp == c0.Entry.DataTyp {
					return nil
				}
				return fmt.Errorf("cannot assign %s to %q of type %s",
					dTyp[c1.Entry.DataTyp], c0.Data.(string), dTyp[c0.Entry.DataTyp])
			case EXPRESSION:
				if typ, err := c1.validateExpr(); err == nil {
					if typ == c0.Entry.DataTyp {
						return nil
					}
					return fmt.Errorf("cannot assign %s to %q of type %s",
						dTyp[typ], c0.Data.(string), dTyp[c0.Entry.DataTyp])
				} else {
					return err
				}
			}
		case DataFloat:
			// Accept both integer and float.
			switch c1.Typ {
			case INTEGER_DATA, FLOAT_DATA:
				return nil
			case IDENTIFIER_DATA:
				if c1.Entry.DataTyp == DataFloat || c1.Entry.DataTyp == DataInteger {
					return nil
				}
				return fmt.Errorf("cannot assign %s to %q of type %s",
					dTyp[c1.Entry.DataTyp], c0.Data.(string), dTyp[c0.Entry.DataTyp])
			case EXPRESSION:
				if typ, err := c1.validateExpr(); err == nil {
					if typ == DataInteger || typ == DataFloat {
						return nil
					}
					return fmt.Errorf("cannot assign %s to %q of type %s",
						dTyp[typ], c0.Data.(string), dTyp[c0.Entry.DataTyp])
				} else {
					return err
				}
			}
		default:
			return fmt.Errorf("variable %q has unsupported data type", c0.Entry.Name)
		}
	case EXPRESSION:
		if _, err := n.validateExpr(); err != nil {
			return err
		}
	default:
		for _, e1 := range n.Children {
			if err := e1.validate(target); err != nil {
				return err
			}
		}
	}
	return nil
}

// validateExpr validates an expression and returns its resulting datatype.
// If the expression is illegal, an error is returned.
func (n *Node) validateExpr() (dataType, error) {
	if n.Data == nil {
		// TODO: add validation for function call parameters.
		// FUNCTION call.
		name := n.Children[0].Data.(string)
		if f, ok := Global.Get(name); ok {
			return f.DataTyp, nil
		} else {
			return 0, fmt.Errorf("undeclared function %q", name)
		}
	}

	switch len(n.Children) {
	case 2:
		c0 := n.Children[0]
		c1 := n.Children[1]
		var c0t, c1t dataType

		// Set operand 1 type.
		switch c0.Typ {
		case IDENTIFIER_DATA:
			c0t = c0.Entry.DataTyp
		case FLOAT_DATA:
			c0t = DataFloat
		case INTEGER_DATA:
			c0t = DataInteger
		case EXPRESSION:
			var err error
			if c0t, err = c0.validateExpr(); err != nil {
				return c0t, err
			}
		}

		// Set operand 2 type.
		switch c1.Typ {
		case IDENTIFIER_DATA:
			c1t = c1.Entry.DataTyp
		case FLOAT_DATA:
			c1t = DataFloat
		case INTEGER_DATA:
			c1t = DataInteger
		case EXPRESSION:
			var err error
			if c1t, err = c1.validateExpr(); err != nil {
				return c1t, err
			}
		}

		// Validate both operands and expression.
		op := 0 // Index based on expression operator.
		switch n.Data.(string) {
		case "+":
			op = opPlus
		case "-":
			op = opMinus
		case "*":
			op = opMul
		case "/":
			op = opDiv
		case "%":
			op = opMod
		case "|":
			op = opOr
		case "&":
			op = opAnd
		case "^":
			op = opXor
		case "<<":
			op = opLsh
		case ">>":
			op = opRsh
		case ">":
			op = opGt
		case "<":
			op = opLt
		case "=":
			op = opEq
		}

		// Use lookup table to quickly determine compatibility.
		if !lut[c0t][c1t][op] {
			return 0, fmt.Errorf("illegal expression: %s %s %s on line %d:%d",
				dTyp[c0t], n.Data.(string), dTyp[c1t], n.Line, n.Pos)
		}

		// Set result data type and return.
		if c0t == c1t {
			return c0t, nil
		}
		if c0t == DataFloat {
			return c0t, nil
		}
		return c1t, nil
	case 1:
		c0 := n.Children[0]
		if c0.Typ == FLOAT_DATA {
			return 0, fmt.Errorf("unary operator %s not deifned for data type %s",
				n.Data.(string), dTyp[DataFloat])
		}
		return DataInteger, nil
	}
	return 0, fmt.Errorf("malformed expression on line %d:%d", n.Line, n.Pos)
}
