package lir

import (
	"fmt"
	"strings"
	"vslc/src/ir/lir/types"
)

// ----------------------------
// ----- Type definitions -----
// ----------------------------

// Block defines a basic block. A basic block is a sequence of instructions. The Block is terminated by a single
// branch instruction, be it a function call, unconditional jump, conditional jump, or return statement.
type Block struct {
	f            *Function // f is the Function that owns the Block.
	id           int       // id is th unique global identifier of the block.
	instructions []Value   // instructions holds all the instructions defined for the Block.
	term         Value     // term defines the terminating instruction of the Block.
}

// ---------------------
// ----- Constants -----
// ---------------------

// labelBlock is the prepended string prefix for textual LIR block names.
const labelBlock = "block"

// -------------------
// ----- Globals -----
// -------------------

// ---------------------
// ----- Functions -----
// ---------------------

// Name returns the LIR textual name of the Block.
func (b *Block) Name() string {
	return fmt.Sprintf("%s%d", labelBlock, b.id)
}

// String returns the LIR textual representation of the Block b.
func (b *Block) String() string {
	sb := strings.Builder{}
	sb.WriteString(fmt.Sprintf("%s:\n", b.Name()))

	// Append instructions.
	for i1, e1 := range b.instructions {
		if e1.IsEnabled() {
			sb.WriteRune('\t')
			sb.WriteString(e1.String())
			if i1 < len(b.instructions)-1 {
				sb.WriteRune('\n')
			}
		}
	}
	if b.term == nil || !b.term.IsEnabled() {
		panic(
			fmt.Sprintf("%s is not terminated. Terminate using unconditional or conditional jump, function call or return and enable the terminating instruction",
				b.Name()),
		)
	}
	return sb.String()
}

// Instructions returns Block b's slice of instructions.
func (b *Block) Instructions() []Value {
	return b.instructions
}

// ---------------------------------
// ----- Constant instructions -----
// ---------------------------------

// CreateConstantInt creates an integer constant.
func (b *Block) CreateConstantInt(i int) *Constant {
	inst := &Constant{
		b:   b,
		id:  b.f.getId(),
		typ: types.Int,
		val: i,
		en:  true,
	}
	inst.name = fmt.Sprintf("%s%d", labelDataInstruction, inst.id)
	b.instructions = append(b.instructions, inst)
	b.f.m.constants = append(b.f.m.constants, inst) // Append to Module's slice of constants.
	return inst
}

// CreateConstantFloat creates a floating point constant.
func (b *Block) CreateConstantFloat(f float64) *Constant {
	inst := &Constant{
		b:   b,
		id:  b.f.getId(),
		typ: types.Float,
		val: f,
		en:  true,
	}
	inst.name = fmt.Sprintf("%s%d", labelDataInstruction, inst.id)
	b.instructions = append(b.instructions, inst)
	b.f.m.constants = append(b.f.m.constants, inst) // Append to Module's slice of constants.
	return inst
}

// -----------------------------
// ----- Cast instructions -----
// -----------------------------

// CreateIntToFloat casts a types.Int Value into a types.Float Value.
func (b *Block) CreateIntToFloat(v Value) *CastInstruction {
	if v.Type() != types.DataInstruction && v.Type() != types.LoadInstruction &&
		v.Type() != types.Constant && v.Type() != types.FunctionCallInstruction &&
		v.Type() != types.CastInstruction {
		panic(fmt.Sprintf("can't create data cast from %s", v.Type().String()))
	}
	inst := &CastInstruction{
		b:   b,
		id:  b.f.getId(),
		typ: types.Float,
		src: v,
		en:  true,
	}
	b.instructions = append(b.instructions, inst)
	return inst
}

// CreateFloatToInt casts a types.Float Value into a types.Int Value.
func (b *Block) CreateFloatToInt(v Value) *CastInstruction {
	if v.Type() != types.DataInstruction && v.Type() != types.LoadInstruction &&
		v.Type() != types.Constant && v.Type() != types.FunctionCallInstruction &&
		v.Type() != types.CastInstruction {
		panic(fmt.Sprintf("can't create data cast from %s", v.Type().String()))
	}
	inst := &CastInstruction{
		b:   b,
		id:  b.f.getId(),
		typ: types.Int,
		src: v,
		en:  true,
	}
	b.instructions = append(b.instructions, inst)
	return inst
}

// -----------------------------
// ----- Data instructions -----
// -----------------------------

// CreateAdd creates an LIR add instruction and puts the result in the returned virtual register.
// Result = op1 + op2
func (b *Block) CreateAdd(op1, op2 Value) *DataInstruction {
	return b.createArithmeticInstruction(types.Add, op1, op2)
}

// CreateSub creates an LIR sub instruction and puts the result in the returned virtual register.
// Result = op1 - op2
func (b *Block) CreateSub(op1, op2 Value) *DataInstruction {
	return b.createArithmeticInstruction(types.Sub, op1, op2)
}

// CreateMul creates an LIR sub instruction and puts the result in the returned virtual register.
// Result = op1 * op2
func (b *Block) CreateMul(op1, op2 Value) *DataInstruction {
	return b.createArithmeticInstruction(types.Mul, op1, op2)
}

// CreateDiv creates an LIR div instruction and puts the result in the returned virtual register.
// Result = op1 / op2
func (b *Block) CreateDiv(op1, op2 Value) *DataInstruction {
	if op2.Type() == types.Constant {
		if op2.DataType() == types.Int {
			if op2.(*Constant).val.(int) == 0 {
				panic(fmt.Sprintf("division by zero: constant %s", op2.Name()))
			}
		} else {
			if op2.(*Constant).val.(float64) == 0.0 {
				panic(fmt.Sprintf("division by zero: constant %s", op2.Name()))
			}
		}
	}
	return b.createArithmeticInstruction(types.Mul, op1, op2)
}

// CreateRem creates an LIR rem instruction and puts the result in the returned virtual register.
// Result = op1 % op2
func (b *Block) CreateRem(op1, op2 Value) *DataInstruction {
	if op2.Type() == types.Constant {
		if op2.DataType() == types.Int {
			if op2.(*Constant).val.(int) == 0 {
				panic(fmt.Sprintf("division by zero: constant %s", op2.Name()))
			}
		} else {
			if op2.(*Constant).val.(float64) == 0.0 {
				panic(fmt.Sprintf("division by zero: constant %s", op2.Name()))
			}
		}
	}
	return b.createArithmeticInstruction(types.Rem, op1, op2)
}

// CreateLShift creates an LIR left shift instruction and puts the result in the returned virtual register.
// Result = op1 << op2
func (b *Block) CreateLShift(op1, op2 Value) *DataInstruction {
	return b.createArithmeticInstruction(types.LShift, op1, op2)
}

// CreateRShift creates an LIR right shift instruction and puts the result in the returned virtual register.
// Result = op1 >> op2
func (b *Block) CreateRShift(op1, op2 Value) *DataInstruction {
	return b.createArithmeticInstruction(types.RShift, op1, op2)
}

// CreateAnd creates an LIR arithmetic and instruction and puts the result in the returned virtual register.
// Result = op1 & op2
func (b *Block) CreateAnd(op1, op2 Value) *DataInstruction {
	return b.createArithmeticInstruction(types.And, op1, op2)
}

// CreateXor creates an LIR arithmetic XOR instruction and puts the result in the returned virtual register.
// Result = op1 ^ op2
func (b *Block) CreateXor(op1, op2 Value) *DataInstruction {
	return b.createArithmeticInstruction(types.Xor, op1, op2)
}

// CreateOr creates an LIR arithmetic OR instruction and puts the result in the returned virtual register.
// Result = op1 | op2
func (b *Block) CreateOr(op1, op2 Value) *DataInstruction {
	return b.createArithmeticInstruction(types.Or, op1, op2)
}

// CreateNeg creates an LIR neg instruction and puts the result in the returned virtual register.
// Result = -op1
func (b *Block) CreateNeg(op1 Value) *DataInstruction {
	return b.createArithmeticInstruction(types.Neg, op1, nil)
}

// CreateNot creates an LIR not instruction and puts the result in the returned virtual register.
// Result = ~op1
func (b *Block) CreateNot(op1 Value) *DataInstruction {
	return b.createArithmeticInstruction(types.Not, op1, nil)
}

// createArithmeticInstruction creates an arithmetic data instruction with the given operator and operands.
// The method panics if an error occurs.
func (b *Block) createArithmeticInstruction(op types.ArithmeticOperation, op1, op2 Value) *DataInstruction {
	if op1.Type() != types.DataInstruction && op1.Type() != types.Constant && op1.Type() != types.LoadInstruction && op1.Type() != types.FunctionCallInstruction {
		panic(fmt.Sprintf("cannot use value %s of type %s as operand", op1.Name(), op1.Type().String()))
	}
	if op < types.Neg {
		if op2 == nil {
			panic("second operand is <nil>")
		}
		if op2.Type() != types.DataInstruction && op2.Type() != types.Constant && op2.Type() != types.LoadInstruction && op2.Type() != types.FunctionCallInstruction {
			panic(fmt.Sprintf("cannot use value %s of type %s, as operand for arithmetic instruction", op2.Name(), op2.Type().String()))
		}
	}
	if op1.DataType() != op2.DataType() {
		// Cast datatype. Prefer float over int.
		if op1.DataType() == types.Int {
			op1 = b.CreateIntToFloat(op1)
		} else {
			op2 = b.CreateIntToFloat(op2)
		}
	}

	// Verify that the expression is allowed with the given operator.
	if !expLut[op1.DataType()][op2.DataType()][op] {
		panic(fmt.Sprintf("invalid operator %s with operands %s (%s) and %s (%s)",
			op.String(), op1.Name(), op1.DataType().String(), op2.Name(), op2.DataType().String()))
	}

	// Create, append and return the expression.
	inst := &DataInstruction{
		b:   b,
		id:  b.f.getId(),
		op:  op,
		op1: op1,
		op2: op2,
		en:  true,
	}
	b.instructions = append(b.instructions, inst)
	return inst
}

// CreateFunctionCall creates an LIR function call of the provided target function using the provided parameters.
// Result = target(arguments ...)
func (b *Block) CreateFunctionCall(target *Function, arguments []Value) *FunctionCallInstruction {
	if target == nil {
		panic("no target function provided, target function is <nil>")
	}

	if len(target.params) != len(arguments) {
		panic(fmt.Sprintf("expected %d arguments, got %d", len(target.params), len(arguments)))
	}

	// Verify correct data type of arguments.
	for i1, e1 := range arguments {
		param := target.Params()[i1]
		if e1.DataType() != param.DataType() {
			if e1.DataType() == types.Int {
				// Cast int to float.
				cast := b.CreateIntToFloat(e1)
				b.instructions = append(b.instructions, cast)
				arguments[i1] = cast
			} else if e1.DataType() == types.Float {
				// Cast float to int.
				cast := b.CreateFloatToInt(e1)
				b.instructions = append(b.instructions, cast)
				arguments[i1] = cast
			} else {
				// String.
				panic("cannot cast string to float nor int during function call")
			}
		}
	}

	inst := &FunctionCallInstruction{
		b:         b,
		id:        b.f.getId(),
		target:    target,
		arguments: arguments,
		en:        true,
	}
	b.instructions = append(b.instructions, inst)
	return inst
}

// -------------------------------
// ----- Branch instructions -----
// -------------------------------

// CreateBranch creates an unconditional branch to the target branch. This method terminates the Block b.
func (b *Block) CreateBranch(target *Block) *BranchInstruction {
	if b.term != nil {
		panic(fmt.Sprintf("basic block %s is already terminated", b.Name()))
	}
	if target == nil {
		panic("cannot create jump: target bock is <nil>")
	}
	inst := &BranchInstruction{
		b:   b,
		id:  b.f.getId(),
		thn: target,
		en:  true,
	}
	b.instructions = append(b.instructions, inst)
	b.term = inst
	return inst
}

// CreateConditionalBranch creates a conditional branch (if-then-else) to the target branch. This method terminates
// the Block b. The thn Block is taken if the Value rel is not equal to 0, else the els Block is taken.
func (b *Block) CreateConditionalBranch(op types.RelationalOperation, op1, op2 Value, thn, els *Block) *BranchInstruction {
	if b.term != nil {
		panic(fmt.Sprintf("basic block %s is already terminated", b.Name()))
	}
	if thn == nil {
		panic("cannot create jump: target then bock is <nil>")
	}
	if els == nil {
		panic("cannot create jump: target else bock is <nil>")
	}
	if op1.Type() != types.DataInstruction && op1.Type() != types.Constant && op1.Type() != types.LoadInstruction && op1.Type() != types.FunctionCallInstruction {
		panic(fmt.Sprintf("cannot use value %s as compare operand", op1.Name()))
	}
	if op2.Type() != types.DataInstruction && op2.Type() != types.Constant && op2.Type() != types.LoadInstruction && op2.Type() != types.FunctionCallInstruction {
		panic(fmt.Sprintf("cannot use value %s as compare operand", op2.Name()))
	}
	if op > types.GreaterThanOrEqual {
		panic(fmt.Sprintf("undefined relational operator: %d", op))
	}
	inst := &BranchInstruction{
		b:   b,
		id:  b.f.getId(),
		thn: thn,
		els: els,
		op1: op1,
		op2: op2,
		op:  op,
		en:  true,
	}
	b.instructions = append(b.instructions, inst)
	b.term = inst
	return inst
}

// CreateReturn creates a return statement. This method terminates Block b.
func (b *Block) CreateReturn(val Value) *ReturnInstruction {
	if val.Type() != types.DataInstruction && val.Type() != types.Constant && val.Type() != types.LoadInstruction && val.Type() != types.FunctionCallInstruction {
		panic(fmt.Sprintf("cannot use value %s as return value", val.Name()))
	}
	inst := &ReturnInstruction{
		b:   b,
		id:  b.f.getId(),
		val: val,
		en:  true,
	}
	b.instructions = append(b.instructions, inst)
	b.term = inst
	return inst
}

// -------------------------------
// ----- Memory instructions -----
// -------------------------------

// CreateStore creates a StoreInstruction given a source virtual register and a destination variable.
// The source virtual register must be either DataInstruction, LoadInstruction or FunctionCallInstruction.
// The destination must be a Global, Param or Local instruction type.
func (b *Block) CreateStore(src, dst Value) *StoreInstruction {
	if src.Type() != types.DataInstruction && src.Type() != types.Constant && src.Type() != types.LoadInstruction &&
		src.Type() != types.FunctionCallInstruction && src.Type() != types.CastInstruction {
		panic(fmt.Sprintf("cannot create %s: source type %s not allowed",
			types.StoreInstruction.String(), src.Type().String()))
	}
	if dst.Type() != types.Global && dst.Type() != types.Param && dst.Type() != types.DeclareInstruction {
		panic(fmt.Sprintf("cannot create %s: destination type %s not allowed",
			types.StoreInstruction.String(), dst.Type().String()))
	}
	if src.DataType() != dst.DataType() {
		// Cast to destination data type.
		if src.DataType() == types.Int {
			src = b.CreateIntToFloat(src)
		} else {
			src = b.CreateFloatToInt(src)
		}
	}
	inst := &StoreInstruction{
		b:   b,
		id:  b.f.getId(),
		src: src,
		dst: dst,
		en:  true,
	}
	b.instructions = append(b.instructions, inst)
	return inst
}

// CreateLoad creates a LoadInstruction given a source variable. The source variable must be either a types.Local,
// types.Global or types.Param type Value.
func (b *Block) CreateLoad(src Value) *LoadInstruction {
	if src.Type() != types.Global && src.Type() != types.Param && src.Type() != types.DeclareInstruction {
		panic(fmt.Sprintf("cannot create load from %s: can only load from globals, arguments or locally declared variables",
			src.Type().String()))
	}
	inst := &LoadInstruction{
		b:   b,
		id:  b.f.getId(),
		src: src,
		en:  true,
	}
	b.instructions = append(b.instructions, inst)
	return inst
}

// --------------------------------
// ----- Declare instructions -----
// --------------------------------

// CreateDeclare creates a locally declared variable that resides on the stack of the Function that owns Block b.
func (b *Block) CreateDeclare(name string, typ types.DataType) *DeclareInstruction {
	if typ > types.Float {
		panic(fmt.Sprintf("cannot declare a variable: only %s and %s variables are allowed",
			types.Int.String(), types.Float.String()))
	}
	inst := &DeclareInstruction{
		b:   b,
		id:  b.f.getId(),
		typ: typ,
		en:  true,
	}
	if len(name) > 0 {
		inst.name = name
	} else {
		inst.name = fmt.Sprintf("%s%d", labelDeclare, inst.id)
	}
	// Append declaration to Block b's Function's slice of locally declared variables.
	b.f.variables = append(b.f.variables, inst)
	return inst
}

// ---------------------------
// ----- Print statement -----
// ---------------------------

// CreatePrint creates an LIR function call statement that prints a slice of LIR Values.
// Runtime execution uses standard library printf. Print appends a newline character to the printout.
func (b *Block) CreatePrint(val []Value) *FunctionCallInstruction {
	for _, e1 := range val {
		if e1.Type() != types.DataInstruction && e1.Type() != types.LoadInstruction &&
			e1.Type() != types.Constant && e1.Type() != types.FunctionCallInstruction &&
			e1.Type() != types.CastInstruction {
			panic(fmt.Sprintf("cannot print a %s value", e1.Type().String()))
		}
	}

	// Check if printf is defined.
	printf := b.f.m.GetFunction(reservedNames[0])
	if printf == nil {
		// Define printf and add it to Module m.
		b.f.m.Lock()
		printf = &Function{
			m:      b.f.m,
			id:     b.f.m.seq,
			name:   reservedNames[0],
			typ:    types.Int,
			params: make([]*Param, 2),
		}
		b.f.m.seq++
		format := &Param{
			f:    printf,
			id:   printf.getId(),
			name: "format",
			typ:  types.String,
			en:   true,
		}
		valist := &Param{
			f:    printf,
			id:   printf.getId(),
			name: "args",
			typ:  types.VaList,
			en:   true,
		}
		printf.params[0] = format
		printf.params[1] = valist
		b.f.m.functions = append(b.f.m.functions, printf)
		b.f.m.fmap[printf.name] = printf
		b.f.m.Unlock()
	}

	// Pre allocate string buffer.
	sb := strings.Builder{}
	sb.Grow(len(val) * 3) // A % and data type format identifier plus a single space (newline at end).

	// Build format string.
	for i1, e1 := range val {
		sb.WriteRune('%')
		switch e1.DataType() {
		case types.Int:
			sb.WriteRune('d')
		case types.Float:
			sb.WriteRune('f')
		case types.String:
			sb.WriteRune('s')
		default:
			panic(fmt.Sprintf("cannot print data type %s", e1.String()))
		}
		if i1 < len(val)-1 {
			sb.WriteRune(' ')
		}
	}
	sb.WriteRune('\n')

	// Create string constant and load the constant address.
	format := b.f.m.CreateGlobalString(sb.String())
	fload := b.CreateLoad(format)

	// Create variable argument list.
	valist := &VaList{
		b:    b,
		id:   b.f.getId(),
		vars: val,
		en:   true,
	}

	b.instructions = append(b.instructions, valist)

	// Create function call to printf.
	inst := &FunctionCallInstruction{
		b:         b,
		id:        b.f.getId(),
		target:    printf,
		arguments: []Value{fload, valist},
		en:        true,
	}
	b.instructions = append(b.instructions, inst)
	return inst
}
