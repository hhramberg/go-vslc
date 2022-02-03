// print.go provides functions for generating print statement assembly.

package riscv

import (
	"fmt"
	"math"
	"vslc/src/ir"
	"vslc/src/util"
)

// genPrint generates a print statement recursively. A error is returned if something went wrong.
func genPrint(n *ir.Node, f *ir.Symbol, wr *util.Writer, st *util.Stack, rf *registerFile) error {
	wr.Write("# Print statement begin.\n") // TODO: delete.

	fl := float32(12.34)
	fmt.Println(floatToChars(fl))

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
		case ir.FLOAT_DATA:
			// TODO: Continue.
			// Hack the stack and save a0 to stack without changing the sp.
			wr.Write("\tfsw\t%s, %d(%s)\n", regi[fa0], -4, regi[sp])

			wr.Write("\tflw\t%s, %s%d\n", regi[fa0], labelFloat, e1.Data.(int)) // Move constant to register.

			// Restore stack.
			wr.Write("\tflw\t%s, %d(%s)\n", regi[fa0], -4, regi[sp])
		case ir.INTEGER_DATA:
			// Hack the stack and save a0 to stack without changing the sp.
			wr.Write("\tsw\t%s, %d(%s)\n", regi[t0], -4, regi[sp])

			wr.Write("\tmv\t%s, %d\n", regi[t0], e1.Data.(int)) // Move constant to register.
			wr.Write("\tmv\t%s, %d\n", regi[a0], stdout)        // Print to stdout.

			// Restore stack.
			wr.Write("\tlw\t%s, %d(%s)\n", regi[t0], -4, regi[sp])
		case ir.EXPRESSION:
			if r, err := genExpression(n, f, wr, st, rf); err != nil {
				return err
			} else {
				if r.typ == integer {
					// Integer.
				} else {
					// Floating point.
				}
			}
		case ir.IDENTIFIER_DATA:
		default:
			return fmt.Errorf("can't print item of type %s", e1.String())
		}
	}
	wr.Write("# Print statement end.\n") // TODO: delete.
	return nil
}

// intToChars converts an integer to a byte stream of ASCII characters.
func intToChars(i int) string {
	var res []byte
	var sign bool
	if i < 0 {
		sign = true
		res = make([]byte, 20) // (2^64) - 1 is ~ 1,9e19 = 20 characters at most.
		i = -i
	} else {
		res = make([]byte, 21) // Add one for sign.
	}
	base := 10
	end := &(res[len(res)-1])
	r := end
	i1 := len(res) - 1
	for ; i1 >= 0 && i != 0; i1-- {
		*r = byte((i % base) + '0')
		r = &(res[i1-1])
		i /= base
	}

	if sign {
		res[i1] = '-'
		i1--
	}

	return string(res[i1+1:])
}

// floatToChars converts a float to a byte stream of ASCII characters.
func floatToChars(f float32) string {
	res := make([]byte, 32) // float32 has 4-decimal precision.
	var sign bool
	if f < 0 {
		sign = true
		f = -f
	}

	ip := int(f)           // Integer part.
	fp := f - float32(ip)  // Float part.
	istr := intToChars(ip) // Convert integer part to string.
	if sign {
		copy(res[1:], istr) // Copy into result, but preserve one space for sign bit.
		res[0] = '-'
	} else {
		copy(res, istr) // Copy into result.
	}
	res[len(istr)] = '.' // Add decimal point.

	i1 := len(istr)+1 // Position of first decimal.
	fp = fp * float32(math.Pow(10, 4)) // 4-digit precision.
	fstr := intToChars(int(f))
	copy(res[i1:], fstr)
	i1 += len(fstr)
	return string(res[:i1])
}
