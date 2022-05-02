package lir

import (
	"fmt"
	"vslc/src/ir/lir/types"
)

// ----------------------------
// ----- Type definitions -----
// ----------------------------

// LoadInstruction defines a load instruction that loads the data from a global variable, a parameter or a locally
// declared variable. Loading a string equals loading the pointer value of the first byte of the string.
type LoadInstruction struct {
	b   *Block      // b is the basic block element that owns this instruction.
	id  int         // id is the unique identifier of this instruction in function body.
	src Value       // src defines the variable to load. Either global, param or local.
	hw  interface{} // Hardware register of the LoadInstruction's virtual register.
	en  bool        // Set to true if instruction is enabled.
}

// StoreInstruction defines a store instruction that saves the contents of a virtual register to a memory allocated
// variable. A variable may be a global variable, local variable or function parameter.
type StoreInstruction struct {
	b   *Block // b is the basic block element that owns this instruction.
	id  int    // id is the unique identifier of this instruction in function body.
	src Value  // src defines the virtual register to save from.
	dst Value  // dst defines the variable to store to. Either global, param or local.
	hw  interface{}
	en  bool // Set to true if instruction is enabled.
}

// ---------------------
// ----- Constants -----
// ---------------------

// labelLoad is the prefix for load instructions.
const labelLoad = "load"

// labelStore is the prefix for store instructions.
const labelStore = "store"

// -------------------
// ----- Globals -----
// -------------------

// ---------------------
// ----- Functions -----
// ---------------------

// Id returns the unique id of the LoadInstruction.
func (inst *LoadInstruction) Id() int {
	return inst.id
}

// Name returns the textual representation of the virtual register Value of the LoadInstruction.
func (inst *LoadInstruction) Name() string {
	return fmt.Sprintf("%s%d", labelDataInstruction, inst.id)
}

// Type returns the constant identifying this instruction as a LoadInstruction.
func (inst *LoadInstruction) Type() types.InstructionType {
	return types.LoadInstruction
}

// DataType returns the DataType of the declared variable that was loaded.
func (inst *LoadInstruction) DataType() types.DataType {
	return inst.src.DataType()
}

// String returns the textual LIR representation of the LoadInstruction.
func (inst *LoadInstruction) String() string {
	return fmt.Sprintf("%s = %s %s", inst.Name(), labelLoad, inst.src.Name())
}

// SetHW sets the LoadInstruction's assigned hardware register during register allocation.
func (inst *LoadInstruction) SetHW(hw interface{}) {
	inst.hw = hw
}

// GetHW retrieves the LoadInstruction's assigned hardware register.
func (inst *LoadInstruction) GetHW() interface{} {
	return inst.hw
}

// Operand1 returns the first operand of the LoadInstruction inst.
func (inst *LoadInstruction) Operand1() Value {
	return inst.src
}

// Operand2 returns <nil> for the LoadInstruction, because it does not have two operands.
func (inst *LoadInstruction) Operand2() Value {
	return nil
}

// Enable enables the instruction, resulting in that it will be printed using Module.String.
func (inst *LoadInstruction) Enable() {
	inst.en = true
}

// Disable disables the instruction, resulting in that it won't be printed using Module.String.
func (inst *LoadInstruction) Disable() {
	inst.en = false
}

// IsEnabled returns true if the isntruction is enabled.
func (inst *LoadInstruction) IsEnabled() bool {
	return inst.en
}

// Id returns the unique id of the StoreInstruction.
func (inst *StoreInstruction) Id() int {
	return inst.id
}

// Name returns the textual representation of the virtual register Value of the StoreInstruction.
func (inst *StoreInstruction) Name() string {
	return fmt.Sprintf("%s%d", labelStore, inst.id)
}

// Type returns the constant identifying this instruction as a StoreInstruction.
func (inst *StoreInstruction) Type() types.InstructionType {
	return types.StoreInstruction
}

// DataType returns the constant identifying this instruction's datatype as types.Unknown.
func (inst *StoreInstruction) DataType() types.DataType {
	return inst.dst.DataType()
}

// String returns the textual LIR representation of the StoreInstruction.
func (inst *StoreInstruction) String() string {
	return fmt.Sprintf("%s %s, %s", labelStore, inst.src.Name(), inst.dst.Name())
}

// SetHW panics the StoreInstruction because it operates on existing virtual registers only.
func (inst *StoreInstruction) SetHW(hw interface{}) {
	inst.hw = hw
}

// GetHW return <nil> the StoreInstruction.
func (inst *StoreInstruction) GetHW() interface{} {
	return inst.hw
}

// Operand1 returns the destination variable of the StoreInstruction inst.
func (inst *StoreInstruction) Operand1() Value {
	return inst.src
}

// Operand2 returns the source virtual register of the StoreInstruction.
func (inst *StoreInstruction) Operand2() Value {
	return inst.dst
}

// Enable enables the instruction, resulting in that it will be printed using Module.String.
func (inst *StoreInstruction) Enable() {
	inst.en = true
}

// Disable disables the instruction, resulting in that it won't be printed using Module.String.
func (inst *StoreInstruction) Disable() {
	inst.en = false
}

// IsEnabled returns true if the isntruction is enabled.
func (inst *StoreInstruction) IsEnabled() bool {
	return inst.en
}
