package lir

import (
"fmt"
"vslc/src/ir/lir/types"
)

// ----------------------------
// ----- Type definitions -----
// ----------------------------

// PreserveInstruction defines an instruction that casts either types.Int to types.Float,
// or vice versa.
type PreserveInstruction struct {
	b   *Block         // b is the basic block element that owns this instruction.
	id  int            // id is the unique identifier of this instruction in function body.
	src Value          // src is the source Value that was preserve.
	hw  interface{}    // hw defines the hardware register of the PreserveInstruction's virtual register.
	en  bool           // Set to true if instruction is enabled.
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

// Id returns the unique id of the PreserveInstruction.
func (inst *PreserveInstruction) Id() int {
	return inst.id
}

// Name returns the textual representation of the virtual register Value of the PreserveInstruction.
func (inst *PreserveInstruction) Name() string {
	return fmt.Sprintf("%s%d", labelDataInstruction, inst.id)
}

// Type returns the PreserveInstruction identifying this instruction as a PreserveInstruction.
func (inst *PreserveInstruction) Type() types.InstructionType {
	return types.PreserveInstruction
}

// DataType returns the DataType of the declared variable that was loaded.
func (inst *PreserveInstruction) DataType() types.DataType {
	return inst.src.DataType()
}

// String returns the textual LIR representation of the PreserveInstruction.
func (inst *PreserveInstruction) String() string {
	return fmt.Sprintf("%s = %s", inst.Name(), inst.src.Name())
}

// SetHW panics for the PreserveInstruction, because it's a memory value, not a virtual register.
func (inst *PreserveInstruction) SetHW(hw interface{}) {
	inst.hw = hw
}

// GetHW returns <nil> for the PreserveInstruction.
func (inst *PreserveInstruction) GetHW() interface{} {
	return inst.hw
}

// Operand1 returns the source Value for the PreserveInstruction that was initially cast.
func (inst *PreserveInstruction) Operand1() Value {
	return inst.src
}

// Operand2 returns <nil> for the PreserveInstruction.
func (inst *PreserveInstruction) Operand2() Value {
	return nil
}

// Enable enables the instruction, resulting in that it will be printed using Module.String.
func (inst *PreserveInstruction) Enable() {
	inst.en = true
}

// Disable disables the instruction, resulting in that it won't be printed using Module.String.
func (inst *PreserveInstruction) Disable() {
	inst.en = false
}

// IsEnabled returns true if the isntruction is enabled.
func (inst *PreserveInstruction) IsEnabled() bool {
	return inst.en
}

