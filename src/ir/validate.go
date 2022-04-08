package ir

import (
	"errors"
	"fmt"
	"sync"
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

// -------------------
// ----- globals -----
// -------------------

// lutExp is the lookup table for binary expressions, relations and type compatibility.
// Dimensions
// 1 Datatype of operand 1.
// 2 Datatype of operand 2.
// 3 Operator.
var lutExp = [2][2][opEq + 1]bool{
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

// lutAssign is the lookup table for relations and type compatibility.
// Dimensions
// 1 Datatype of operand 1.
// 2 Datatype of operand 2.
var lutAssign = [2][2]bool{
	{
		// OP1 is an integer.
		true,  // int := int allowed.
		false, // int := float not allowed.
	},
	{
		// OP1 is a float.
		true, // float := int allowed.
		true, // float := float allowed.
	},
}

// ----------------------
// ----- functions ------
// ----------------------

// ValidateTree validates types for expressions and assignments.
func ValidateTree(opt util.Options) error {
	if opt.Threads > 1 {
		// Parallel.
		wg := sync.WaitGroup{} // Used for synchronising worker threads with main thread.

		// Initiate worker threads.
		t := opt.Threads  // Max number of threads to initiate.
		l := len(Funcs.F) // Number of functions defined in program.
		if t > l {
			t = l // Cannot launch more threads than functions.
		}
		n := l / t   // Number of jobs per worker thread.
		res := l % t // Residual work for res first threads.

		// Allocate memory for errors; one per worker thread.
		errs := util.NewPerror(t)

		// Launch t threads.
		for i1 := 0; i1 < l; i1 += n {
			m := n
			i := i1
			if i1 < res {
				// Indicate that this worker thread should do one more job.
				m++
				i1++
			}
			wg.Add(1) // Tell main thread to wait for new thread to finish.
			go func(i, j int, wg *sync.WaitGroup) {
				defer wg.Done() // Alert main thread that this worker is done when returning.

				st := util.Stack{} // Scope stack.
				st.Push(&Global)

				// Validate function body.
				for i2 := 0; i2 < j; i2++ {
					f := Funcs.F[i+i2]
					st.Push(&(f.Locals))
					if err := f.Node.validate(&st); err != nil {
						errs.Append(err)
					}

					// Deallocate stack. Can be omitted?
					st.Pop()
				}
				st.Pop()
			}(i, m, &wg)
		}

		// Wait for worker threads to finish.
		wg.Wait()

		// Check for errors.
		if errs.Len() > 0 {
			for e1 := range errs.Errors() {
				fmt.Println(e1)
			}
			return errors.New("multiple errors during parallel validation")
		}
	} else {
		// Sequential.
		st := util.Stack{} // Stack used for identifier lookup.
		st.Push(&Global)   // Push global symbol table on stack.
		for _, e1 := range Funcs.F {
			st.Push(&(e1.Locals))
			if err := e1.Node.validate(&st); err != nil {
				return err
			}
			st.Pop()
		}
		st.Pop()
	}
	return nil
}

// validate recursively checks expressions and assignments for type validation.
func (n *Node) validate(st *util.Stack) error {
	switch n.Typ {
	case ASSIGNMENT_STATEMENT:
		if err := n.validateAssign(st); err != nil {
			return err
		}
	case EXPRESSION:
		if _, err := n.validateExpression(st); err != nil {
			return err
		}
	case RELATION:
		if err := n.validateRelation(st); err != nil {
			return err
		}
	case BLOCK:
		if n.Entry != nil {
			// FUNCTION BLOCKs don't have Entry, because the entry is bound to the FUNCTION node.
			st.Push(&(n.Entry.Locals))
		}

		// Validate children of block.
		for _, e1 := range n.Children {
			if err := e1.validate(st); err != nil {
				return err
			}
		}
		if n.Entry != nil {
			st.Pop()
		}
	case STATEMENT_LIST:
		for i1, e1 := range n.Children[:len(n.Children)-1] {
			if e1.Typ == RETURN_STATEMENT && i1 != len(n.Children)-1 {
				// Can't have statements after return.
				return fmt.Errorf("line %d:%d: %s not allowed after %s",
					n.Children[i1+1].Line, n.Children[i1+1].Pos, nt[n.Children[i1+1].Typ], nt[e1.Typ])
			}
		}
		for _, e1 := range n.Children {
			if err := e1.validate(st); err != nil {
				return err
			}
		}
	default:
		for _, e1 := range n.Children {
			if err := e1.validate(st); err != nil {
				return err
			}
		}
	}
	return nil
}

// validateExpression validates an expression and returns its resulting datatype.
// If the expression is illegal, an error is returned.
func (n *Node) validateExpression(st *util.Stack) (int, error) {
	if n.Data == nil {
		// FUNCTION call.
		name := n.Children[0].Data.(string)
		if f, ok := Global.Get(name); ok {
			args := n.Children[1].Children[0].Children // List of parameters.
			if len(args) != f.Nparams {
				return 0, fmt.Errorf("function %q expects %d parameters, got %d at line %d:%d",
					f.Name, f.Nparams, len(args), n.Children[0].Line, n.Children[0].Pos)
			}

			if f.Nparams > 0 {
				params := f.Node.Children[2].Children // functions params: one or more typed variable list of indents.
				seq := 0
				for i1 := 0; i1 < len(params); i1++ {
					// For all typed variable lists in parameter list.
					for i2 := 0; i2 < len(params[i1].Children); i2++ {
						// For all identifiers in typed variable list.
						arg := args[seq]
						param := params[i1].Children[i2]

						switch arg.Typ {
						case EXPRESSION:
							if t, err := arg.validateExpression(st); err != nil {
								return 0, err
							} else {
								if t != param.Entry.DataTyp {
									return 0, fmt.Errorf("function %q parameter %d expects %s, got %s at line %d:%d",
										f.Name, seq+1, DTyp[param.Entry.DataTyp], DTyp[t], n.Children[0].Line, n.Children[0].Pos)
								}
							}
						case IDENTIFIER_DATA:
							var dt int
							if e, err := GetEntry(arg.Data.(string), st); err == nil {
								dt = e.DataTyp
							} else {
								return 0, fmt.Errorf("reference to identifier %q not found at line %d:%d: %s",
									arg.Data.(string), arg.Line, arg.Pos, err)
							}
							if param.Entry.DataTyp != dt {
								return 0, fmt.Errorf("function %q parameter %d expects %s, got %s at line %d:%d",
									f.Name, seq+1, DTyp[param.Entry.DataTyp], DTyp[arg.Entry.DataTyp], n.Children[0].Line, n.Children[0].Pos)
							}
						case INTEGER_DATA:
							if param.Entry.DataTyp != DataInteger {
								return 0, fmt.Errorf("function %q parameter %d expects %s, got %s at line %d:%d",
									f.Name, seq+1, DTyp[param.Entry.DataTyp], DTyp[DataInteger], n.Children[0].Line, n.Children[0].Pos)
							}
						case FLOAT_DATA:
							if param.Entry.DataTyp != DataFloat {
								return 0, fmt.Errorf("function %q parameter %d expects %s, got %s at line %d:%d",
									f.Name, seq+1, DTyp[param.Entry.DataTyp], DTyp[DataFloat], n.Children[0].Line, n.Children[0].Pos)
							}
						default:
							return 0, fmt.Errorf("unexpected node type in function call: %s", nt[arg.Typ])
						}

						seq++
					}
				}
			}
			return f.DataTyp, nil
		} else {
			return 0, fmt.Errorf("undeclared function %q", name)
		}
	}

	switch len(n.Children) {
	case 2:
		c0 := n.Children[0]
		c1 := n.Children[1]
		var c0t, c1t int

		// Set operand 1 type.
		switch c0.Typ {
		case IDENTIFIER_DATA:
			if e, err := GetEntry(c0.Data.(string), st); err != nil {
				return 0, fmt.Errorf("identifier %q not declated at line %d:%d",
					c0.Data.(string), c0.Line, c0.Pos)
			} else {
				c0t = e.DataTyp
			}
		case FLOAT_DATA:
			c0t = DataFloat
		case INTEGER_DATA:
			c0t = DataInteger
		case EXPRESSION:
			var err error
			if c0t, err = c0.validateExpression(st); err != nil {
				return c0t, err
			}
		}

		// Set operand 2 type.
		switch c1.Typ {
		case IDENTIFIER_DATA:
			if e, err := GetEntry(c1.Data.(string), st); err != nil {
				return 0, fmt.Errorf("identifier %q not declated at line %d:%d",
					c1.Data.(string), c1.Line, c1.Pos)
			} else {
				c1t = e.DataTyp
			}
		case FLOAT_DATA:
			c1t = DataFloat
		case INTEGER_DATA:
			c1t = DataInteger
		case EXPRESSION:
			var err error
			if c1t, err = c1.validateExpression(st); err != nil {
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
		default:
			return 0, fmt.Errorf("operator %q not defined for expression at line %d:%d",
				n.Data.(string), n.Line, n.Pos)
		}

		// Use lookup table to quickly determine compatibility.
		if !lutExp[c0t][c1t][op] {
			return 0, fmt.Errorf("illegal expression: %s %s %s on line %d:%d",
				DTyp[c0t], n.Data.(string), DTyp[c1t], n.Line, n.Pos)
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
			if n.Data.(string) != "-" {
				return 0, fmt.Errorf("unary operator %s not deifned for data type %s",
					n.Data.(string), DTyp[DataFloat])
			}
			return DataFloat, nil
		}
		return DataInteger, nil
	}
	return 0, fmt.Errorf("malformed expression on line %d:%d", n.Line, n.Pos)
}

// validateRelation validates a relation. If the relation is illegal, an error is returned.
func (n *Node) validateRelation(st *util.Stack) error {
	var dt1, dt2 int
	c1 := n.Children[0]
	c2 := n.Children[1]
	switch c1.Typ {
	case EXPRESSION:
		if dt, err := c1.validateExpression(st); err != nil {
			return err
		} else {
			dt1 = dt
		}
	case IDENTIFIER_DATA:
		if s, err := GetEntry(c1.Data.(string), st); err == nil {
			dt1 = s.DataTyp
		} else {
			return fmt.Errorf("identifier %q not declared, at line %d:%d",
				c1.Data.(string), c1.Line, c1.Pos)
		}
	case FLOAT_DATA:
		dt1 = DataFloat
	case INTEGER_DATA:
		dt1 = DataInteger
	}
	switch c2.Typ {
	case EXPRESSION:
		if dt, err := n.Children[0].validateExpression(st); err != nil {
			return err
		} else {
			dt2 = dt
		}
	case IDENTIFIER_DATA:
		if s, err := GetEntry(c2.Data.(string), st); err == nil {
			dt2 = s.DataTyp
		} else {
			return fmt.Errorf("identifier %q not declared, at line %d:%d",
				c2.Data.(string), c2.Line, c2.Pos)
		}
	case FLOAT_DATA:
		dt2 = DataFloat
	case INTEGER_DATA:
		dt2 = DataInteger
	}

	// Validate both operands and relation.
	op := 0 // Index based on expression operator.
	switch n.Data.(string) {
	case ">":
		op = opGt
	case "<":
		op = opLt
	case "=":
		op = opEq
	default:
		return fmt.Errorf("operator %q not defined for relation at line %d:%d", n.Data.(string), n.Line, n.Pos)
	}

	if !lutExp[dt1][dt2][op] {
		return fmt.Errorf("operator %s not defined for %s and %s at line %d:%d",
			n.Data.(string), DTyp[dt1], DTyp[dt2], c1.Line, c1.Pos)
	}
	return nil
}

// validateAssign validates an assignment statement. If the assignment is illegal, an error is returned.
func (n *Node) validateAssign(st *util.Stack) error {
	c1 := n.Children[0]
	c2 := n.Children[1]
	var c1t, c2t int

	if s, err := GetEntry(c1.Data.(string), st); err == nil {
		c1t = s.DataTyp
	} else {
		return fmt.Errorf("identifier %q not declared, at line %d:%d",
			c1.Data.(string), c1.Line, c1.Pos)
	}

	switch c2.Typ {
	case EXPRESSION:
		if dt, err := c2.validateExpression(st); err != nil {
			return err
		} else {
			c2t = dt
		}
	case IDENTIFIER_DATA:
		if s, err := GetEntry(c2.Data.(string), st); err == nil {
			c2t = s.DataTyp
		} else {
			return fmt.Errorf("identifier %q not declared, at line %d:%d",
				c2.Data.(string), c2.Line, c2.Pos)
		}
	case FLOAT_DATA:
		c2t = DataFloat
	case INTEGER_DATA:
		c2t = DataInteger
	}

	if !lutAssign[c1t][c2t] {
		return fmt.Errorf("cannot assign %s to variable %q, %s is not assignlable to %s at line %d:%d",
			DTyp[c2t], c1.Data.(string), DTyp[c2t], DTyp[c1t], c1.Line, c1.Pos)
	}
	return nil
}

// GetEntry retrieves a Symbol entry from the scope stack St.
func GetEntry(name string, st *util.Stack) (*Symbol, error) {
	for i1 := 0; i1 < st.Size(); i1++ {
		if s := st.Get(1 + i1).(*SymTab); s != nil {
			if e, ok := s.Get(name); ok {
				return e, nil
			}
		} else {
			return nil, fmt.Errorf("compiler error: scope stack malformed")
		}
	}
	return nil, fmt.Errorf("identifier %q not declared", name)
}
