package lir

import (
	"fmt"
	"vslc/src/ir/lir/types"
)

// ----------------------------
// ----- Type definitions -----
// ----------------------------

// Global defines an LIR global variable.
type Global struct {
	m    *Module        // m is the Module that owns this Global.
	id   int            // id is the unique identifier of the global variable.
	name string         // name defines the unique string name of the global variable.
	typ  types.DataType // typ defines the data type of the global variable.
	hw   interface{}
	en   bool // Set to true if instruction is enabled.
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

// Id returns the unique id of the Global.
func (inst *Global) Id() int {
	return inst.id
}

// Name returns the textual representation of the virtual register Value of the Global.
func (inst *Global) Name() string {
	return inst.name
}

// Type returns the constant identifying this instruction as a Global variable.
func (inst *Global) Type() types.InstructionType {
	return types.Global
}

// DataType returns the DataType of the declared variable that was loaded.
func (inst *Global) DataType() types.DataType {
	return inst.typ
}

// String returns the textual LIR representation of the Global.
func (inst *Global) String() string {
	return fmt.Sprintf("%s: %s", inst.Name(), inst.typ.String())
}

// SetHW panics for the Global, because it's a memory value, not a virtual register.
func (inst *Global) SetHW(hw interface{}) {
	inst.hw = hw
}

// GetHW returns <nil> for the Global.
func (inst *Global) GetHW() interface{} {
	return inst.hw
}

// Operand1 returns <nil> for the Global.
func (inst *Global) Operand1() Value {
	return nil
}

// Operand2 returns <nil> for the Global.
func (inst *Global) Operand2() Value {
	return nil
}

// Enable enables the instruction, resulting in that it will be printed using Module.String.
func (inst *Global) Enable() {
	inst.en = true
}

// Disable disables the instruction, resulting in that it won't be printed using Module.String.
func (inst *Global) Disable() {
	inst.en = false
}

// IsEnabled returns true if the isntruction is enabled.
func (inst *Global) IsEnabled() bool {
	return inst.en
}
