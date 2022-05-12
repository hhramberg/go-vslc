// Package types defines LIR instruction types, data types etc.
package types

// ----------------------------
// ----- Type definitions -----
// ----------------------------

// ArithmeticOperation defines a type of arithmetic operation, either unary or binary.
type ArithmeticOperation uint

// RelationalOperation defines the logical operators.
type RelationalOperation uint

// InstructionType defines different type of LIR instructions.
type InstructionType uint

// DataType defines LIR data types.
type DataType uint

// ---------------------
// ----- Constants -----
// ---------------------
const (
	Add    ArithmeticOperation = iota // Add identifies the arithmetic operation a = b + c.
	Sub                               // Sub identifies the arithmetic operation a = b - c.
	Mul                               // Mul identifies the arithmetic operation a = b * c.
	Div                               // Div identifies the arithmetic operation a = b / c.
	Rem                               // Rem identifies the arithmetic operation a = b % c.
	LShift                            // LShift identifies the arithmetic operation a = b << c.
	RShift                            // RShift identifies the arithmetic operation a = b >> c.
	And                               // And identifies the arithmetic operation a = b & c.
	Xor                               // Xor identifies the arithmetic operation a = b ^ c.
	Or                                // Or identifies the arithmetic operation a = b | c.
	Neg                               // Neg identifies the arithmetic operation a = -b.
	Not                               // Not identifies the arithmetic operation a = ~b.
)

const (
	Eq                 RelationalOperation = iota // Eq defines ==.
	Neq                                           // Neq defines !=.
	LessThan                                      // LessThan defines <.
	LessThanOrEqual                               // LessThanOrEqual defines <=.
	GreaterThan                                   // GreaterThan defines >.
	GreaterThanOrEqual                            // GreaterThanOrEqual defines >=.
)

const (
	DataInstruction InstructionType = iota
	LoadInstruction
	StoreInstruction
	Constant
	BranchInstruction
	ReturnInstruction
	DeclareInstruction
	FunctionCallInstruction
	Global
	Param
	PrintInstruction
	CastInstruction
	PreserveInstruction
)

const (
	Int DataType = iota
	Float
	String
	VaList // Variable argument list.
	Unknown
)

// -------------------
// ----- Globals -----
// -------------------

// aTyp provides string literals for ArithmeticOperation constants.
var aTyp = [...]string{
	"add",
	"sub",
	"mul",
	"div",
	"rem",
	"lshift",
	"rshift",
	"and",
	"xor",
	"or",
	"neg",
	"not",
}

// iTyp provides string literals for InstructionType constants.
var iTyp = [...]string{
	"DataInstruction",
	"LoadInstruction",
	"StoreInstruction",
	"BranchInstruction",
	"ConditionalBranchInstruction",
	"ReturnInstruction",
	"DeclareInstruction",
	"FunctionCallInstruction",
	"Global",
	"Param",
	"Local",
	"PrintInstruction",
	"CastInstruction",
	"PreserveInstruction",
}

// dTyp provides string literals for DataType constants.
var dTyp = [...]string{
	"Int",
	"Float",
	"String",
	"...",
	"Unknown",
}

// lTyp provides string literals for RelationalOperation constants.
var lTyp = [...]string{
	"Eq",
	"Neq",
	"LessThan",
	"LessThanOrEqual",
	"GreaterThan",
	"GreaterThanOrEqual",
}

// ---------------------
// ----- Functions -----
// ---------------------

// String provides a print friendly string representation of the ArithmeticOperation.
func (inst ArithmeticOperation) String() string {
	return aTyp[inst]
}

// String provides a print friendly string representation of the InstructionType.
func (inst InstructionType) String() string {
	return iTyp[inst]
}

// String provides a print friendly string representation of the DataType.
func (inst DataType) String() string {
	return dTyp[inst]
}

// String provides a print friendly string representation of the RelationalOperation.
func (inst RelationalOperation) String() string {
	return lTyp[inst]
}
