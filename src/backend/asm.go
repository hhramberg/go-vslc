package backend

import (
	"errors"
	"vslc/src/backend/arm"
	"vslc/src/backend/riscv"
	"vslc/src/util"
)

// ----------------------------
// ----- Type definitions -----
// ----------------------------

// ---------------------
// ----- Constants -----
// ---------------------

// -------------------
// ----- Globals -----
// -------------------

// ---------------------
// ----- Functions -----
// ---------------------

// GenerateAssembler takes the syntax tree and generates output assembler code
// based on architecture defined by opt.
func GenerateAssembler(opt util.Options) error {
	switch opt.Target {
	case util.Aarch64:
		return arm.GenArm(opt)
	case util.Riscv64:
		return riscv.GenRiscv(opt)
	case util.Riscv32:
		return errors.New("RISC-V 32-bit not supported yet")
	default:
		return errors.New("unsupported output architecture")
	}
}
