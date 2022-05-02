package lir

import (
	"fmt"
	"strings"
	"vslc/src/ir/lir/types"
)

// ----------------------------
// ----- Type definitions -----
// ----------------------------

// Function defines an LIR function.
type Function struct {
	m         *Module               // m is the Module that owns this Function.
	id        int                   // id is the unique identifier of this instruction in function body.
	name      string                // name defines the unique string name of function.
	typ       types.DataType        // typ defines the return types.DataType of the function.
	blocks    []*Block              // blocks defines the function body's basic blocks.
	params    []*Param              // params defines the functions parameters.
	variables []*DeclareInstruction // variables holds all the locally defined variables of the function's body.
	seq       int                   // seq defines the locally unique sequence identifier for all children of Function.
	vseq      int                   // vseq defines the unique sequence number for local variables of the Function.
	en        bool                  // Set to true if instruction is enabled.
}

// Param defines an LIR Function parameter.
type Param struct {
	f       *Function      // f is the Function that owns this parameter.
	id      int            // id is the unique function local id of the parameter.
	name    string         // name is the string identifier name given to this parameter.
	typ     types.DataType // typ is the data type of the parameter.
	styp    types.DataType // styp defines the subtype data type of arrays.
	operand Value          // Used for **argv.
	hw      interface{}    // hw defines the instruction's hardware allocated register. Usually set to argument register 0-7.
	en      bool           // Set to true if instruction is enabled.
}

// FunctionCallInstruction defines an LIR function call.
type FunctionCallInstruction struct {
	b         *Block      // b is the basic block element that owns this instruction.
	id        int         // id is the unique identifier of this instruction in function body.
	target    *Function   // target points to the target Function to call.
	arguments []Value     // arguments provides the arguments to pass to the Function during the call.
	hw        interface{} // hw defines the instruction's hardware allocated register. Usually set to argument register 0.
	en        bool        // Set to true if instruction is enabled.
}

// ---------------------
// ----- Constants -----
// ---------------------

// labelFunction is the prefix for all textual LIR functions.
const labelFunction = "function"

// -------------------
// ----- Globals -----
// -------------------

// ---------------------
// ----- Functions -----
// ---------------------

// Id returns the unique id of the FunctionCallInstruction.
func (f *Function) Id() int {
	return f.id
}

// Name returns the textual representation of the virtual register Value of the FunctionCallInstruction.
func (f *Function) Name() string {
	return f.name
}

// DataType returns the DataType of the declared variable that was loaded.
func (f *Function) DataType() types.DataType {
	return f.typ
}

// String returns the textual LIR representation of Function f.
func (f *Function) String() string {
	sb := strings.Builder{}
	sb.WriteString(labelFunction)
	sb.WriteRune(' ')
	sb.WriteString(f.name)
	sb.WriteRune('(')
	if len(f.params) > 0 {
		for i1, e1 := range f.params {
			sb.WriteString(e1.String())
			if i1 < len(f.params)-1 {
				sb.WriteString(", ")
			}
		}
	}
	sb.WriteString("): ")
	sb.WriteString(f.typ.String())

	// Append function body.
	if len(f.blocks) > 0 {
		sb.WriteString(" {\n")
		for _, e1 := range f.variables {
			sb.WriteRune('\t')
			sb.WriteString(e1.String())
			sb.WriteRune('\n')
		}
		for _, e1 := range f.blocks {
			sb.WriteString(e1.String())
			sb.WriteRune('\n')
		}
		sb.WriteRune('}')
	}
	return sb.String()
}

// Blocks returns Function f's basic blocks.
func (f *Function) Blocks() []*Block {
	return f.blocks
}

// Params returns Function f's slice of parameters.
func (f *Function) Params() []*Param {
	return f.params
}

// Locals returns Function f's slice of locally declared variables.
func (f *Function) Locals() []*DeclareInstruction {
	return f.variables
}

// CreateParam creates a new parameter for Function f.
func (f *Function) CreateParam(name string, typ types.DataType) *Param {
	if len(name) < 0 {
		panic("no name given for parameter")
	}
	if p := f.GetParam(name); p != nil {
		panic(fmt.Sprintf("duplicate declaration: parameter %s already defined for function %s",
			name, f.name))
	}
	p := &Param{
		f:    f,
		id:   f.getId(),
		name: name,
		typ:  typ,
		en:   true,
	}
	f.params = append(f.params, p)
	return p
}

// GetParam returns the named parameter of Function f, if it exists. If it does not exist, <nil> is returned.
func (f *Function) GetParam(name string) *Param {
	for _, e1 := range f.params {
		if e1.name == name {
			return e1
		}
	}
	return nil
}

// CreateBlock creates a new basic block and appends it to the Function f.
func (f *Function) CreateBlock() *Block {
	b := &Block{
		f:            f,
		id:           f.m.getId(),
		instructions: make([]Value, 0, 16),
		term:         nil,
	}
	f.blocks = append(f.blocks, b)
	return b
}

// CreateGlobalString creates and returns a global string.
func (f *Function) CreateGlobalString(s string) *String {
	return f.m.CreateGlobalString(s)
}

// getId returns a function local unique identifier.
func (f *Function) getId() int {
	id := f.seq
	f.seq++
	return id
}

// getVSeq returns a unique variable sequence number which defines the variables position
// on stack.
func (f *Function) getVSeq() int {
	seq := f.vseq
	f.vseq++
	return seq
}

// ---------------------
// ----- Parameter -----
// ---------------------

// Id returns the unique id of the Param.
func (inst *Param) Id() int {
	return inst.id
}

// Name returns the textual representation of the virtual register Value of the Param.
func (inst *Param) Name() string {
	return inst.name
}

// Type returns the constant identifying this instruction as a Param.
func (inst *Param) Type() types.InstructionType {
	return types.Param
}

// DataType returns the DataType of the declared variable that was loaded.
func (inst *Param) DataType() types.DataType {
	return inst.typ
}

// String returns the textual LIR representation of the Param.
func (inst *Param) String() string {
	return fmt.Sprintf("%s: %s", inst.name, inst.typ.String())
}

// SetHW panics for the Param, because it's a memory value, not a virtual register.
func (inst *Param) SetHW(hw interface{}) {
	inst.hw = hw
}

// GetHW returns <nil> for the Param.
func (inst *Param) GetHW() interface{} {
	return inst.hw
}

// Operand1 panics for the Param.
func (inst *Param) Operand1() Value {
	return inst.operand
}

// Operand2 panics for the Param.
func (inst *Param) Operand2() Value {
	panic("Param does not have a operands")
}

// Enable enables the instruction, resulting in that it will be printed using Module.String.
func (inst *Param) Enable() {
	inst.en = true
}

// Disable disables the instruction, resulting in that it won't be printed using Module.String.
func (inst *Param) Disable() {
	inst.en = false
}

// IsEnabled returns true if the instruction is enabled.
func (inst *Param) IsEnabled() bool {
	return inst.en
}

// -------------------------
// ----- Function call -----
// -------------------------

// Id returns the unique id of the FunctionCallInstruction.
func (inst *FunctionCallInstruction) Id() int {
	return inst.id
}

// Name returns the textual representation of the virtual register Value of the FunctionCallInstruction.
func (inst *FunctionCallInstruction) Name() string {
	return fmt.Sprintf("%s%d", labelDataInstruction, inst.id)
}

// Type returns the constant identifying this instruction as a FunctionCallInstruction.
func (inst *FunctionCallInstruction) Type() types.InstructionType {
	return types.FunctionCallInstruction
}

// DataType returns the DataType of the declared variable that was loaded.
func (inst *FunctionCallInstruction) DataType() types.DataType {
	return inst.target.DataType()
}

// String returns the textual LIR representation of the FunctionCallInstruction.
func (inst *FunctionCallInstruction) String() string {
	sb := strings.Builder{}
	for i1, e1 := range inst.arguments {
		sb.WriteString(e1.Name())
		if i1 < len(inst.arguments)-1 {
			sb.WriteRune(',')
			sb.WriteRune(' ')
		}
	}
	return fmt.Sprintf("%s = call %s(%s)", inst.Name(), inst.target.Name(), sb.String())
}

// SetHW panics for the FunctionCallInstruction, because it's a memory value, not a virtual register.
func (inst *FunctionCallInstruction) SetHW(hw interface{}) {
	inst.hw = hw
}

// GetHW returns <nil> for the FunctionCallInstruction.
func (inst *FunctionCallInstruction) GetHW() interface{} {
	return inst.hw
}

// Operand1 returns <nil> for the FunctionCallInstruction.
func (inst *FunctionCallInstruction) Operand1() Value {
	return &VaList{
		vars: inst.arguments,
	}
}

// Operand2 returns <nil> for the FunctionCallInstruction.
func (inst *FunctionCallInstruction) Operand2() Value {
	return nil
}

// Enable enables the instruction, resulting in that it will be printed using Module.String.
func (inst *FunctionCallInstruction) Enable() {
	inst.en = true
}

// Disable disables the instruction, resulting in that it won't be printed using Module.String.
func (inst *FunctionCallInstruction) Disable() {
	inst.en = false
}

// IsEnabled returns true if the instruction is enabled.
func (inst *FunctionCallInstruction) IsEnabled() bool {
	return inst.en
}

// Target returns a pointer to the Function being called.
func (inst *FunctionCallInstruction) Target() *Function {
	return inst.target
}

// Arguments returns a slice of the function call arguments.
func (inst *FunctionCallInstruction) Arguments() []Value {
	return inst.arguments
}
