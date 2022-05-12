package lir

import (
	"fmt"
	"vslc/src/ir/lir/types"
)

// ----------------------------
// ----- Type definitions -----
// ----------------------------

// Constant defines an integer or floating point constant
type Constant struct {
	b    *Block         // b is the basic block element that owns this instruction.
	id   int            // id is the unique identifier of this instruction in function body.
	name string         // name defines the optional name of the local variable.
	typ  types.DataType // typ defines the variable's data type.
	val  interface{}    // val holds the constant's data value.
	lseq int            // lseq holds the global data segment label sequence number of the Constant.
	used int            // used gets incremented every time the constant is loaded from the data segment.
	hw   interface{}    // Hardware register of the DataInstruction's virtual register.
	en   bool           // Set to true if instruction is enabled.
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

// Id returns the unique id of the Constant.
func (inst *Constant) Id() int {
	return inst.id
}

// Name returns the textual representation of the virtual register Value of the Constant.
func (inst *Constant) Name() string {
	return inst.name
}

// Type returns the constant identifying this instruction as a Constant.
func (inst *Constant) Type() types.InstructionType {
	return types.Constant
}

// DataType returns the DataType of the declared variable that was loaded.
func (inst *Constant) DataType() types.DataType {
	return inst.typ
}

// String returns the textual LIR representation of the Constant.
func (inst *Constant) String() string {
	if inst.typ == types.Int {
		return fmt.Sprintf("%s = %s(%d)", inst.Name(), inst.typ.String(), inst.val.(int))
	}
	return fmt.Sprintf("%s = %s(%f)", inst.Name(), inst.typ.String(), inst.val.(float64))
}

// SetHW panics for the Constant, because it's a memory value, not a virtual register.
func (inst *Constant) SetHW(hw interface{}) {
	inst.hw = hw
}

// GetHW returns <nil> for the Constant.
func (inst *Constant) GetHW() interface{} {
	return inst.hw
}

// Operand1 returns <nil> for the Constant.
func (inst *Constant) Operand1() Value {
	return nil
}

// Operand2 returns <nil> for the Constant.
func (inst *Constant) Operand2() Value {
	return nil
}

// Enable enables the instruction, resulting in that it will be printed using Module.String.
func (inst *Constant) Enable() {
	inst.en = true
}

// Disable disables the instruction, resulting in that it won't be printed using Module.String.
func (inst *Constant) Disable() {
	inst.en = false
}

// IsEnabled returns true if the isntruction is enabled.
func (inst *Constant) IsEnabled() bool {
	return inst.en
}

// Value returns either the int or float value of Constant inst.
func (inst *Constant) Value() interface{} {
	return inst.val
}

// GlobalSeq returns the globally assigned data sequence number of the constant.
func (inst *Constant) GlobalSeq() int {
	return inst.lseq
}

// Use increments the use counter of the Constant.
func (inst *Constant) Use() {
	inst.used++
}

// Used returns true if the Constant has been loaded.
func (inst *Constant) Used() bool {
	return inst.used > 0
}
