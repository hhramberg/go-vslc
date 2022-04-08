// Package regfile provides type definitions for virtual register files.
package regfile

// ----------------------------
// ----- Type definitions -----
// ----------------------------

// Register defines a physical register interface.
// A register has a type (floating point or integer), an identifier (usually ranging from 0-32)
// and a flag stating whether it's used.
type Register interface {
	Id() int        // The unique id of the register.
	Type() int      // Type returns either float or int.
	String() string // String returns the assembler string for the register.
}

// RegisterFile defines an interface for a virtual register file.
// A register file must support retrieval of SP, FP, LR and temporary registers.
type RegisterFile interface {
	SP() Register                                // Returns the stack pointer register.
	LR() Register                                // Returns the link register.
	FP() Register                                // Returns the frame pointer register.
	GetI(i int) Register                         // Return the i'th integer register.
	GetF(i int) Register                         // Returns the i'th floating point register.
	FreeI(i int)                                 // Free/de-allocate integer register with index i.
	FreeF(i int)                                 // Free/de-allocate floating register with index i.
	GetNextTempI() Register                      // Returns the next available temporary integer register.
	GetNextTempF() Register                      // Returns the next available temporary floating point register.
	GetNextTempIExclude(exc []Register) Register // Returns the next available temporary integer register with exclusion indices.
	GetNextTempFExclude(exc []Register) Register // Returns the next available temporary floating point register with exclusion indices.
	Ki() int                                     // Ki returns the number of usable temporary integer registers; allocated and un-allocated.
	Kf() int                                     // Kf returns the number of usable temporary floating point registers; allocated and un-allocated.
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
