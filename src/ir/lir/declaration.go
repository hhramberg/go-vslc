package lir

import (
	"fmt"
	"vslc/src/ir/lir/types"
)

// ----------------------------
// ----- Type definitions -----
// ----------------------------

// DeclareInstruction defines a local variable allocation in memory.
type DeclareInstruction struct {
	b    *Block         // b is the basic block element that owns this instruction.
	id   int            // id is the unique identifier of this instruction in function body.
	seq  int            // seq is the unique sequence number given to the variable.
	name string         // name defines the optional name of the local variable.
	typ  types.DataType // typ defines the variable's data type.
	hw   interface{}
	en   bool // Set to true if instruction is enabled.
}

// ---------------------
// ----- Constants -----
// ---------------------

// labelDeclare is used when printing declarations without names.
const labelDeclare = "var"

// -------------------
// ----- Globals -----
// -------------------

// ---------------------
// ----- Functions -----
// ---------------------

// Id returns the unique id of the DeclareInstruction.
func (inst *DeclareInstruction) Id() int {
	return inst.id
}

// Name returns the textual representation of the virtual register Value of the DeclareInstruction.
func (inst *DeclareInstruction) Name() string {
	return inst.name
}

// Type returns the constant identifying this instruction as a DeclareInstruction.
func (inst *DeclareInstruction) Type() types.InstructionType {
	return types.DeclareInstruction
}

// DataType returns the DataType of the declared variable that was loaded.
func (inst *DeclareInstruction) DataType() types.DataType {
	return inst.typ
}

// String returns the textual LIR representation of the DeclareInstruction.
func (inst *DeclareInstruction) String() string {
	return fmt.Sprintf("declare %s: %s", inst.Name(), inst.typ.String())
}

// SetHW panics for the DeclareInstruction, because it's a memory value, not a virtual register.
func (inst *DeclareInstruction) SetHW(hw interface{}) {
	inst.hw = hw
}

// GetHW returns <nil> for the DeclareInstruction.
func (inst *DeclareInstruction) GetHW() interface{} {
	return inst.hw
}

// Operand1 returns <nil> for the DeclareInstruction.
func (inst *DeclareInstruction) Operand1() Value {
	return nil
}

// Operand2 returns <nil> for the DeclareInstruction.
func (inst *DeclareInstruction) Operand2() Value {
	return nil
}

// Enable enables the instruction, resulting in that it will be printed using Module.String.
func (inst *DeclareInstruction) Enable() {
	inst.en = true
}

// Disable disables the instruction, resulting in that it won't be printed using Module.String.
func (inst *DeclareInstruction) Disable() {
	inst.en = false
}

// IsEnabled returns true if the isntruction is enabled.
func (inst *DeclareInstruction) IsEnabled() bool {
	return inst.en
}

// Seq returns the declaration instruction's/variable's sequence id and unique positon on stack.
func (inst *DeclareInstruction) Seq() int {
	return inst.seq
}
