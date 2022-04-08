package lir

import (
	"fmt"
	"strings"
	"vslc/src/ir/lir/types"
)

// ----------------------------
// ----- Type definitions -----
// ----------------------------

// Block defines a basic block. A basic block is a sequence of instructions that is terminated by a branch instruction.
type Block struct {
	f            *Function // Parent function that owns the basic block.
	id           int       // Unique identifier of basic block.
	term         Value     // Branch instruction or return instruction.
	instructions []Value   // Instructions in the basic block.
}

// ---------------------
// ----- Constants -----
// ---------------------

// labelBlockPrefix defines the textual LIR representation of a basic block label.
const labelBlockPrefix = "block"

// labelAllocPrefix defines the textual LIR representation of a declaration label.
const labelAllocPrefix = "var"

// -------------------
// ----- globals -----
// -------------------

// ---------------------
// ----- functions -----
// ---------------------

// Id returns the uniquely assigned identifier of Block b.
func (b *Block) Id() int {
	return b.id
}

// Name returns the textual LIR label name of Block b.
func (b *Block) Name() string {
	return fmt.Sprintf("%s%d", labelBlockPrefix, b.id)
}

// Type returns the LIR object type of Block b.
func (b *Block) Type() types.Type {
	return types.Block
}

// String returns the textual LIR representation of all instructions in Block b.
func (b *Block) String() string {
	sb := strings.Builder{}
	sb.WriteString(fmt.Sprintf("%s%d:\n", labelBlockPrefix, b.id))
	for _, e1 := range b.instructions {
		sb.WriteRune('\t')
		sb.WriteString(e1.String())
		sb.WriteRune('\n')
	}
	if b.term == nil {
		sb.WriteString(fmt.Sprintf("// Error: basic block %s is not terminated. Terminate using return statement or branch statement.\n",
			b.Name()))
	}
	return sb.String()
}

// Instructions returns the instructions of the basic Block b.
func (b *Block) Instructions() []*Value {
	res := make([]*Value, len(b.instructions))
	for i1 := range b.instructions {
		res[i1] = &(b.instructions[i1])
	}
	return res
}

// ------------------------------
// ----- Branch instruction -----
// ------------------------------

// CreateConditionalBranch creates a conditional branch of type IF-THEN-ELSE. Even though you may want a simple IF-THEN
// branch, it is still required to perform a branch, which means that two subsequent asic blocks are required.
// The conditional branch instructions perform a sub instruction (op1 - op2) and performs a check of the result to
// zero.
func (b *Block) CreateConditionalBranch(op types.LogicalOperation, op1, op2 Value, thn *Block, els *Block) *BranchInstruction {
	if types.Equal < op || op > types.LessThanOrEqual {
		panic(fmt.Sprintf("function %s, block %s: unexpected logic operation %d",
			b.f.Name(), b.Name(), op))
	}
	if op1.Type() != types.Data && op1.Type() != types.Load {
		panic(fmt.Sprintf("operand 1 is not a value, cannot use %s as input to CreateConditionalBranch",
			op1.Type().String()))
	}
	if op2.Type() != types.Data && op2.Type() != types.Load {
		panic(fmt.Sprintf("operand 2 is not a value, cannot use %s as input to CreateConditionalBranch",
			op2.Type().String()))
	}
	cmp := b.CreateSub(op1, op2)
	br := &BranchInstruction{
		b:    b,
		id:   b.f.getId(),
		typ:  types.Conditional,
		op:   op,
		next: thn,
		els:  els,
		val:  cmp,
	}
	br.name = fmt.Sprintf("cond%d", br.id)
	b.instructions = append(b.instructions, br)
	b.term = br
	return br
}

// CreateBranch creates an unconditional branch instruction, effectively terminating Block b.
func (b *Block) CreateBranch(dst *Block) *BranchInstruction {
	br := &BranchInstruction{
		b:    b,
		id:   b.f.getId(),
		typ:  types.Unconditional,
		next: dst,
	}
	br.name = fmt.Sprintf("jump%d", br.id)
	b.instructions = append(b.instructions, br)
	b.term = br
	return br
}

// CreateReturn creates a return instruction, effectively terminating Block b.
func (b *Block) CreateReturn(val Value) *BranchInstruction {
	if val.Type() != types.Data && val.Type() != types.Load {
		panic(fmt.Sprintf("operand 1 is not a value, cannot use %s as input to CreateReturn",
			val.Type().String()))
	}
	br := &BranchInstruction{
		b:   b,
		id:  b.f.getId(),
		typ: types.Return,
		val: val,
	}
	br.name = fmt.Sprintf("ret%d", br.id)
	b.instructions = append(b.instructions, br)
	b.term = br
	return br
}

// ---------------------------------
// ----- Constant instructions -----
// ---------------------------------

// CreateConstantInt creates an integer constant.
func (b *Block) CreateConstantInt(i int) *Constant {
	c := &Constant{
		id:  b.f.getId(),
		typ: types.Int,
		val: i,
	}
	c.name = fmt.Sprintf("%s%d", labelDataPrefix, c.id)
	b.instructions = append(b.instructions, c)
	return c
}

// CreateConstantFloat creates a floating point constant.
func (b *Block) CreateConstantFloat(f float64) *Constant {
	c := &Constant{
		id:  b.f.getId(),
		typ: types.Float,
		val: f,
	}
	c.name = fmt.Sprintf("%s%d", labelDataPrefix, c.id)
	b.instructions = append(b.instructions, c)
	return c
}

// -----------------------------------
// ----- Arithmetic instructions -----
// -----------------------------------

// CreateAdd creates an add instruction. The resulting DataInstruction = op1 + op2.
func (b *Block) CreateAdd(op1, op2 Value) *DataInstruction {
	if op1.Type() != types.Data && op1.Type() != types.Load {
		panic(fmt.Sprintf("operand 1 is not a value, cannot use %s as input to CreateAdd",
			op1.Type().String()))
	}
	if op2.Type() != types.Data && op2.Type() != types.Load {
		panic(fmt.Sprintf("operand 2 is not a value, cannot use %s as input to CreateAdd",
			op2.Type().String()))
	}
	inst := &DataInstruction{
		b:   b,
		id:  b.f.getId(),
		op:  types.Add,
		op1: op1,
		op2: op2,
	}
	b.instructions = append(b.instructions, inst)
	return inst
}

// CreateSub creates a subtraction instruction. The resulting DataInstruction = op1 - op2.
func (b *Block) CreateSub(op1, op2 Value) *DataInstruction {
	if op1.Type() != types.Data && op1.Type() != types.Load {
		panic(fmt.Sprintf("operand 1 is not a value, cannot use %s as input to CreateSub",
			op1.Type().String()))
	}
	if op2.Type() != types.Data && op2.Type() != types.Load {
		panic(fmt.Sprintf("operand 2 is not a value, cannot use %s as input to CreateSub",
			op2.Type().String()))
	}
	inst := &DataInstruction{
		b:   b,
		id:  b.f.getId(),
		op:  types.Subtract,
		op1: op1,
		op2: op2,
	}
	b.instructions = append(b.instructions, inst)
	return inst
}

// CreateMul creates a multiplication instruction. The resulting DataInstruction = op1 * op2.
func (b *Block) CreateMul(op1, op2 Value) *DataInstruction {
	if op1.Type() != types.Data && op1.Type() != types.Load {
		panic(fmt.Sprintf("operand 1 is not a value, cannot use %s as input to CreateMul",
			op1.Type().String()))
	}
	if op2.Type() != types.Data && op2.Type() != types.Load {
		panic(fmt.Sprintf("operand 2 is not a value, cannot use %s as input to CreateMul",
			op2.Type().String()))
	}
	inst := &DataInstruction{
		b:   b,
		id:  b.f.getId(),
		op:  types.Multiply,
		op1: op1,
		op2: op2,
	}
	b.instructions = append(b.instructions, inst)
	return inst
}

// CreateDiv creates a division instruction. The resulting DataInstruction = op1 / op2.
func (b *Block) CreateDiv(op1, op2 Value) *DataInstruction {
	if op1.Type() != types.Data && op1.Type() != types.Load {
		panic(fmt.Sprintf("operand 1 is not a value, cannot use %s as input to CreateDiv",
			op1.Type().String()))
	}
	if op2.Type() != types.Data && op2.Type() != types.Load {
		panic(fmt.Sprintf("operand 2 is not a value, cannot use %s as input to CreateDiv",
			op2.Type().String()))
	}
	inst := &DataInstruction{
		b:   b,
		id:  b.f.getId(),
		op:  types.Division,
		op1: op1,
		op2: op2,
	}
	b.instructions = append(b.instructions, inst)
	return inst
}

// CreateRem creates a remainder/modulus instruction. The resulting DataInstruction = op1 % op2.
func (b *Block) CreateRem(op1, op2 Value) *DataInstruction {
	if op1.Type() != types.Data && op1.Type() != types.Load {
		panic(fmt.Sprintf("operand 1 is not a value, cannot use %s as input to CreateRem",
			op1.Type().String()))
	}
	if op2.Type() != types.Data && op2.Type() != types.Load {
		panic(fmt.Sprintf("operand 2 is not a value, cannot use %s as input to CreateRem",
			op2.Type().String()))
	}
	inst := &DataInstruction{
		b:   b,
		id:  b.f.getId(),
		op:  types.Remainder,
		op1: op1,
		op2: op2,
	}
	b.instructions = append(b.instructions, inst)
	return inst
}

// CreateLShift creates a left shift instruction. The resulting DataInstruction = op1 << op2.
func (b *Block) CreateLShift(op1, op2 Value) *DataInstruction {
	if op1.Type() != types.Data && op1.Type() != types.Load {
		panic(fmt.Sprintf("operand 1 is not a value, cannot use %s as input to CreateLShift",
			op1.Type().String()))
	}
	if op2.Type() != types.Data && op2.Type() != types.Load {
		panic(fmt.Sprintf("operand 2 is not a value, cannot use %s as input to CreateLShift",
			op2.Type().String()))
	}
	inst := &DataInstruction{
		b:   b,
		id:  b.f.getId(),
		op:  types.LeftShift,
		op1: op1,
		op2: op2,
	}
	b.instructions = append(b.instructions, inst)
	return inst
}

// CreateRShift creates a right shift instruction. The resulting DataInstruction = op1 >> op2.
func (b *Block) CreateRShift(op1, op2 Value) *DataInstruction {
	if op1.Type() != types.Data && op1.Type() != types.Load {
		panic(fmt.Sprintf("operand 1 is not a value, cannot use %s as input to CreateRShift",
			op1.Type().String()))
	}
	if op2.Type() != types.Data && op2.Type() != types.Load {
		panic(fmt.Sprintf("operand 2 is not a value, cannot use %s as input to CreateRShift",
			op2.Type().String()))
	}
	inst := &DataInstruction{
		b:   b,
		id:  b.f.getId(),
		op:  types.RightShift,
		op1: op1,
		op2: op2,
	}
	b.instructions = append(b.instructions, inst)
	return inst
}

// CreateXOR creates a bitwise XOR instruction. The resulting DataInstruction = op1 ^ op2.
func (b *Block) CreateXOR(op1, op2 Value) *DataInstruction {
	if op1.Type() != types.Data && op1.Type() != types.Load {
		panic(fmt.Sprintf("operand 1 is not a value, cannot use %s as input to CreateXOR",
			op1.Type().String()))
	}
	if op2.Type() != types.Data && op2.Type() != types.Load {
		panic(fmt.Sprintf("operand 2 is not a value, cannot use %s as input to CreateXOR",
			op2.Type().String()))
	}
	inst := &DataInstruction{
		b:   b,
		id:  b.f.getId(),
		op:  types.Xor,
		op1: op1,
		op2: op2,
	}
	b.instructions = append(b.instructions, inst)
	return inst
}

// CreateOR creates a bitwise OR instruction. The resulting DataInstruction = op1 | op2.
func (b *Block) CreateOR(op1, op2 Value) *DataInstruction {
	if op1.Type() != types.Data && op1.Type() != types.Load {
		panic(fmt.Sprintf("operand 1 is not a value, cannot use %s as input to CreateOR",
			op1.Type().String()))
	}
	if op2.Type() != types.Data && op2.Type() != types.Load {
		panic(fmt.Sprintf("operand 2 is not a value, cannot use %s as input to CreateOR",
			op2.Type().String()))
	}
	inst := &DataInstruction{
		b:   b,
		id:  b.f.getId(),
		op:  types.Or,
		op1: op1,
		op2: op2,
	}
	b.instructions = append(b.instructions, inst)
	return inst
}

// CreateAND creates a bitwise AND instruction. The resulting DataInstruction = op1 & op2.
func (b *Block) CreateAND(op1, op2 Value) *DataInstruction {
	if op1.Type() != types.Data && op1.Type() != types.Load {
		panic(fmt.Sprintf("operand 1 is not a value, cannot use %s as input to CreateAND",
			op1.Type().String()))
	}
	if op2.Type() != types.Data && op2.Type() != types.Load {
		panic(fmt.Sprintf("operand 2 is not a value, cannot use %s as input to CreateAND",
			op2.Type().String()))
	}
	inst := &DataInstruction{
		b:   b,
		id:  b.f.getId(),
		op:  types.And,
		op1: op1,
		op2: op2,
	}
	b.instructions = append(b.instructions, inst)
	return inst
}

// CreateNOT creates a bitwise NOT instruction. The resulting DataInstruction = ~op1.
func (b *Block) CreateNOT(op1 Value) *DataInstruction {
	if op1.Type() != types.Data && op1.Type() != types.Load {
		panic(fmt.Sprintf("operand 1 is not a value, cannot use %s as input to CreateNOT",
			op1.Type().String()))
	}
	inst := &DataInstruction{
		b:   b,
		id:  b.f.getId(),
		op:  types.Not,
		op1: op1,
	}
	b.instructions = append(b.instructions, inst)
	return inst
}

// CreateNeg creates an arithmetic negate instruction. The resulting DataInstruction = -op1.
func (b *Block) CreateNeg(op1 Value) *DataInstruction {
	if op1.Type() != types.Data && op1.Type() != types.Load {
		panic(fmt.Sprintf("operand 1 is not a value, cannot use %s as input to CreateNeg",
			op1.Type().String()))
	}
	inst := &DataInstruction{
		b:   b,
		id:  b.f.getId(),
		op:  types.Neg,
		op1: op1,
	}
	b.instructions = append(b.instructions, inst)
	return inst
}

// -------------------------------
// ----- Memory instructions -----
// -------------------------------

// CreateDeclareInt creates a new types.Int variable on the stack.
func (b *Block) CreateDeclareInt(name string) *DeclareInstruction {
	inst := &DeclareInstruction{
		b:   b,
		id:  b.f.getId(),
		typ: types.Int,
	}
	if len(name) > 0 {
		inst.name = name
	} else {
		inst.name = fmt.Sprintf("%s%d", labelAllocPrefix, inst.id)
	}
	b.instructions = append(b.instructions, inst)
	b.f.variables = append(b.f.variables, inst)
	return inst
}

// CreateDeclareFloat creates a new types.Float variable on the stack.
func (b *Block) CreateDeclareFloat(name string) *DeclareInstruction {
	inst := &DeclareInstruction{
		b:   b,
		id:  b.f.getId(),
		typ: types.Float,
	}
	if len(name) > 0 {
		inst.name = name
	} else {
		inst.name = fmt.Sprintf("%s%d", labelAllocPrefix, inst.id)
	}
	b.instructions = append(b.instructions, inst)
	b.f.variables = append(b.f.variables, inst)
	return inst
}

// CreateLoad loads the value of Value src into the returned MemoryInstruction.
func (b *Block) CreateLoad(src Value) *MemoryInstruction {
	if src.Type() != types.Param && src.Type() != types.Global && src.Type() != types.Local {
		panic(fmt.Sprintf("cannot load from %s", src.Type().String()))
	}
	inst := &MemoryInstruction{
		b:   b,
		id:  b.f.getId(),
		typ: types.LoadInstruction,
		src: src,
	}
	b.instructions = append(b.instructions, inst)
	return inst
}

// CreateStore stores the value of virtual register src into the memory allocated variable dst.
func (b *Block) CreateStore(src Value, dst *DeclareInstruction) *MemoryInstruction {
	if dst.Type() != types.Param && dst.Type() != types.Global && dst.Type() != types.Local {
		panic(fmt.Sprintf("cannot store to %s", dst.Type().String()))
	}
	if src.Type() != types.Data {
		panic(fmt.Sprintf("cannot store from %s", src.Type().String()))
	}
	inst := &MemoryInstruction{
		b:   b,
		id:  b.f.getId(),
		typ: types.StoreInstruction,
		src: src,
		dst: dst,
	}
	b.instructions = append(b.instructions, inst)
	return inst
}

// -------------------------------
// ----- String instructions -----
// -------------------------------

// CreateString creates a string in the module's global data. The return value of CreateString is a pointer to the
// string, like a C-style char-pointer.
func (b *Block) CreateString(s string) *Global {
	return b.f.m.CreateString(s)
}
