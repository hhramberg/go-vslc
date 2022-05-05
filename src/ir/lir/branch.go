package lir

import (
	"fmt"
	"vslc/src/ir/lir/types"
)

// ----------------------------
// ----- Type definitions -----
// ----------------------------

// BranchInstruction defines an unconditional or conditional branch instruction.
type BranchInstruction struct {
	b        *Block                    // b is the basic block element that owns this instruction.
	id       int                       // id is the unique identifier of this instruction in function body.
	thn      *Block                    // thn is the target for unconditional and the target THEN block of conditional branches.
	els      *Block                    // els is the target for conditional ELSE block. Is <nil> for unconditional branches.
	op1, op2 Value                     // op1 and op2 are the Values to check compare for conditional branches. Is <nil> for unconditional branches.
	op       types.RelationalOperation // op defines the type of relation operation of conditional branch.
	hw       interface{}
	en       bool // Set to true if instruction is enabled.
}

// ReturnInstruction defines a return statement.
type ReturnInstruction struct {
	b   *Block // b is the basic block element that owns this instruction.
	id  int    // id is the unique identifier of this instruction in function body.
	val Value  // val is the returned value of the return statement.
	hw  interface{}
	en  bool // Set to true if instruction is enabled.
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

// Id returns the unique id of the BranchInstruction.
func (inst *BranchInstruction) Id() int {
	return inst.id
}

// Name returns the textual representation of the virtual register Value of the BranchInstruction.
func (inst *BranchInstruction) Name() string {
	return fmt.Sprintf("branch%d", inst.id)
}

// Type returns the BranchInstruction constant, identifying this instruction as a BranchInstruction.
func (inst *BranchInstruction) Type() types.InstructionType {
	return types.BranchInstruction
}

// DataType returns the DataType types.Unknown, because no result is generated for a branch instruction.
func (inst *BranchInstruction) DataType() types.DataType {
	return types.Unknown
}

// String returns the textual LIR representation of the BranchInstruction.
func (inst *BranchInstruction) String() string {
	if inst.els == nil {
		// Unconditional branch.
		return fmt.Sprintf("br %s", inst.thn.Name())
	}
	// Conditional branch.
	return fmt.Sprintf("br %s, %s, %s ? %s : %s", inst.op.String(), inst.op1.Name(), inst.op2.Name(), inst.thn.Name(), inst.els.Name())
}

// SetHW panics for the BranchInstruction, because it doesn't put the result in a new virtual register.
func (inst *BranchInstruction) SetHW(hw interface{}) {
	inst.hw = hw
}

// GetHW panics for the BranchInstruction.
func (inst *BranchInstruction) GetHW() interface{} {
	return inst.hw
}

// Operand1 returns the first operand of conditional BranchInstruction.
func (inst *BranchInstruction) Operand1() Value {
	return inst.op1
}

// Operand2 returns the second operand of conditional BranchInstruction.
func (inst *BranchInstruction) Operand2() Value {
	return inst.op2
}

// Enable enables the instruction, resulting in that it will be printed using Module.String.
func (inst *BranchInstruction) Enable() {
	inst.en = true
}

// Disable disables the instruction, resulting in that it won't be printed using Module.String.
func (inst *BranchInstruction) Disable() {
	inst.en = false
}

// IsEnabled returns true if the isntruction is enabled.
func (inst *BranchInstruction) IsEnabled() bool {
	return inst.en
}

// Operator returns the logical operator of BranchInstruction inst.
func (inst *BranchInstruction) Operator() types.RelationalOperation {
	return inst.op
}

// Then returns the then basic Block of BranchInstruction inst.
func (inst *BranchInstruction) Then() *Block {
	return inst.thn
}

// Else returns the else basic Block of BranchInstruction inst.
func (inst *BranchInstruction) Else() *Block {
	return inst.els
}

// Id returns the unique id of the ReturnInstruction.
func (inst *ReturnInstruction) Id() int {
	return inst.id
}

// Name returns the textual representation of the virtual register Value of the ReturnInstruction.
func (inst *ReturnInstruction) Name() string {
	return fmt.Sprintf("return%d", inst.id)
}

// Type returns the types.ReturnInstruction constant, identifying this instruction as a ReturnInstruction.
func (inst *ReturnInstruction) Type() types.InstructionType {
	return types.ReturnInstruction
}

// DataType returns the DataType of the returned value.
func (inst *ReturnInstruction) DataType() types.DataType {
	return inst.val.DataType()
}

// String returns the textual LIR representation of the ReturnInstruction.
func (inst *ReturnInstruction) String() string {
	return fmt.Sprintf("ret %s", inst.val.Name())
}

// SetHW panics for the ReturnInstruction, because it doesn't put the result in a new virtual register.
func (inst *ReturnInstruction) SetHW(hw interface{}) {
	inst.hw = hw
}

// GetHW returns <nil> for the ReturnInstruction.
func (inst *ReturnInstruction) GetHW() interface{} {
	return inst.hw
}

// Operand1 returns the return value of the ReturnInstruction.
func (inst *ReturnInstruction) Operand1() Value {
	return inst.val
}

// Operand2 panics for the ReturnInstruction.
func (inst *ReturnInstruction) Operand2() Value {
	return nil
}

// Enable enables the instruction, resulting in that it will be printed using Module.String.
func (inst *ReturnInstruction) Enable() {
	inst.en = true
}

// Disable disables the instruction, resulting in that it won't be printed using Module.String.
func (inst *ReturnInstruction) Disable() {
	inst.en = false
}

// IsEnabled returns true if the isntruction is enabled.
func (inst *ReturnInstruction) IsEnabled() bool {
	return inst.en
}


