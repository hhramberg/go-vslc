package arm

import (
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

// genPrint generates aarch64 assembler for a print statement. An error is returned if something went wrong.
// Prints are called by using the standard library printf function which must be linked.
func genPrint(n *ir.Node, rf *registerFile, wr *util.Writer, st *util.Stack) error {
	for i1, e1 := range n.Children[0].Children {
		// Move format string and item to print into registers x0 and x1.
		switch e1.Typ {
		case ir.STRING_DATA:
			// Load format string.
			wr.Write("\tadrp\t%s, %s\n", rf.regi[r0].String(), labelPrintfString)
			wr.Write("\tadd\t%s, %s, :lo12:%s\n", rf.regi[r0].String(), rf.regi[r0].String(), labelPrintfString)

			// Load string to print.
			wr.Write("\tadrp\t%s, %s\n", rf.regi[r1].String(), fmt.Sprintf("%s%d", labelString, e1.Data.(int)))
			wr.Write("\tadd\t%s, %s, :lo12:%s\n", rf.regi[r1].String(), rf.regi[r1].String(), fmt.Sprintf("%s%d", labelString, e1.Data.(int)))
		case ir.INTEGER_DATA:
			// Load format string.
			wr.Write("\tadrp\t%s, %s\n", rf.regi[r0].String(), labelPrintfInt)
			wr.Write("\tadd\t%s, %s, :lo12:%s\n", rf.regi[r0].String(), rf.regi[r0].String(), labelPrintfInt)

			// Load immediate into register.
			if err := genLoadImmToRegister(e1.Data.(int), &rf.regi[r1], wr); err != nil {
				return err
			}
		case ir.FLOAT_DATA:
			// Load format string.
			wr.Write("\tadrp\t%s, %s\n", rf.regi[r0].String(), labelPrintfFloat)
			wr.Write("\tadd\t%s, %s, :lo12:%s\n", rf.regi[r0].String(), rf.regi[r0].String(), labelPrintfFloat)

			// Create string for float constant and move to register v0.
			label := floatToGlobalString(e1.Data.(float64))
			wr.Write("\tldr\t%s, =%s\n", rf.regf[v0].String(), label)
		case ir.EXPRESSION:
			r, err := genExpression(e1, rf, wr, st)
			if err != nil {
				return err
			}

			if r.typ == i {
				// Load format string.
				wr.Write("\tadrp\t%s, %s\n", rf.regi[r0].String(), labelPrintfInt)
				wr.Write("\tadd\t%s, %s, :lo12:%s\n", rf.regi[r0].String(), rf.regi[r0].String(), labelPrintfInt)

				// Move constant.
				wr.Write("\tmov\t%s, %s\n", rf.regi[r1].String(), r.String())
			} else {
				// Load format string.
				wr.Write("\tadrp\t%s, %s\n", rf.regi[r0].String(), labelPrintfFloat)
				wr.Write("\tadd\t%s, %s, :lo12:%s\n", rf.regi[r0].String(), rf.regi[r0].String(), labelPrintfFloat)

				// Move constant.
				wr.Write("\tmov\t%s, %s\n", rf.regf[v0].String(), r.String())
			}
		case ir.IDENTIFIER_DATA:
			r, err := loadIdentifier(e1.Data.(string), rf, wr, st)
			if err != nil {
				return err
			}

			if r.typ == i {
				// Load format string.
				wr.Write("\tadrp\t%s, %s\n", rf.regi[r0].String(), labelPrintfInt)
				wr.Write("\tadd\t%s, %s, :lo12:%s\n", rf.regi[r0].String(), rf.regi[r0].String(), labelPrintfInt)

				// Move constant.
				wr.Write("\tmov\t%s, %s\n", rf.regi[r1].String(), r.String())
			} else {
				// Load format string.
				wr.Write("\tadrp\t%s, %s\n", rf.regi[r0].String(), labelPrintfFloat)
				wr.Write("\tadd\t%s, %s, :lo12:%s\n", rf.regi[r0].String(), rf.regi[r0].String(), labelPrintfFloat)

				// Move constant.
				wr.Write("\tmov\t%s, %s\n", rf.regf[v0].String(), r.String())
			}
		default:
			return fmt.Errorf("compiler error: expected node of type STRING_DATA, INTEGER_DATA, FLOAT_DATA, EXPRESSION or IDENTIFIER, got %s",
				e1.Type())
		}

		// Call printf.
		wr.Write("\tbl\tprintf\n")
		if i1 == len(n.Children[0].Children)-1 {
			// Last print item. Append newline.

			// Load format string.
			wr.Write("\tadrp\t%s, %s\n", rf.regi[r0].String(), labelPrintfNewline)
			wr.Write("\tadd\t%s, %s, :lo12:%s\n", rf.regi[r0].String(), rf.regi[r0].String(), labelPrintfNewline)

			// Call printf with only format string.
			wr.Write("\tbl\tprintf\n")
		}
	}
	return nil
}
