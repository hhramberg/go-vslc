package lir

import (
	"fmt"
	"vslc/src/ir/lir/types"

	"strings"
)

// ----------------------------
// ----- Type definitions -----
// ----------------------------

// PrintInstruction defines an instruction that uses system calls to print a single Value to stdout.
type PrintInstruction struct {
	b   *Block // b is the basic block element that owns this instruction.
	id  int    // id is the unique identifier of this instruction in function body.
	val Value  // Value to print.
	hw  interface{}
	en  bool // Set to true if instruction is enabled.
}

// VaList defines a variable argument list.
type VaList struct {
	b    *Block      // b is the basic block element that owns this instruction.
	id   int         // id is the unique identifier of this instruction in function body.
	vars []Value     // Value slice of values that's passed in the VaList.
	hw   interface{} // hw defines the hardware register assigned to VaList.
	en   bool        // Set to true if instruction is enabled.
}

// ---------------------
// ----- Constants -----
// ---------------------

// labelPrint defines the print prefix for textual LIR.
const labelPrint = "print"

// -------------------
// ----- Globals -----
// -------------------

// ---------------------
// ----- Functions -----
// ---------------------

// Id returns the unique id of the PrintInstruction.
func (inst *PrintInstruction) Id() int {
	return inst.id
}

// Name returns the textual representation of the virtual register Value of the PrintInstruction.
func (inst *PrintInstruction) Name() string {
	return fmt.Sprintf("%s%d", labelPrint, inst.id)
}

// Type returns the constant identifying this instruction as a PrintInstruction.
func (inst *PrintInstruction) Type() types.InstructionType {
	return types.PrintInstruction
}

// DataType returns the DataType Unknown, because print doesn't asm new data.
func (inst *PrintInstruction) DataType() types.DataType {
	return types.Unknown
}

// String returns the textual LIR representation of the PrintInstruction.
func (inst *PrintInstruction) String() string {
	return fmt.Sprintf("%s %s", labelPrint, inst.Name())
}

// SetHW panics for the PrintInstruction, because it's a memory value, not a virtual register.
func (inst *PrintInstruction) SetHW(hw interface{}) {
	inst.hw = hw
}

// GetHW returns <nil> for the PrintInstruction.
func (inst *PrintInstruction) GetHW() interface{} {
	return inst.hw
}

// Operand1 returns <nil> for the PrintInstruction.
func (inst *PrintInstruction) Operand1() Value {
	return nil
}

// Operand2 returns <nil> for the PrintInstruction.
func (inst *PrintInstruction) Operand2() Value {
	return nil
}

// Enable enables the instruction, resulting in that it will be printed using Module.String.
func (inst *PrintInstruction) Enable() {
	inst.en = true
}

// Disable disables the instruction, resulting in that it won't be printed using Module.String.
func (inst *PrintInstruction) Disable() {
	inst.en = false
}

// IsEnabled returns true if the isntruction is enabled.
func (inst *PrintInstruction) IsEnabled() bool {
	return inst.en
}

// Id returns the unique id of the VaList.
func (inst *VaList) Id() int {
	return inst.id
}

// Name returns the textual representation of the virtual register Value of the VaList.
func (inst *VaList) Name() string {
	return fmt.Sprintf("%s%d", labelDataInstruction, inst.id)
}

// Type returns the constant identifying this instruction as a VaList.
func (inst *VaList) Type() types.InstructionType {
	return types.DataInstruction
}

// DataType returns the DataType of the VaList.
func (inst *VaList) DataType() types.DataType {
	return types.VaList
}

// String returns the textual LIR representation of the VaList.
func (inst *VaList) String() string {
	sb := strings.Builder{}
	sb.WriteString(inst.Name())
	sb.WriteString(" = va_list [")
	for i1, e1 := range inst.vars {
		sb.WriteString(e1.Name())
		if i1 < len(inst.vars)-1 {
			sb.WriteString(", ")
		}
	}
	sb.WriteRune(']')
	return sb.String()
}

// SetHW panics for the VaList, because it's a memory value, not a virtual register.
func (inst *VaList) SetHW(hw interface{}) {
	inst.hw = hw
}

// GetHW returns <nil> for the VaList.
func (inst *VaList) GetHW() interface{} {
	return inst.hw
}

// Operand1 returns <nil> for the VaList.
func (inst *VaList) Operand1() Value {
	return nil
}

// Operand2 returns <nil> for the VaList.
func (inst *VaList) Operand2() Value {
	return nil
}

// Enable enables the instruction, resulting in that it will be printed using Module.String.
func (inst *VaList) Enable() {
	inst.en = true
}

// Disable disables the instruction, resulting in that it won't be printed using Module.String.
func (inst *VaList) Disable() {
	inst.en = false
}

// IsEnabled returns true if the isntruction is enabled.
func (inst *VaList) IsEnabled() bool {
	return inst.en
}

// Values returns the values pointed to by the VaList.
func (inst *VaList) Values() []Value {
	return inst.vars
}
