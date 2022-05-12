package backend

import (
	"errors"
	"vslc/src/backend/arm"
	"vslc/src/ir"
	"vslc/src/ir/lir"
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

// GenerateAssembler takes the syntax tree and generates output assembler code
// based on architecture defined by opt.
func GenerateAssembler(opt util.Options, m *lir.Module, root *ir.Node) error {
	switch opt.TargetArch {
	case util.Aarch64:
		return arm.GenArm(opt, m, root)
	case util.Riscv64:
		//return riscv.GenRiscv(opt)
		return errors.New("RISC-V 64-bit not supported yet")
	case util.Riscv32:
		return errors.New("RISC-V 32-bit not supported")
	default:
		return errors.New("unsupported output architecture")
	}
}
