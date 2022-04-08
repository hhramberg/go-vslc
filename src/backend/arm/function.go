package arm

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
// - Use register file LRU to assign registers.
// - Generate function body.
// - De-allocate stack.
// - Return x0 for integer functions, use v0 for floating point functions.
func genFunction(fun *ir.Symbol, wr *util.Writer, opt util.Options) error {
	// Verify input symbol.
	if fun == nil {
		return errors.New("function symbol table entry is <nil>")
	}
	if fun.Typ != ir.SymFunc {
		return errors.New("symbol table entry is not a function")
	}
	if fun.Node == nil {
		return errors.New("function syntax tree entry is <nil>")
	}
	if fun.Node.Typ != ir.FUNCTION {
		return fmt.Errorf("expected syntax tree node FUNCTION, got %s", fun.Node.Type())
	}

	// Write function name label.
	wr.Write("\n")
	wr.Label(fun.Name)

	// Calculate new stack size.
	sa := wordSize * (fun.Nparams + fun.Nlocals + 2) // Stack adjust. Accommodate all local variables, params and FP + LR.
	spill := sa % stackAlign
	if spill != 0 {
		sa += stackAlign - spill
	}

	// Adjust stack and set stack frame pointer.
	wr.Write("\tsub\t%s, %s, #%d\n", regi[sp], regi[sp], sa)

	// Save old frame pointer and link register.
	wr.Write("\tstp\t%s, %s, [%s, #%d]\n", regi[fp], regi[lr], regi[sp], sa-(wordSize<<1))

	// Set frame pointer to old stack  pointer.
	wr.Write("\tadd\t%s, %s, #%d\n", regi[fp], regi[sp], sa)

	ii := 0 // Number of integer parameters.
	fi := 0 // Number of float parameters.

	// Put arguments on stack.
	offset := -(wordSize * 3) // Offset by 3: 2 for skipping old SP and LR, one to align with current word.
	for _, e1 := range fun.Params {
		if e1.DataTyp == i {
			// Integer parameter.
			if ii > paramReg {
				// Load from stack, store on stack. Reuse x0, because argument passed in x0 is stored on stack by this point.
				wr.Write("\tldr\t%s, [%s, #%d]\n", regi[r0], regi[fp], wordSize*e1.Seq)
				wr.Write("\tstr\t%s, [%s, #%d]\n", regi[r0], regi[fp], offset)
			} else {
				// Store directly on stack from register.
				wr.Write("\tstr\t%s, [%s, #%d]\n", regi[r0+ii], regi[fp], offset)
			}
			ii++
		} else {
			// Float parameter.
			if fi > paramReg {
				// Load from stack, store on stack. Reuse v0, because argument passed in v0 is stored on stack by this point.
				wr.Write("\tldr\t%s, [%s, #%d]\n", regi[v0], regi[fp], wordSize*e1.Seq)
				wr.Write("\tstr\t%s, [%s, #%d]\n", regi[v0], regi[fp], offset)
			} else {
				// Store directly on stack from register.
				wr.Write("\tstr\t%s, [%s, #%d]\n", regi[v0+fi], regi[fp], offset)
			}
			fi++
		}
		offset -= wordSize
	}

	// Generate function body.
	rf := CreateRegisterFile()
	ls := util.Stack{} // Label stack for continue statement.
	st := util.Stack{} // Scope stack for local scopes.
	st.Push(&ir.Global.HT)
	st.Push(&fun.Locals.HT)
	if err := gen(fun.Node.Children[3], fun, &rf, wr, &st, &ls); err != nil {
		return err
	}
	st.Pop()
	st.Pop()

	return nil
}

// genReturn generates a function return statement. An error is returned if something went wrong.
func genReturn(n *ir.Node, fun *ir.Symbol, rf *registerFile, wr *util.Writer, st *util.Stack) error {
	// Generate return value.
	c1 := n.Children[0]
	var r *register

	// Generate return value.
	switch c1.Typ {
	case ir.INTEGER_DATA:
		r = &rf.regi[r0]
		if err := genLoadImmToRegister(c1.Data.(int), r, wr); err != nil {
			return err
		}
	case ir.FLOAT_DATA:
		r = &rf.regf[v0]
		label := floatToGlobalString(c1.Data.(float64))
		wr.Write("\tldr\t%s, =%s\n", r.String(), label)
	case ir.EXPRESSION:
		var err error
		if r, err = genExpression(c1, rf, wr, st); err != nil {
			return err
		}
	case ir.IDENTIFIER_DATA:
		var err error
		if r, err = loadIdentifier(c1.Data.(string), rf, wr, st); err != nil {
			return err
		}
	default:
		return fmt.Errorf("compiler error: expected node type of INTEGER_DATA, FLOAT_DATA, EXPRESSION or IDENTIFIER, got %s",
			c1.Type())
	}

	// Check if correct register index was assigned.
	if r.idx != r0 {
		if r.typ == i {
			wr.Write("\tmov\t%s, %s\n", rf.regi[r0].String(), r.String())
		} else {
			wr.Write("\tmov\t%s, %s\n", rf.regf[v0].String(), r.String())
		}
	}

	// Check if return value is of correct type.
	if r.typ != fun.DataTyp {
		if r.typ == i {
			// Cast integer to float.
			wr.Write("\tscvtf\t%s, %s\n", rf.regf[v0].String(), r.String())
		} else {
			// Cast float to integer.
			wr.Write("\tfcvts\t%s, %s\n", rf.regi[r0].String(), r.String())
		}
	}

	// Calculate allocated stack size.
	sa := wordSize * (fun.Nparams + fun.Nlocals + 2) // Stack adjust.
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
