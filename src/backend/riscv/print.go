// print.go provides functions for generating print statement assembly.

package riscv

import (
	"fmt"
	"vslc/src/backend/xtoa"
	"vslc/src/ir"
	"vslc/src/util"
)

// genPrint generates a print statement recursively. A error is returned if something went wrong.
func genPrint(n *ir.Node, f *ir.Symbol, wr *util.Writer, st *util.Stack, rf *registerFile) error {
	wr.Write("; ----- Print statement begin -----\n") // TODO: delete.

	for _, e1 := range n.Children[0].Children {
		// For every item to be printed.
		switch e1.Typ {
		case ir.STRING_DATA:
			// From: https://smist08.wordpress.com/2019/09/07/risc-v-assembly-language-hello-world/
			wr.Ins2imm("addi", regi[a0], regi[zero], stdout)                            // Print to stdout.
			wr.Write("\tla\t%s, %s%d\n", regi[a1], labelString, e1.Data.(int))          // Load address of string constant.
			wr.Ins2imm("addi", regi[a2], regi[zero], len(ir.Strings.St[e1.Data.(int)])) // Length of string to print.
			wr.Ins2imm("addi", regi[a3], regi[zero], sysWrite)                          // System call for write. TODO: validate for 64/32-bit, may be different.
			wr.Write("\tecall\n")                                                       // Call the system call write.
		case ir.INTEGER_DATA, ir.FLOAT_DATA:
			var chars string
			if e1.Typ == ir.INTEGER_DATA {
				chars = xtoa.ItoA(e1.Data.(int))
			} else {
				chars = xtoa.FtoA(ir.Floats.Ft[e1.Data.(int)])
			}

			// Calculate character buffer and stack alignment.
			sa := len(chars)
			if sa%stackAlign != 0 {
				// Align stack.
				sa += stackAlign - sa%stackAlign
			}

			// Allocate stack for character buffer.
			wr.Ins2imm("addi", regi[sp], regi[sp], -sa)

			// Store all characters in buffer.
			for i2, e2 := range chars {
				idx := sa + i2 - len(chars)
				wr.Write("\tli\t%s, %d\n", regi[t0], e2)
				wr.Write("\tsb\t%s, %d(%s)\n", regi[t0], idx, regi[sp])
			}

			// Set up write system call.
			buf := sa - len(chars)
			wr.Write("\tli\t%s, %d\n", regi[a7], sysWrite)            // Syscall write.
			wr.Write("\tli\t%s, %d\n", regi[a0], stdout)              // Write to stdout.
			wr.Write("\taddi\t%s, %s, %d\n", regi[a1], regi[sp], buf) // Address of first character.
			wr.Write("\tli\t%s, %d\n", regi[a2], len(chars))          // Length of character stream.
			wr.Write("\tecall\n")                                     // Call write.

			// De-allocate stack for character buffer.
			wr.Ins2imm("addi", regi[sp], regi[sp], sa)
		case ir.EXPRESSION, ir.IDENTIFIER_DATA:
			var r *register
			var err error
			if e1.Typ == ir.EXPRESSION {
				if r, err = genExpression(n, f, wr, st, rf); err != nil {
					return err
				}
			} else {
				r = rf.loadIdentifierToReg(e1.Data.(string), f, wr, st)
			}

			if r.typ == integer {
				// Integer.

				// TODO: Optimise register usage. Registers are statically assigned for this print statement.

				// STACK layout:
				//
				// BOTTOM
				//
				// buf [13:22]
				// buf[4:13]
				// 0x00, 0x00, 0x00, sign-byte, buf[0:4] - From 0 trough 3 (Go slice terminology).
				// p (pointer to current position in buffer)
				// i (the integer to print)
				// a1 Preserved
				// a0 Preserved
				//
				// TOP <--- SP

				// The below procedure is effectively the intToChars function defined in this file
				// compiled with GodBolt for RISC-V.

				// Allocate stack space for buffer. Maximum character length is 21 with sign bit.
				// Need space for character buffer (21), sign byte (1), end and current pointer, base and registers a0 and a1.
				buf := 22 + (wordSize * 5)
				if buf%stackAlign != 0 {
					buf += stackAlign - buf%stackAlign
				}

				// Positions of variables, relative to sp.
				begin := buf - 21 // Beginning of buffer on stack.
				p := wordSize * 4 // Position on stack of variable p.
				i := wordSize * 3 // Position on stack of number to stringify.
				sign := buf - 22  // Position on stack of sign byte.
				base := 10

				// Allocate more stack space.
				wr.Ins2imm("addi", regi[sp], regi[sp], -buf)

				// Preserve a0 by saving it to stack top of stack.
				wr.Write("\t%s\t%s, %d(%s)\n", store, regi[a0], wordSize, regi[sp])
				// Preserve a1 by saving it to stack.
				wr.Write("\t%s\t%s, %d(%s)\n", store, regi[a1], wordSize<<1, regi[sp])

				wr.Write("\tli\t%s, %d\n", regi[a0], base) // Base = 10.

				// Store integer to print on stack.
				wr.Write("\t%s\t%s, %d(%s)\n", store, r.String(), i, regi[sp])

				// Buffer[21] is at address[sp] + buf.

				// Character pointer p, currently set to point to end of Buffer.
				wr.Ins2imm("addi", regi[a0], regi[sp], p)
				wr.Write("\t%s\t%s, %d(%s)\n", store, regi[a0], wordSize*4, regi[sp])

				// Sign is byte is directly after the character buffer: address[Buffer[21]] + 1.
				wr.Write("\tli\t%s, %d\n", regi[a0], 0)
				wr.Write("\tsb\t%s, %d(%s)\n", regi[a0], sign, regi[sp])

				// Check sign bit.
				lsign := util.NewLabel(util.LabelIfEnd)
				wr.LoadStore(load, regi[a0], i, regi[sp])
				wr.Write("\tli\t%s, %d\n", regi[a1], 0) // Load 0 to a1 for comparison with integer in a0.
				wr.Write("\tbge\t%s, %s, %s\n", regi[a0], regi[a1], lsign)
				// IF-THEN: set sign byte true.
				wr.Write("\tli\t%s, %d\n", regi[a0], 1)
				wr.Write("\tsb\t%s, %d(%s)\n", regi[a0], begin-1, regi[sp])

				// Set number positive.
				wr.LoadStore(load, regi[a0], i, regi[sp])
				wr.Write("\tli\t%s, %d\n", regi[a1], i)
				wr.Ins3("mul", regi[a0], regi[a0], regi[a1])
				wr.LoadStore(store, regi[a0], i, regi[sp])

				// ELSE:
				wr.Label(lsign)

				// Iterate over i and create string.
				lloophead := util.NewLabel(util.LabelWhileHead)

				// *p = (i % base) + '0';

				// First iteration of DO-WHILE, to catch i == 0.
				wr.Label(lloophead)
				wr.LoadStore(load, regi[a0], i, regi[sp])
				wr.Write("\tli\t%s, %d\n", regi[a1], base)
				wr.Ins3("rem", regi[a0], regi[a0], regi[a1]) // set a0 = i % base.
				wr.Ins2imm("addi", regi[a0], regi[a0], '0')  // Add '0' to a0.
				wr.LoadStore(load, regi[a1], p, regi[sp])    // Load the pointer p.
				wr.LoadStore("sb", regi[a0], 0, regi[a1])    // Set *p = a0.

				// p--;
				//wr.LoadStore(load, regi[a0], p, regi[sp])
				// p is already in a1.
				wr.Ins2imm("addi", regi[a1], regi[a1], -1)
				wr.LoadStore(store, regi[a1], p, regi[sp])

				// i /= base;
				wr.LoadStore(load, regi[a0], i, regi[sp])
				wr.Write("\tli\t%s, %d\n", regi[s1], base)
				wr.Ins3("div", regi[a0], regi[a0], regi[a1])
				wr.LoadStore(store, regi[a0], i, regi[sp])

				// Start loop.
				// i is already in a0.
				wr.Ins2imm("addi", regi[a1], regi[zero], 0)
				wr.Write("\tbne\t%s, %s, %s\n", regi[a1], regi[a2], lloophead)

				// Check for signbit.
				lprint := util.NewLabel(util.LabelIfEnd)
				wr.LoadStore("lb", regi[a0], sign, regi[sp])
				wr.Write("\tli\t%s, %d\n", regi[a1], 0)
				wr.Write("\tbeq\t%s, %s, %s\n", regi[a0], regi[a1], lprint)

				// Set sign.
				wr.LoadStore(load, regi[a1], p, regi[sp])
				wr.Write("\tli\t%s, %d\n", regi[a0], '-')
				wr.Write("\tsb\t%s, %d(%s)\n", regi[a0], 0, regi[a1])
				wr.Ins2imm("addi", regi[a1], regi[a1], 1)
				wr.LoadStore(store, regi[a1], p, regi[sp])

				wr.Label(lprint)

				// Set length of string in a2 = p - sp.
				wr.LoadStore(load, regi[a0], p, regi[sp])
				wr.Ins3("sub", regi[a2], regi[a0], regi[sp])

				// Write integer from buffer to stdout.
				wr.Write("\tli\t%s, %d\n", regi[a7], sysWrite)            // Syscall write.
				wr.Write("\tli\t%s, %d\n", regi[a0], stdout)              // Write to stdout.
				wr.Write("\taddi\t%s, %s, %d\n", regi[a1], regi[sp], buf) // Address of first character.
				//wr.Write("\tli\t%s, %d\n", regi[a2], len(chars))          // Length of character stream.
				wr.Write("\tecall\n") // Call write.

				// Restore a0 and a1 from stack.
				wr.Write("\t%s\t%s, %d(%s)\n", load, regi[a0], wordSize, regi[sp])
				wr.Write("\t%s\t%s, %d(%s)\n", load, regi[a1], wordSize<<1, regi[sp])

				// De-allocate temporary stack.
				wr.Ins2imm("addi", regi[sp], regi[sp], buf)

				// ----------------------------
				// ... or use printf.

				// See: https://web.eecs.utk.edu/~smarz1/courses/ece356/notes/assembly/

				// Call printf from standard library.

				// Load address of static global variable for string "%f" into a0.
				//wr.Write("\tla\t%s, _STR_printf_i\n", regi[a0])

				// Move float to write into a1.
				//wr.Write("\tmv\t%s, %s\n", regi[a1], r.String())

				// Call printf.
				//wr.Write("\tcall\tprintf\n")
			} else {
				// Floating point.
				// See: https://web.eecs.utk.edu/~smarz1/courses/ece356/notes/assembly/

				// Call printf from standard library.

				// Load address of static global variable for string "%f" into a0.
				wr.Write("\tla\t%s, _STR_printf_f\n", regi[a0])

				// Move float to write into fa0.
				wr.Write("\tmv\t%s, %s\n", regi[fa0], r.String())

				// Call printf.
				wr.Write("\tcall\tprintf\n")
			}
		default:
			return fmt.Errorf("can't print item of type %s", e1.String())
		}
	}
	wr.Write("; ----- Print statement end -----\n") // TODO: delete.
	return nil
}
