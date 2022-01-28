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
	case util.Riscv:
		return riscv.GenRiscv(opt)
	default:
		return errors.New("unsupported output architecture")
	}
}
