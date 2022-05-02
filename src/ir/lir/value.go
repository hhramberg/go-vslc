// Package ir provides functions and data structures for creating LIR instructions.
package lir

import (
	"vslc/src/ir/lir/types"
)

// ----------------------------
// ----- Type definitions -----
// ----------------------------

// Value defines an LIR virtual register value. A Value is an instruction that generates a virtual register.
// Instructions that asm Value types are all the arithmetic instructions, load instruction and function call
// instruction.
type Value interface {
	Id() int
	Name() string
	Type() types.InstructionType
	DataType() types.DataType
	String() string
	SetHW(hw interface{})
	GetHW() interface{}
	Operand1() Value
	Operand2() Value
	Enable()
	Disable()
	IsEnabled() bool
}

// ---------------------
// ----- Constants -----
// ---------------------

// -------------------
// ----- Globals -----
// -------------------

// ---------------------
// ----- Functions -----
// ---------------------
