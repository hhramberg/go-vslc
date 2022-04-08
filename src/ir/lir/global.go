package lir

import (
	"fmt"
	"vslc/src/ir/lir/types"
)

// ----------------------------
// ----- Type definitions -----
// ----------------------------

// Global defines a LIR global variable. A global variable can be either int, float or string.
// strings are supported because strings must be globally available during assembly generation anyways.
type Global struct {
	id   int            // LIR sequence number.
	name string         // Optional name of global variable.
	typ  types.DataType // Data type of global variable.
	val  interface{}    // Value of variable.
	wr   interface{}    // Register Interference Graph (RIG) node wrapper.
}

// ---------------------
// ----- Constants -----
// ---------------------

// globalLabelPrefix defines the label prefix for global variables.
const globalLabelPrefix = "g"

// -------------------
// ----- globals -----
// -------------------

// ---------------------
// ----- functions -----
// ---------------------

// Id returns the assigned and unique identifier given to the Global g when it was created.
func (g Global) Id() int {
	return g.id
}

// Name returns the name string of the Global g if it was assigned curing g's creation. If no name was given, the
// name is equal to the global label concatenated with g's unique identifier.
func (g Global) Name() string {
	if len(g.name) > 0 {
		return g.name
	}
	return fmt.Sprintf("%s%d", globalLabelPrefix, g.id)
}

// Type returns the types.Global constant, as this is a globally declared data object.
func (g Global) Type() types.Type {
	return types.Global
}

// DataType returns the datatype of the global data object, either types.Int, types.Float or types.String.
func (g Global) DataType() types.DataType {
	return g.typ
}

// String returns a textual LIR string representation of the Global variable g.
func (g Global) String() string {
	if g.typ == types.String {
		return fmt.Sprintf("global %s: %s, %q", g.name, g.typ.String(), g.val.(string))
	}
	return fmt.Sprintf("global %s: %s", g.name, g.typ.String())
}

// Has2Operands returns false for Global, because globals don't have operands.
func (g Global) Has2Operands() bool {
	return false
}

// Has1Operand returns false for Global, because globals don't have operands.
func (g Global) Has1Operand() bool {
	return false
}

// GetOperand1 panics when called on Global, because globals aren't computed.
func (g Global) GetOperand1() Value {
	panic("global does not have operands")
}

// GetOperand2 panics when called on Global, because globals aren't computed.
func (g Global) GetOperand2() Value {
	panic("global does not have operands")
}

// SetHW doesn't do anything for Global, but is implemented to support interface Value.
func (g Global) SetHW(r interface{}) {
}

// GetHW returns nil, because Global resides in memory.
func (g Global) GetHW() interface{} {
	panic("global is a memory data object, it doesn't reside in registers")
}

// SetWrapper sets the wrapper for this instruction during register allocation.
func (g Global) SetWrapper(wr interface{}) {
	g.wr = wr
}

// GetWrapper sets the wrapper for this instruction during register allocation.
func (g Global) GetWrapper() interface{} {
	return g.wr
}

// IsConstant returns false for Global.
func (d *Global) IsConstant() bool {
	return false
}
