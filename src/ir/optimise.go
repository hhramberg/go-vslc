package ir

import (
	"errors"
	"fmt"
	"math/bits"
	"sync"
	"vslc/src/util"
)

// errs handles error reporting during parallel optimisation.
// Since errors can occur at multiple worker threads there may be multiple
// errors to report when main thread resumes control.
var errs struct {
	err []error    // Slice of errors.
	mx  sync.Mutex // For synchronising worker threads.
}

// Optimise applies optimisations to the parse tree starting at the root node.
func Optimise(opt util.Options) error {
	if opt.Threads > 1 {
		// Parallel.
		wg := sync.WaitGroup{} // Used for synchronising worker threads with main thread.

		// Flatten global list so that we can calculate the number of declared functions.
		Root.Children[0].paraPrepare()

		// Initiate worker threads.
		t := opt.Threads                    // Max number of threads to initiate.
		l := len(Root.Children[0].Children) // Number of functions defined in program.
		if t > l {
			t = l // Cannot launch more threads than functions.
		}
		n := l / t   // Number of jobs per worker thread.
		res := l % t // Residual work for res first threads.

		// Allocate memory for errors; one per worker thread.
		errs = struct {
			err []error
			mx  sync.Mutex
		}{err: make([]error, 0, t), mx: sync.Mutex{}}

		// Launch t threads.
		for i1 := 0; i1 < l; i1 += n {
			m := n
			if i1 < res {
				// Indicate that this worker thread should do one more job.
				m++
				i1++
			}
			wg.Add(1) // Tell main thread to wait for new thread to finish.
			go func(i, j int, wg *sync.WaitGroup) {
				defer wg.Done() // Alert main thread that this worker is done when returning.
				for i2 := 0; i2 < j; i2++ {
					if err := Root.Children[0].Children[i+i2].optimise(); err != nil {
						errs.mx.Lock()
						errs.err = append(errs.err, err)
						errs.mx.Unlock()
					}
				}
			}(i1, m, &wg)
		}

		// Wait for worker threads to finish.
		wg.Wait()

		// Check for errors.
		if len(errs.err) > 0 {
			return errors.New("multiple errors during parallel optimisation")
		}
	} else {
		// Sequential.
		if err := Root.optimise(); err != nil {
			return err
		}
	}
	Root.Print(0, true) // TODO: Delete.
	return nil
}

// Errors returns the slice of reported errors, if any, during parallel optimisation.
func Errors() []error {
	return errs.err
}

// paraPrepare eliminates the global list structure of the root node in preparation
// for the parallel optimisation run.
func (n *Node) paraPrepare() {
	if n.Typ != GLOBAL_LIST {
		return
	}

	// Recursively locate all global list nodes.
	for _, e1 := range n.Children {
		e1.paraPrepare()
	}

	// Flatten global list structure.
	n.flattenList()
}

// optimise starts the recursive optimisation process. This function must not be called
// by the parallel run form the root node.
func (n *Node) optimise() error {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("got error on node: %s\n", n.String())
		}
	}()
	// Traverse the subtree recursively.
	for _, e1 := range n.Children {
		if err := e1.optimise(); err != nil {
			return err
		}
	}

	// Look for optimisation option.
	switch n.Typ {
	case EXPRESSION_LIST, PRINT_LIST, VARIABLE_LIST, STATEMENT_LIST, GLOBAL_LIST, DECLARATION_LIST, ARGUMENT_LIST:
		n.flattenList()
	case TYPED_VARIABLE_LIST:
		// Move type data to this node and remove variable list.
		n.Data = n.Children[0].Data
		n.Children = n.Children[1].Children
	case DECLARATION:
		// Move type data to this node.
		n.Data = n.Children[0].Data
		n.Children = n.Children[1:]
	case EXPRESSION:
		if err := n.constantFolding(); err != nil {
			return err
		}
	case STATEMENT, PRINT_ITEM, GLOBAL:
		n.deleteLonelyNode()
	}
	return nil
}

// constantFolding eliminates arithmetic expressions that consists of only constant values.
func (n *Node) constantFolding() error {
	if n.Typ != EXPRESSION {
		return nil
	}

	if len(n.Children) == 2 {
		// Binary operators.
		c0 := n.Children[0]
		c1 := n.Children[1]

		// Check for two integers expression.
		if c0.Typ == INTEGER_DATA && c1.Typ == INTEGER_DATA {
			// Both operands are integer constants.
			a := c0.Data.(int)
			b := c1.Data.(int)
			var res int
			switch n.Data.(string) {
			case "+":
				res = a + b
			case "-":
				res = a - b
			case "*":
				res = a * b
			case "/":
				if b == 0 {
					return fmt.Errorf("line %d:%d: expression %d / %d not allowed: cannot divide by zero",
						c1.Line, c1.Pos, a, b)
				}
				res = a / b
			case "%":
				if b == 0 {
					return fmt.Errorf("line %d:%d: expression %d %% %d not allowed: cannot divide by zero",
						c1.Line, c1.Pos, a, b)
				}
				res = a % b
			case "&":
				res = a & b
			case "|":
				res = a | b
			case "^":
				res = a ^ b
			case ">>":
				res = a >> b
			case "<<":
				res = a << b
			}
			*n = *(c0)
			n.Data = res
			return nil
		}

		// Check for two float expression.
		if c0.Typ == FLOAT_DATA && c1.Typ == FLOAT_DATA {
			// Both operands are floating point constants.
			a := c0.Data.(float32)
			b := c1.Data.(float32)
			var res float32
			switch n.Data.(string) {
			case "+":
				res = a + b
			case "-":
				res = a - b
			case "*":
				res = a * b
			case "/":
				if b == 0.0 {
					return fmt.Errorf("line %d:%d: expression %f / %f not allowed: cannot divide by zero",
						c1.Line, c1.Pos, a, b)
				}
				res = a / b
			default:
				return fmt.Errorf("line %d:%d: binary operator %s not defined for %s",
					n.Line, n.Pos, n.Data.(string), dTyp[DataFloat])
			}
			*n = *c0
			n.Data = res
			return nil
		}

		// Check for first operand is integer.
		if c0.Typ == INTEGER_DATA {
			// First operator is an integer constant.
			switch c1.Typ {
			case FLOAT_DATA:
				a := float32(c0.Data.(int))
				b := c1.Data.(float32)
				var res float32
				// These optimisations will leave the result of the expression as float.
				switch n.Data.(string) {
				case "+":
					res = a + b
				case "-":
					res = a - b
				case "*":
					res = a * b
				case "/":
					if b == 0.0 {
						return fmt.Errorf("line %d:%d: expression %d / %f not allowed: cannot divide by zero",
							n.Line, n.Pos, c0.Data.(int), b)
					}
					res = a / b
				default:
					return fmt.Errorf("line %d:%d: operator %s not defined for %s and %s",
						n.Line, n.Pos, n.Data.(string), dTyp[DataInteger], dTyp[DataFloat])
				}
				*n = *c1
				n.Data = res
			case IDENTIFIER_DATA:
				// Identifier data may be bool or float, but is caught in symbol table validation.
				// These optimisations do not require knowing the type of the identifier.
				switch n.Data.(string) {
				case "*":
					switch c0.Data.(int) {
					case 1:
						// Multiply by 1: set result to other operand.
						*n = *(c1)
					case 0:
						// Multiply by 0: set result to zero.
						*n = *(c0)
					}
				case "|":
					// OR by 0: set result to other operand.
					if c0.Data.(int) == 0 {
						*n = *(c1)
					}
				case "&":
					// AND by 0: set result to zero.
					if c0.Data.(int) == 0 {
						*n = *(c1)
						n.Data = 0
					}
				}
			default:
				return fmt.Errorf("line %d:%d: operation %s not defined for %s and unknown",
					n.Line, n.Pos, n.Data.(string), dTyp[DataInteger])
			}
			return nil
		}

		// Check for second operand is integer.
		if c1.Typ == INTEGER_DATA {
			// Second operator is a constant.
			// Replace multiply and division with left and right shift if possible.
			switch c0.Typ {
			case FLOAT_DATA:
				a := c0.Data.(float32)
				b := float32(c1.Data.(int))
				var res float32
				switch n.Data.(string) {
				case "+":
					res = a + b
				case "-":
					res = a - b
				case "*":
					res = a * b
				case "/":
					if b == 0.0 {
						return fmt.Errorf("line %d:%d: expression %d / %f not allowed: cannot divide by zero",
							n.Line, n.Pos, c0.Data.(int), b)
					}
					res = a / b
				default:
					return fmt.Errorf("line %d:%d: operator %s not defined for %s and %s",
						n.Line, n.Pos, n.Data.(string), dTyp[DataFloat], dTyp[DataInteger])
				}
				*n = *c0
				n.Data = res
			case IDENTIFIER_DATA:
				switch n.Data.(string) {
				case "*":
					if c1.Data.(int) == 1 {
						// Multiplication by identity integer.
						*n = *(c0)
					} else if b := bits.OnesCount(c1.Data.(uint)); b == 1 {
						// Multiplication by integer that is power of 2.
						n.Data = "<<"
						c1.Data = b
					} else if b == 2 && c1.Data.(uint)&0x1 == 0x1 && c0.Typ == IDENTIFIER_DATA {
						// Operator op1 is a power of 2 plus one.
						//
						// This i helpful when a = b * c, where
						// b is an IDENTIFIER
						// c is 9 for example, where 9 = 8 + 1
						// Which gives: (b << 3) + b

						// Create a new expression.
						exp := Node{
							Typ:      EXPRESSION,
							Line:     n.Line,
							Pos:      n.Pos,
							Data:     "+",
							Entry:    nil,
							Children: make([]*Node, 2),
						}

						// Adjust original expression.
						n.Data = "<<"
						c1.Data = b - 1

						// Node n is the set as first child of new expression.
						ex0 := *n
						exp.Children[0] = &ex0

						// Result of first child is added to the result of the shift.
						ex1 := *c0

						// Second child is added to the results of the ex0 expression.
						exp.Children[1] = &ex1

						// Set exp as the new Node n.
						*n = exp
					}
				case "/":
					if c1.Data.(int) == 1 {
						// Division by identity integer.
						*n = *(c0)
					} else if b := bits.OnesCount(c1.Data.(uint)); b == 1 {
						// Division by integer that is power of 2.
						n.Data = ">>"
						c1.Data = b
					} else if b == 2 && c1.Data.(uint)&0x1 == 0x1 && c0.Typ == IDENTIFIER_DATA {
						// Operator op1 is a power of 2 plus one.
						//
						// This i helpful when a = b / c, where
						// b is an IDENTIFIER
						// c is 9 for example, where 9 = 8 + 1
						// Which gives: (b >> 3) - b

						// Create a new expression.
						exp := Node{
							Typ:      EXPRESSION,
							Line:     n.Line,
							Pos:      n.Pos,
							Data:     "-",
							Entry:    nil,
							Children: make([]*Node, 2),
						}

						// Adjust original expression.
						n.Data = ">>"
						c1.Data = b - 1

						// Node n is the set as first child of new expression.
						ex0 := *n
						exp.Children[0] = &ex0

						// Result of first child is added to the result of the shift.
						ex1 := *c0

						// Second child is added to the results of the ex0 expression.
						exp.Children[1] = &ex1

						// Set exp as the new Node n.
						*n = exp
					}
				case "%":
					if c1.Data.(int) == 1 {
						*n = *(c0)
					}
				case "|":
					if c1.Data.(int) == 0 {
						*n = *(c0)
					}
				case "&":
					if c1.Data.(int) == 0 {
						*n = *(c0)
						n.Data = 0
					}
				}
			default:
				return fmt.Errorf("line %d:%d: operation %s not defined for unknown and %s",
					n.Line, n.Pos, n.Data.(string), dTyp[DataInteger])
			}
		}
	}

	// Unary operators.
	if len(n.Children) == 1 {
		if n.Data == nil {
			*n = *(n.Children[0])
		} else if n.Children[0].Typ == INTEGER_DATA {
			// Unary operators.
			switch n.Data.(string) {
			case "-":
				data := -(n.Data.(int))
				*n = *(n.Children[0])
				n.Data = data
			case "~":
				data := int(bits.Reverse(n.Children[0].Data.(uint)))
				*n = *(n.Children[0])
				n.Data = data
			default:
				return fmt.Errorf("unary operatior %s not defined for %s", n.Data.(string), dTyp[DataInteger])
			}
		} else if n.Children[0].Typ == FLOAT_DATA {
			return fmt.Errorf("unary operatior %s not defined for %s", n.Data.(string), dTyp[DataFloat])
		}
	}

	return nil
}

// flattenList eliminates recursive list structures such that one list Node has one or more elements
// and not one element and possible one recursive list element.
func (n *Node) flattenList() {
	if len(n.Children) < 2 {
		return
	}
	c := n.Children[0].Children
	e := n.Children[1]
	n.Children = make([]*Node, 0, len(c)+1)
	n.Children = append(n.Children, c...)
	n.Children = append(n.Children, e)
}

// deleteLonelyNode removes nodes that have a single child and puts the contents
// of the child into the current node. Does not delete node if node holds data.
func (n *Node) deleteLonelyNode() {
	if len(n.Children) != 1 && n.Data != nil {
		return
	}
	*n = *(n.Children[0])
}
