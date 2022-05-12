package arm

import (
	"fmt"
	"vslc/src/backend/regfile"
	"vslc/src/ir/lir"
	"vslc/src/ir/lir/types"
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

// genBranch generates aarch64 assembler of an LIR branch instruction. An error is returned if something went wrong.
func genBranch(v *lir.BranchInstruction, rf regfile.RegisterFile, wr *util.Writer, ls *util.Stack) error {
	if v.Else() == nil {
		// Unconditional branch.
		wr.Write("\tb\t%s\n", v.Then().Name())
		return nil
	}

	// Generate test.
	op1 := v.Operand1()
	op2 := v.Operand2()
	if op1.DataType() == types.Int {
		// Int compare.
		wr.Write("\tcmp\t%s, %s\n",
			op1.GetHW().(*lir.LiveNode).Reg.(regfile.Register).String(),
			op2.GetHW().(*lir.LiveNode).Reg.(regfile.Register).String())
	} else {
		// Float compare.
		wr.Write("\tfcmp\t%s, %s\n",
			op1.GetHW().(*lir.LiveNode).Reg.(regfile.Register).String(),
			op2.GetHW().(*lir.LiveNode).Reg.(regfile.Register).String())
	}

	// Generate jump to ELSE block if condition is false. THEN block follows jump instruction sequentially.
	switch v.Operator() {
	case types.Eq:
		// Jump if op1 != op2.
		wr.Write("\tb.ne\t%s\n", v.Else().Name())
	case types.Neq:
		// Jump if op1 == op2.
		wr.Write("\tb.eq\t%s\n", v.Else().Name())
	case types.LessThan:
		// Jump if op1 >= op2.
		wr.Write("\tb.ge\t%s\n", v.Else().Name())
	case types.LessThanOrEqual:
		// Jump if op1 > op2.
		wr.Write("\tb.gt\t%s\n", v.Else().Name())
	case types.GreaterThan:
		// Jump if op1 <= op2.
		wr.Write("\tb.le\t%s\n", v.Else().Name())
	case types.GreaterThanOrEqual:
		// Jump if op1 < op2.
		wr.Write("\tB.LT\t%s\n", v.Else().Name())
	default:
		return fmt.Errorf("unexpected logical operation: %d", v.Operator())
	}
	return nil
}
