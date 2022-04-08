package types

import "fmt"

// ----------------------------
// ----- Type definitions -----
// ----------------------------

// DataType defines whether a value is float, int or string.
type DataType int

// Type defines which type of instruction the value was generated from.
type Type int

// LogicalOperation defines the logical operators for branch instructions.
type LogicalOperation int

// ArithmeticOperation defines the arithmetic operators for general instructions.
type ArithmeticOperation int

// BranchType defines whether a branch is un-conditional, conditional or a return statement.
type BranchType int

// MemoryType differentiates between memory load and store operations.
type MemoryType int

// ---------------------
// ----- Constants -----
// ---------------------

const (
	Data     Type = iota // Data defines a lir.Value that can be used as generic data.
	Function             // Function defines that a data object is a function reference.
	Param                // Param defines a data object as a function parameter.
	Global               // Global defines a data object as a globally declared variable.
	Local                // Local defines a data object as a locally declared variable.
	Block                // Block defines a basic block object.
	Branch               // Branch defines that a data object is a branch instruction.
	Load                 // Load defines a lir.Value that has loaded something from memory.
	Store                // Store defines a lir.Value that stores something to memory.
)

const (
	Int     DataType = iota // Int defines an architecture specific signed integer.
	Float                   // Float defines an architecture specific IEEE 754 floating point number.
	String                  // String defines a sequence of ASCII characters.
	Unknown                 // Unknown datatype is not allowed.
)

const (
	Equal              LogicalOperation = iota // Equal defines the logical == operator.
	NotEqual                                   // NotEqual defines the logical != operator.
	GreaterThan                                // GreaterThan defines the logical > operator.
	LessThan                                   // LessThan defines the logical < operator.
	GreaterThanOrEqual                         // GreaterThanOrEqual defines the logical >= operator.
	LessThanOrEqual                            // LessThanOrEqual defines the logical <= operator.
)

const (
	Add        ArithmeticOperation = iota // Add defines the arithmetic addition operation.
	Subtract                              // Subtract defines the arithmetic subtraction operation.
	Multiply                              // Subtract defines the arithmetic multiply operation.
	Division                              // Subtract defines the arithmetic division operation.
	Remainder                             // Subtract defines the arithmetic remainder/modulus operation. If a = 13 and b = 6, then a % b = 1.
	LeftShift                             // LeftShift defines the bitwise left shift operation.
	RightShift                            // RightShift defines the bitwise right shift operation.
	Xor                                   // Xor defines the bitwise exclusive OR operation.
	Or                                    // Or defines the bitwise OR operation.
	And                                   // And defines the bitwise AND operation.
	Neg                                   // Neg defines the arithmetic negate operation.
	Not                                   // Not defines the bitwise inversion operation. If a = 0b1010, then ~a = 0b0101.
)

const (
	Return        BranchType = iota // A return statement type of branch.
	Unconditional                   // An unconditional branch, a simple jump.
	Conditional                     // A conditional branch, an IF-THEN/IF-THEN-ELSE.
)

const (
	LoadInstruction MemoryType = iota
	StoreInstruction
)

// -------------------
// ----- Globals -----
// -------------------

// tTyp provides string literals for lir data object types.
var tTyp = [...]string{
	"data",
	"function",
	"param",
	"global",
	"local",
	"block",
	"branch",
	"load",
	"store",
}

// dTyp provides string literals for lir data types.
var dTyp = [...]string{
	"int",
	"float",
	"string",
	"unknown",
}

// logTyp provides string literals for lir logical operation instructions.
var logTyp = [...]string{
	"EQ",
	"NE",
	"GT",
	"LT",
	"GTE",
	"LTE",
}

// aTyp provides string literals for lir arithmetic and bitwise operation instructions.
var aTyp = [...]string{
	"add",
	"sub",
	"mul",
	"div",
	"rem",
	"lsl",
	"rsl",
	"xor",
	"or",
	"and",
	"neg",
	"not",
}

// bTyp provides string literals for lir branch instructions.
var bTyp = [...]string{
	"ret",
	"b",
	"if",
}

// ---------------------
// ----- Functions -----
// ---------------------

// String returns the string literal representing the DataType.
func (t Type) String() string {
	if t < 0 || t >= Type(len(dTyp)) {
		panic(fmt.Sprintf("unexpected data type: %d", t))
	}
	return tTyp[t]
}

// String returns the string literal representing the DataType.
func (dt DataType) String() string {
	if dt < 0 || dt >= DataType(len(dTyp)) {
		panic(fmt.Sprintf("unexpected data type: %d", dt))
	}
	return dTyp[dt]
}

// String returns the string literal representing the LogicalOperation datatype.
func (lo LogicalOperation) String() string {
	if lo < 0 || lo >= LogicalOperation(len(logTyp)) {
		panic(fmt.Sprintf("unexpected logical operation: %d", lo))
	}
	return logTyp[lo]
}

// String returns the string literal representing the LogicalOperation datatype.
func (ao ArithmeticOperation) String() string {
	if ao < 0 || ao >= ArithmeticOperation(len(aTyp)) {
		panic(fmt.Sprintf("unexpected arithmetic or bitwise operation: %d", ao))
	}
	return aTyp[ao]
}

// String returns the string literal representing the BranchType datatype.
func (bt BranchType) String() string {
	if bt < 0 || bt >= BranchType(len(bTyp)) {
		panic(fmt.Sprintf("unexpected branch type: %d", bt))
	}
	return bTyp[bt]
}

// String returns the string literal representing the MemoryType datatype.
func (mt MemoryType) String() string {
	if mt < LoadInstruction || mt >= StoreInstruction {
		panic(fmt.Sprintf("unexpected memory operation: %d", mt))
	}
	return tTyp[int(Load)+int(mt)]
}
