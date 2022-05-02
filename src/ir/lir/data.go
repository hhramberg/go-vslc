// Package lir provides structures and functions for creating Lightweight Intermediate Representation.
package lir

import (
	"fmt"
	"vslc/src/ir/lir/types"
)

// ----------------------------
// ----- Type definitions -----
// ----------------------------

// DataInstruction defines an arithmetic instruction that leaves the result in a new virtual register.
type DataInstruction struct {
	b        *Block                    // b is the basic block element that owns this instruction.
	id       int                       // id is the unique identifier of this instruction in function body.
	op       types.ArithmeticOperation // op defines the type of arithmetic operation of this instruction.
	hw       interface{}               // Hardware register of the DataInstruction's virtual register.
	op1, op2 Value                     // op1 and op2 holds the first and second operands respectively.
	en       bool                      // Set to true if instruction is enabled.
}

// ---------------------
// ----- Constants -----
// ---------------------

// labelDataInstruction defines the virtual register prefix for temporaries.
const labelDataInstruction = "%"

// -------------------
// ----- Globals -----
// -------------------

// expLut is the lookup table for expression compatibility.
var expLut = [2][2][types.Not+1]bool {
	{
		// First operand is types.Int.
		{
			// Second operand is types.Int.
			// Both operands are types.Int. Allow all arithmetic operators.
			true, // Add
			true, // Sub
			true, // Mul
			true, // Div
			true, // Rem
			true, // LShift
			true, // RShift
			true, // And
			true, // Xor
			true, // Or
			true, // Neg
			true, // Not
		},
		{
			// Second operand is types.Float.
			// At least one operand is types.Float. Only allow Add, Sub, Mul and Div.
			true, // Add
			true, // Sub
			true, // Mul
			true, // Div
			false, // Rem
			false, // LShift
			false, // RShift
			false, // And
			false, // Xor
			false, // Or
			false, // Neg
			false, // Not
		},
	},
	{
		// First operand is types.Float.
		{
			// Second operand is types.Int.
			// At least one operand is types.Float. Only allow Add, Sub, Mul and Div.
			true, // Add
			true, // Sub
			true, // Mul
			true, // Div
			false, // Rem
			false, // LShift
			false, // RShift
			false, // And
			false, // Xor
			false, // Or
			false, // Neg
			false, // Not
		},
		{
			// Second operand is types.Float.
			// At least one operand is types.Float. Only allow Add, Sub, Mul and Div.
			true, // Add
			true, // Sub
			true, // Mul
			true, // Div
			false, // Rem
			false, // LShift
			false, // RShift
			false, // And
			false, // Xor
			false, // Or
			false, // Neg
			false, // Not
		},
	},
}

// ---------------------
// ----- Functions -----
// ---------------------

// Id returns the unique identifier of the DataInstruction inst.
func (inst *DataInstruction) Id() int {
	return inst.id
}

// Name returns the LIR textual representation of DataInstruction inst's virtual register.
func (inst *DataInstruction) Name() string {
	return fmt.Sprintf("%s%d", labelDataInstruction, inst.id)
}

// Type returns types.DataInstruction for the DataInstruction type.
func (inst *DataInstruction) Type() types.InstructionType {
	return types.DataInstruction
}

// DataType returns the resulting types.DataType of the DataInstruction inst.
func (inst *DataInstruction) DataType() types.DataType {
	if inst.op >= types.Neg {
		return inst.op1.DataType()
	}
	if inst.op1.DataType() == inst.op2.DataType() {
		return inst.op1.DataType()
	} else {
		// If both operands are different the result is float.
		return types.Float
	}
}

// String returns the LIR textual representation of the DataInstruction inst.
func (inst *DataInstruction) String() string {
	if inst.op < types.Neg {
		return fmt.Sprintf("%s = %s %s, %s", inst.Name(), inst.op.String(), inst.op1.Name(), inst.op2.Name())
	}
	if inst.op <= types.Not {
		return fmt.Sprintf("%s = %s %s", inst.Name(), inst.op.String(), inst.op1.Name())
	}
	panic(fmt.Sprintf("DataInstruction %s has unexpected operand: %d", inst.Name(), inst.op))
}

// SetHW sets the DataInstruction's assigned hardware register during register allocation.
func (inst *DataInstruction) SetHW(hw interface{}) {
	inst.hw = hw
}

// GetHW retrieves the DataInstruction's assigned hardware register.
func (inst *DataInstruction) GetHW() interface{} {
	return inst.hw
}

// Operand1 returns the first operand of the DataInstruction inst.
func (inst *DataInstruction) Operand1() Value {
	return inst.op1
}

// Operand2 returns the second operand of the DataInstruction inst if it is a binary instruction. If it's a unary
// operation, <nil> is returned.
func (inst *DataInstruction) Operand2() Value {
	return inst.op2
}

// Enable enables the instruction, resulting in that it will be printed using Module.String.
func (inst *DataInstruction) Enable() {
	inst.en = true
}

// Disable disables the instruction, resulting in that it won't be printed using Module.String.
func (inst *DataInstruction) Disable() {
	inst.en = false
}

// IsEnabled returns true if the isntruction is enabled.
func (inst *DataInstruction) IsEnabled() bool {
	return inst.en
}

// Operator returns the arithmetic operator of the data instruction.
func (inst *DataInstruction) Operator() types.ArithmeticOperation {
	return inst.op
}
