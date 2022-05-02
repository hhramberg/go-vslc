package lir

import (
	"fmt"
	"vslc/src/ir/lir/types"
)

// ----------------------------
// ----- Type definitions -----
// ----------------------------

// String defines an LIR String variable.
type String struct {
	m   *Module // m is the Module that owns this String.
	id  int     // id is the unique identifier of the String variable.
	val string  // val holds the value of the string constant.
	hw  interface{}
	en  bool // Set to true if instruction is enabled.
}

// StringPointer defines a word sized address pointer to a C-style null-terminated character array.
type StringPointer struct {
	m  *Module // m is the Module that owns this String.
	id int     // id is the unique identifier of the String variable.
	en bool    // Set to true if instruction is enabled.
}

// ---------------------
// ----- Constants -----
// ---------------------

// -------------------
// ----- Strings -----
// -------------------

// ---------------------
// ----- Functions -----
// ---------------------

// Id returns the unique id of the String.
func (inst *String) Id() int {
	return inst.id
}

// Name returns the textual representation of the virtual register Value of the String.
func (inst *String) Name() string {
	return fmt.Sprintf("%s%d", labelString, inst.id)
}

// Type returns the constant identifying this instruction as a String variable.
func (inst *String) Type() types.InstructionType {
	return types.Global
}

// DataType returns the DataType of the declared variable that was loaded.
func (inst *String) DataType() types.DataType {
	return types.String
}

// String returns the textual LIR representation of the String.
func (inst *String) String() string {
	return fmt.Sprintf("%s (%s): %q", inst.Name(), inst.DataType(), inst.val)
}

// SetHW panics for the String, because it's a memory value, not a virtual register.
func (inst *String) SetHW(hw interface{}) {
	inst.hw = hw
}

// GetHW returns <nil> for the String.
func (inst *String) GetHW() interface{} {
	return inst.hw
}

// Operand1 returns <nil> for the String.
func (inst *String) Operand1() Value {
	return nil
}

// Operand2 returns <nil> for the String.
func (inst *String) Operand2() Value {
	return nil
}

// Enable enables the instruction, resulting in that it will be printed using Module.String.
func (inst *String) Enable() {
	inst.en = true
}

// Disable disables the instruction, resulting in that it won't be printed using Module.String.
func (inst *String) Disable() {
	inst.en = false
}

// IsEnabled returns true if the instruction is enabled.
func (inst *String) IsEnabled() bool {
	return inst.en
}

// Value returns the string literal of the string Value.
func (inst *String) Value() string {
	return inst.val
}
