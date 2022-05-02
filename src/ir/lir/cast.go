package lir

import (
	"fmt"
	"vslc/src/ir/lir/types"
)

// ----------------------------
// ----- Type definitions -----
// ----------------------------

// CastInstruction defines an instruction that casts either types.Int to types.Float,
// or vice versa.
type CastInstruction struct {
	b   *Block         // b is the basic block element that owns this instruction.
	id  int            // id is the unique identifier of this instruction in function body.
	typ types.DataType // typ defines the resulting types.DataType that the instructions casts to.
	src Value          // src is the source Value that was cast.
	hw  interface{}    // hw defines the hardware register of the CastInstruction's virtual register.
	en  bool           // Set to true if instruction is enabled.
}

// ---------------------
// ----- CastInstructions -----
// ---------------------

// -------------------
// ----- Globals -----
// -------------------

// ---------------------
// ----- Functions -----
// ---------------------

// Id returns the unique id of the CastInstruction.
func (inst *CastInstruction) Id() int {
	return inst.id
}

// Name returns the textual representation of the virtual register Value of the CastInstruction.
func (inst *CastInstruction) Name() string {
	return fmt.Sprintf("cast%d", inst.id)
}

// Type returns the CastInstruction identifying this instruction as a CastInstruction.
func (inst *CastInstruction) Type() types.InstructionType {
	return types.CastInstruction
}

// DataType returns the DataType of the declared variable that was loaded.
func (inst *CastInstruction) DataType() types.DataType {
	return inst.typ
}

// String returns the textual LIR representation of the CastInstruction.
func (inst *CastInstruction) String() string {
	return fmt.Sprintf("%s = (%s)%s", inst.Name(), inst.typ.String(), inst.src.Name())
}

// SetHW panics for the CastInstruction, because it's a memory value, not a virtual register.
func (inst *CastInstruction) SetHW(hw interface{}) {
	inst.hw = hw
}

// GetHW returns <nil> for the CastInstruction.
func (inst *CastInstruction) GetHW() interface{} {
	return inst.hw
}

// Operand1 returns the source Value for the CastInstruction that was initially cast.
func (inst *CastInstruction) Operand1() Value {
	return inst.src
}

// Operand2 returns <nil> for the CastInstruction.
func (inst *CastInstruction) Operand2() Value {
	return nil
}

// Enable enables the instruction, resulting in that it will be printed using Module.String.
func (inst *CastInstruction) Enable() {
	inst.en = true
}

// Disable disables the instruction, resulting in that it won't be printed using Module.String.
func (inst *CastInstruction) Disable() {
	inst.en = false
}

// IsEnabled returns true if the isntruction is enabled.
func (inst *CastInstruction) IsEnabled() bool {
	return inst.en
}
