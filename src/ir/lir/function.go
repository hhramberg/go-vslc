package lir

import (
	"fmt"
	"strings"
	"vslc/src/ir/lir/types"
)

// ----------------------------
// ----- Type definitions -----
// ----------------------------

// Function represents a function. It has a name, return datatype, parameters and instructions.
// using a function as a Value is the same as calling the function.
type Function struct {
	m         *Module        // Parent module. Used for requesting sequence numbers.
	id        int            // Unique identifier assigned to this function.
	name      string         // Optional name of function.
	typ       types.DataType // Return type of function.
	params    []*Param       // Parameters of function.
	variables []Value        // All declared variables in function body. Used for calculating stack.
	blocks    []*Block       // Basic blocks in function body.
	seq       int            // Sequence number for generating unique identifiers for all children of function.
}

// Param represents a function parameter. A parameter has a name and a datatype. Using a parameter as a Value
// is equal to using any other, non-global, memory allocated variable.
type Param struct {
	f    *Function      // Parent function.
	id   int            // Unique identifier of parameter.
	name string         // Optional name of parameter.
	typ  types.DataType // Data type of parameter.
	wr   interface{}    // Register Interference Graph (RIG) wrapper node.
}

// ---------------------
// ----- Constants -----
// ---------------------

// labelParamPrefix defines the Param types name prefix.
const labelParamPrefix = "p"

// -------------------
// ----- globals -----
// -------------------

// ---------------------
// ----- functions -----
// ---------------------

// ----------------------------
// ----- Function methods -----
// ----------------------------

// Id returns the unique sequence number assigned to Function f when it was created.
func (f *Function) Id() int {
	return f.id
}

// Name returns the given name of Function f, if any, or the assigned unique label and sequence id concatenation.
func (f *Function) Name() string {
	return f.name
}

// Type returns the types.Function data object type.
func (f *Function) Type() types.Type {
	return types.Function
}

// DataType returns the data type value of Function f, either types.Int or types.Float.
func (f *Function) DataType() types.DataType {
	return f.typ
}

// String returns the textual LIR representation of Function f.
func (f *Function) String() string {
	// Function header.
	sb := strings.Builder{}
	sb.WriteString(fmt.Sprintf("function %s(", f.name))
	for i1, e1 := range f.params {
		sb.WriteString(e1.String())
		if i1 < len(f.params)-1 {
			sb.WriteString(", ")
		}
	}
	sb.WriteString(fmt.Sprintf("): %s", f.DataType().String()))

	if len(f.blocks) > 0 {
		sb.WriteString(" {\n")
		// Function body.
		for _, e1 := range f.blocks {
			sb.WriteString(e1.String())
		}
		sb.WriteRune('}')
	}
	return sb.String()
}

// Blocks returns the basic blocks of Function f.
func (f *Function) Blocks() []*Block {
	return f.blocks
}

// CreateBlock creates a new Block for Function f.
func (f *Function) CreateBlock() *Block {
	b := &Block{
		f:            f,
		id:           f.m.getId(),
		term:         nil,
		instructions: make([]Value, 0, 16),
	}
	f.blocks = append(f.blocks, b)
	return b
}

// CreateParamInt creates and adds an integer parameter to Function f.
func (f *Function) CreateParamInt(name string) *Param {
	param := &Param{
		f:   f,
		id:  f.getId(),
		typ: types.Int,
	}
	if len(name) < 1 {
		param.name = fmt.Sprintf("%s%d", labelParamPrefix, param.id)
	}
	f.params = append(f.params, param)
	return param
}

// CreateParamFloat creates and adds a floating point parameter to Function f.
func (f *Function) CreateParamFloat(name string) *Param {
	param := &Param{
		f:   f,
		id:  f.getId(),
		typ: types.Float,
	}
	if len(name) < 1 {
		param.name = fmt.Sprintf("%s%d", labelDataPrefix, param.id)
	}
	f.params = append(f.params, param)
	return param
}

// GetParam returns the parameter with given name, if it exists. If Function f does not have a parameter with the
// given name, nil is returned.
func (f *Function) GetParam(name string) *Param {
	for _, e1 := range f.params {
		if e1.name == name {
			return e1
		}
	}
	return nil
}

// getId returns a unique identifer for any child of Function f.
func (f *Function) getId() int {
	id := f.seq
	f.seq++
	return id
}

// -------------------------
// ----- Param methods -----
// -------------------------

// Id returns the unique sequence number assigned to Param p when it was created.
func (p *Param) Id() int {
	return p.id
}

// Name returns the given name, if any, or the assigned unique label and sequence id concatenation.
func (p *Param) Name() string {
	return p.name
}

// Type returns the types.Param data object type.
func (p *Param) Type() types.Type {
	return types.Param
}

// DataType returns the data type value of Param p, either types.Int or types.Float.
func (p *Param) DataType() types.DataType {
	return p.typ
}

// String returns the textual LIR representation of Param p.
func (p *Param) String() string {
	return fmt.Sprintf("%s: %s", p.name, p.DataType().String())
}

// Has2Operands returns false for Param, because params don't have operands.
func (p Param) Has2Operands() bool {
	return false
}

// Has1Operand returns false for Param, because params don't have operands.
func (p Param) Has1Operand() bool {
	return false
}

// GetOperand1 panics when called on Param, because params aren't computed.
func (p Param) GetOperand1() Value {
	panic("param does not have operands")
}

// GetOperand2 panics when called on Param, because params aren't computed.
func (p Param) GetOperand2() Value {
	panic("param does not have operands")
}

// SetHW doesn't do anything for Param, but is implemented to support interface Value.
func (p Param) SetHW(r interface{}) {
}

// GetHW returns nil, because Param resides in memory.
func (p Param) GetHW() interface{} {
	panic("param is a memory data object, it doesn't reside in registers")
}

// SetWrapper sets the wrapper for this instruction during register allocation.
func (p Param) SetWrapper(wr interface{}) {
	p.wr = wr
}

// GetWrapper sets the wrapper for this instruction during register allocation.
func (p Param) GetWrapper() interface{} {
	return p.wr
}

// IsConstant returns false for Param.
func (d *Param) IsConstant() bool {
	return false
}
