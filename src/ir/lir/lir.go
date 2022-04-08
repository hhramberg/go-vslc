// Package lir provides functions for transforming the syntax tree into the light intermediate representation.
package lir

import (
	"vslc/src/ir/lir/types"
)

// ----------------------------
// ----- Type definitions -----
// ----------------------------

// Value defines a three-address code operand.
type Value interface {
	Id() int                   // Unique identifier assigned to Value when it's created.
	Name() string              // Name of Value.
	Type() types.Type          // Type of instruction that writes to the Value.
	DataType() types.DataType  // Datatype of value (int or float).
	String() string            // LIR textual representation of Value.
	SetHW(interface{})         // Used by register allocation for setting physical register.
	GetHW() interface{}        // Used by backend to retrieve physical register.
	Has2Operands() bool        // Returns true if instruction if a three-address code; it has two dependencies.
	Has1Operand() bool         // Returns true if the instruction is a two-address code; it has a dependency.
	IsConstant() bool          // Returns true if the instruction is a constant value.
	GetOperand1() Value        // Returns dependency one for both three-address code and two-address code instructions.
	GetOperand2() Value        // Returns dependency two for three-address code instructions.
	SetWrapper(wr interface{}) // Set register allocation wrapper.
	GetWrapper() interface{}   // Retrieve register allocation wrapper.
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
