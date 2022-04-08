package arm

import (
	"errors"
	"fmt"
)

import (
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

// genExpression generates aarch64 assembler for arithmetic expressions or function calls. An error is returned
// if something went wrong. The result is put in the register pointed to by the returned register pointer.
func genExpression(n *ir.Node, rf *registerFile, wr *util.Writer, st *util.Stack) (*register, error) {
	if n.Data == nil {
		// Function call.
		return genFunctionCall(n, rf, wr, st)
	}

	// Load operand 1.
	var op1, res *register
	switch n.Children[0].Typ {
	case ir.INTEGER_DATA:
		op1 = rf.lruI()
		wr.Write("\tmov\t%s, #%d\n", op1.String(), n.Children[0].Data.(int))
	case ir.FLOAT_DATA:
		l := floatToGlobalString(n.Children[0].Data.(float64))
		op1 = rf.lruF()
		wr.Write("\tldr\t%s, =%s\n", op1.String(), l)
	case ir.EXPRESSION:
		var err error
		op1, err = genExpression(n.Children[0], rf, wr, st)
		if err != nil {
			return nil, err
		}
	case ir.IDENTIFIER_DATA:
		var err error
		op1, err = loadIdentifier(n.Children[0].Data.(string), rf, wr, st)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("compiler error: expected node type of INTEGER_DATA, FLOAT_DATA, EXPRESSION or IDENTIFIER, got %s",
			n.Children[0].Type())
	}

	if len(n.Children) == 2 {
		// Binary expression.

		// Load operand 2.
		var op2 *register
		switch n.Children[1].Typ {
		case ir.INTEGER_DATA:
			op2 = rf.lruI()
			wr.Write("\tmov\t%s, #%d\n", op2.String(), n.Children[1].Data.(int))
		case ir.FLOAT_DATA:
			l := floatToGlobalString(n.Children[1].Data.(float64))
			op2 = rf.lruF()
			wr.Write("\tldr\t%s, =%s\n", op2.String(), l)
		case ir.EXPRESSION:
			var err error
			op2, err = genExpression(n.Children[1], rf, wr, st)
			if err != nil {
				return nil, err
			}
		case ir.IDENTIFIER_DATA:
			var err error
			op2, err = loadIdentifier(n.Children[1].Data.(string), rf, wr, st)
			if err != nil {
				return nil, err
			}
		default:
			return nil, fmt.Errorf("compiler error: expected node type of INTEGER_DATA, FLOAT_DATA, EXPRESSION or IDENTIFIER, got %s",
				n.Children[1].Type())
		}

		if op1.typ == i && op2.typ == op1.typ {
			// Can only assign to integer register if both operands are integers.
			res = rf.lruI()
		} else {
			res = rf.lruF()
		}

		// Choose instruction from operator.
		if res.typ == i {
			// Integer operations.
			switch n.Data.(string) {
			case "+":
				wr.Write("\tadd\t%s, %s, %s\n", res.String(), op1.String(), op2.String())
			case "-":
				wr.Write("\tsub\t%s, %s, %s\n", res.String(), op1.String(), op2.String())
			case "*":
				wr.Write("\tmul\t%s, %s, %s\n", res.String(), op1.String(), op2.String())
			case "/":
				// Signed division. Division by zero caught in validate.
				wr.Write("\tsdiv\t%s, %s, %s\n", res.String(), op1.String(), op2.String())
			case "%":
				// From: https://stackoverflow.com/questions/35351470/obtaining-remainder-using-single-aarch64-instruction
				// Also division by zero is caught in validate.
				wr.Write("\tudiv\t%s, %s, %s\n", res.String(), op1.String(), op2.String())
				wr.Write("\tmsub\t%s, %s, %s, %s\n", res.String(), res.String(), op2.String(), op1.String())
			case "&":
				wr.Write("\tand\t%s, %s, %s\n", res.String(), op1.String(), op2.String())
			case "^":
				wr.Write("\teor\t%s, %s, %s\n", res.String(), op1.String(), op2.String())
			case "|":
				wr.Write("\torr\t%s, %s, %s\n", res.String(), op1.String(), op2.String())
			case ">>":
				wr.Write("\tlsr\t%s, %s, %s\n", res.String(), op1.String(), op2.String())
			case "<<":
				wr.Write("\tlsl\t%s, %s, %s\n", res.String(), op1.String(), op2.String())
			default:
				return nil, fmt.Errorf("unexpected binary operator %q", n.Data.(string))
			}
		} else {
			switch n.Data.(string) {
			case "+":
				wr.Write("\tfadd\t%s, %s, %s\n", res.String(), op1.String(), op2.String())
			case "-":
				wr.Write("\tfsub\t%s, %s, %s\n", res.String(), op1.String(), op2.String())
			case "*":
				wr.Write("\tfmul\t%s, %s, %s\n", res.String(), op1.String(), op2.String())
			case "/":
				wr.Write("\tfdiv\t%s, %s, %s\n", res.String(), op1.String(), op2.String())
			default:
				return nil, fmt.Errorf("unexpected binary operator %q", n.Data.(string))
			}
		}
	} else {
		// Unary expression.
		switch n.Data.(string) {
		case "-":
			wr.Write("\tneg\t%s, %s\n", res.String(), op1.String())
		case "~":
			wr.Write("\tmvn\t%s, %s\n", res.String(), op1.String())
		default:
			return nil, fmt.Errorf("unexpected unary operator %q", n.Data.(string))
		}
	}
	return res, nil
}

// genFunctionCall generates aarch64 assembler for a function call. An error is returned if something went wrong. The
// result of the function call is put in the register pointed to by the returned register pointer.
func genFunctionCall(n *ir.Node, rf *registerFile, wr *util.Writer, st *util.Stack) (*register, error) {
	// Get function symbol from globals.
	gint := st.Get(st.Size())
	if gint == nil {
		return nil, errors.New("compiler error: scope stack is empty")
	}

	globals := gint.(*map[string]*ir.Symbol)
	fun, ok := (*globals)[n.Children[0].Data.(string)]
	if !ok {
		return nil, fmt.Errorf("undeclared function %q", n.Children[0].Data.(string))
	}

	// Check if we need to pass arguments on stack.
	nargs := len(n.Children[1].Children[0].Children) // Number of arguments.
	var stack int
	if nargs > paramReg {
		// Worst case is everything is same datatype.
		stack = nargs - paramReg
		spill := stack % stackAlign
		if spill != 0 {
			stack += stackAlign - spill
		}
		wr.Write("\tsub\t%s, %s, #%d\n", rf.SP().String(), rf.SP().String(), stack)
	}

	ii := 0 // Number of integer arguments.
	fi := 0 // Number of float arguments.

	args := n.Children[1].Children[0].Children
	if len(n.Children[1].Children[0].Children) != len(fun.Params) {
		return nil, fmt.Errorf("line %d:%d: expected %d arguments, got %d",
			fun.Node.Line, fun.Node.Pos, len(n.Children[2].Children[0].Children), len(fun.Params))
	}

	for i1, e1 := range args {
		// Expression list.
		param := fun.Params[i1]

		if param.DataTyp == i {
			// Require integer. Reject float.
			switch e1.Typ {
			case ir.INTEGER_DATA:
				if ii < paramReg {
					// Put constant in register.
					wr.Write("\tmov\t%s, #%d\n", regi[r0+ii], e1.Data.(int))
				} else {
					// Put constant on stack.
					tmp := rf.lruINoArg()
					wr.Write("\tmov\t%s, #%d\n", tmp, e1.Data.(int)) // Put immediate in register.
					// Store immediate on stack.
					pos := ii - paramReg
					if fi >= paramReg {
						pos += fi - paramReg
					}
					wr.Write("\tstr\t%s, [%s, #%d]\n", tmp.String(), rf.SP().String(), pos)
				}
				ii++
			case ir.FLOAT_DATA:
				// Should not put float as integer, but can be cast using Go compiler. This support exceeds the typed
				// VSL specification.
				data := int(e1.Data.(float64))
				if ii < paramReg {
					// Put argument in register.
					wr.Write("\tmov\t%s, #%d\n", regi[r0+ii], data)
				} else {
					// Put argument on stack.
					tmp := rf.lruINoArg()
					wr.Write("\tmov\t%s, #%d\n", tmp.String(), data) // Put immediate in register.
					// Store immediate on stack.
					pos := ii - paramReg
					if fi >= paramReg {
						pos += fi - paramReg
					}
					wr.Write("\tstr\t%s, [%s, #%d]\n", tmp.String(), rf.SP().String(), pos)
				}
				ii++
			case ir.EXPRESSION:
				arg, err := genExpression(e1, rf, wr, st)
				if err != nil {
					return nil, err
				}
				if arg.typ != param.DataTyp {
					// Should not accept, but include support beyond typed VSL. Cast float to integer.
					tmp := rf.lruINoArg()
					wr.Write("\tfcvtzs\t%s, %s\n", tmp.String(), arg.String())
					arg = tmp
				}
				if ii < paramReg {
					// Put argument in register.
					wr.Write("\tmov\t%s, %s\n", regi[r0+ii], arg.String())
				} else {
					// Put argument on stack.
					pos := ii - paramReg
					if fi >= paramReg {
						pos += fi - paramReg
					}
					wr.Write("\tstr\t%s, [%s, #%d]\n", arg.String(), rf.SP().String(), pos)
				}
				ii++
			case ir.IDENTIFIER_DATA:
				arg, err := loadIdentifier(e1.Data.(string), rf, wr, st)
				if err != nil {
					return nil, err
				}
				if arg.typ != param.DataTyp {
					// Should not accept, but include support beyond typed VSL. Cast float to integer.
					tmp := rf.lruINoArg()
					wr.Write("\tfcvtzs\t%s, %s\n", tmp.String(), arg.String())
					arg = tmp
				}
				if ii < paramReg {
					// Put argument in register.
					wr.Write("\tmov\t%s, %s\n", regi[r0+ii], arg.String())
				} else {
					// Put argument on stack.
					pos := ii - paramReg
					if fi >= paramReg {
						pos += fi - paramReg
					}
					wr.Write("\tstr\t%s, [%s, #%d]\n", arg.String(), rf.SP().String(), pos)
				}
				ii++
			default:
				return nil, fmt.Errorf("line %d:%d: unexpected function argument, got node type %s",
					e1.Line, e1.Pos, e1.Type())
			}
		} else {
			// Require float. Cast integer to float.
			switch e1.Typ {
			case ir.INTEGER_DATA:
				data := float64(e1.Data.(int)) // Cast constant to float.

				// Create float constant in data segment.
				floatStrings.Lock()
				idx := len(floatStrings.s)
				floatStrings.s = append(floatStrings.s, fmt.Sprintf("%x", data))
				floatStrings.Unlock()

				if fi < paramReg {
					// Put constant in register.

					// Load float constant from data segment.
					wr.Write("\tldr\t%s, =%s%d\n", regf[v0+fi], labelFloat, idx)
				} else {
					// Put constant on stack.
					tmp := rf.lruFNoArg()
					wr.Write("\tldr\t%s, =%s%d\n", tmp.String(), labelFloat, idx)
					pos := ii - paramReg
					if fi >= paramReg {
						pos += fi - paramReg
					}
					wr.Write("\tstr\t%s, [%s, #%d]\n", tmp.String(), rf.SP().String(), pos)
				}
				fi++
			case ir.FLOAT_DATA:
				floatStrings.Lock()
				idx := len(floatStrings.s)
				floatStrings.s = append(floatStrings.s, fmt.Sprintf("%x", e1.Data.(float64)))
				floatStrings.Unlock()

				if fi < paramReg {
					wr.Write("\tldr\t%s, =%s%d\n", regf[v0+fi], labelFloat, idx)
				} else {
					// Put constant on stack.
					tmp := rf.lruFNoArg()
					wr.Write("\tldr\t%s, =%s%d\n", tmp.String(), labelFloat, idx)
					pos := ii - paramReg
					if fi >= paramReg {
						pos += fi - paramReg
					}
					wr.Write("\tstr\t%s, [%s, #%d]\n", tmp.String(), rf.SP().String(), pos)
				}
				fi++
			case ir.EXPRESSION:
				arg, err := genExpression(e1, rf, wr, st)
				if err != nil {
					return nil, err
				}
				if arg.typ != param.DataTyp {
					// Cast integer to float.
					tmp := rf.lruF()
					wr.Write("\tscvtf\t%s, %s\n", tmp.String(), arg.String())
					arg = tmp
				}
				if fi < paramReg {
					// Put argument in register.
					wr.Write("\tmov\t%s, %s\n", regi[v0+fi], arg.String())
				} else {
					// Put argument on stack.
					pos := ii - paramReg
					if fi >= paramReg {
						pos += fi - paramReg
					}
					wr.Write("\tstr\t%s, [%s, #%d]\n", arg.String(), rf.SP().String(), pos)
				}
				fi++
			case ir.IDENTIFIER_DATA:
				arg, err := loadIdentifier(e1.Data.(string), rf, wr, st)
				if err != nil {
					return nil, err
				}
				if arg.typ != param.DataTyp {
					// Cast integer to float.
					tmp := rf.lruF()
					wr.Write("\tscvtf\t%s, %s\n", tmp.String(), arg.String())
					arg = tmp
				}
				if fi < paramReg {
					// Put argument in register.
					wr.Write("\tmov\t%s, %s\n", regi[v0+fi], arg.String())
				} else {
					// Put argument on stack.
					pos := ii - paramReg
					if fi >= paramReg {
						pos += fi - paramReg
					}
					wr.Write("\tstr\t%s, [%s, #%d]\n", arg.String(), rf.SP().String(), pos)
				}
				fi++
			default:
				return nil, fmt.Errorf("line %d:%d: unexpected function argument, got node type %s",
					e1.Line, e1.Pos, e1.Type())
			}
		}
	}

	// Call function.
	wr.Write("\tbl\t%s\n", fun.Name)

	// De-allocate stack for arguments, if any.
	if nargs > paramReg {
		stack = nargs - paramReg
		spill := stack % stackAlign
		if spill != 0 {
			stack += stackAlign - spill
		}
		wr.Write("\tadd\t%s, %s, #%d\n", rf.SP().String(), rf.SP().String(), stack)
	}

	// Return a pointer to the register with the function result (r0 for integers, v0 for floating point).
	if fun.DataTyp == i {
		return &(rf.regi[r0]), nil
	}
	return &(rf.regf[v0]), nil
}
