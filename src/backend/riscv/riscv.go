// RISV-V has a downward growing stack that is always 16-bytes aligned.

package riscv

import (
	"errors"
	"math"
	"sync"
	"vslc/src/ir"
	"vslc/src/util"
)

// ----------------------------
// ----- Type definitions -----
// ----------------------------

// register holds the status of a register at any given time.
type register struct {
	id    int        // Zero indexed id of register.
	typ   int        // 0 = int, 1 = float.
	seq   int        // Sequence number when (re)assigned.
	use   bool       // false = not used, true = used.
	entry *ir.Symbol // Symbol entry in any symbol table that is residing in register.
}

// registerFile represents the register file of the current architecture.
type registerFile struct {
	i   []register     // All integer registers defined for architecture.
	f   []register     // All floating point registers defined for architecture.
	iht map[string]int // Hash table linking identifier name and register file index for integer registers.
	fht map[string]int // Hash table linking identifier name and register file index for floating point registers.
	seq int            // Last used sequence number when (re)assigning.
}

// ---------------------
// ----- Constants -----
// ---------------------

const labelFloat = ".CFP32_" // Constant prefix of constant floats in data segment.
const labelString = ".STR_"  // Constant prefix of strings in data segment.

// Base registers (integer).
const (
	x0  = iota // Zero register, RO.
	x1         // Return address (caller save).
	x2         // Stack pointer (callee save).
	x3         // Global pointer.
	x4         // Thread pointer.
	x5         // Temp register (caller saved).
	x6         // Temp register (caller saved).
	x7         // Temp register (caller saved).
	x8         // Frame pointer (callee saved).
	x9         // Saved (callee saved).
	x10        // Function args and return (caller saved).
	x11        // Function args and return (caller saved).
	x12        // Function arguments (caller saved).
	x13        // Function arguments (caller saved).
	x14        // Function arguments (caller saved).
	x15        // Function arguments (caller saved).
	x16        // Function arguments (caller saved).
	x17        // Function arguments (caller saved).
	x18        // Saved (callee saved).
	x19        // Saved (callee saved).
	x20        // Saved (callee saved).
	x21        // Saved (callee saved).
	x22        // Saved (callee saved).
	x23        // Saved (callee saved).
	x24        // Saved (callee saved).
	x25        // Saved (callee saved).
	x26        // Saved (callee saved).
	x27        // Saved (callee saved).
	x28        // Temp (caller saved).
	x29        // Temp (caller saved).
	x30        // Temp (caller saved).
	x31        // Temp (caller saved).
)

// Double precision floating point registers from the D extension.
const (
	f0 = iota
	f1
	f2
	f3
	f4
	f5
	f6
	f7
	f8
	f9
	f10
	f11
	f12
	f13
	f14
	f15
	f16
	f17
	f18
	f19
	f20
	f21
	f22
	f23
	f24
	f25
	f26
	f27
	f28
	f29
	f30
	f31
)

// Aliases.
const (
	zero = x0 // Zero.
	ra   = x1 // Return address.
	sp   = x2 // Stack pointer.
	fp   = x8 // Frame pointer.
)

// Integer argument register aliases.
const (
	a0 = iota + x10
	a1
	a2
	a3
	a4
	a5
	a6
	a7
)

// Floating point argument register aliases.
const (
	fa0 = iota + f10
	fa1
	fa2
	fa3
	fa4
	fa5
	fa6
	fa7
)

// Aliases for volatile integer registers that must not be used by callee unless explicitly saved.
const (
	s0 = x8
	s1 = x9
	s2 = iota + x18
	s3
	s4
	s5
	s6
	s7
	s8
	s9
	s10
	s11
)

// Aliases for volatile float registers that must not be used by callee unless explicitly saved.
const (
	fs0 = f8
	fs1 = f9
	fs2 = iota + 18
	fs3
	fs4
	fs5
	fs6
	fs7
	fs8
	fs9
	fs10
	fs11
)

// Aliases for temporary integer registers.
const (
	t0 = x5
	t1 = x6
	t2 = x7
	t3 = iota + x28
	t4
	t5
	t6
)

// Aliases for temporary floating point registers.
const (
	ft0 = iota + f0
	ft1
	ft2
	ft3
	ft4
	ft5
	ft6
	ft7
)

const (
	ft8 = iota + f28
	ft9
	ft10
	ft11
)

// Register types.
const (
	integer = int(ir.DataInteger)
	float   = int(ir.DataFloat)
)

// 12-bit immediate cannot exceed these values.
const maxImm = 2047
const minImm = -2048

const stackAlign = 16 // The stack must be aligned by 16 bytes.
const wordSize = 4    // This is a 32-bit implementation only, word size is 4 bytes.

// -------------------
// ----- Globals -----
// -------------------

// regi is the short form for registers integers and contains string literals for base integer registers.
var regi = [...]string{
	"x0",
	"x1",
	"x2",
	"x3",
	"x4",
	"x5",
	"x6",
	"x7",
	"x8",
	"x9",
	"x10",
	"x11",
	"x12",
	"x13",
	"x14",
	"x15",
	"x16",
	"x17",
	"x18",
	"x19",
	"x20",
	"x21",
	"x22",
	"x23",
	"x24",
	"x25",
	"x26",
	"x27",
	"x28",
	"x29",
	"x30",
	"x31",
}

// regf is the short form for registers floats and contains string literals for floating point registers.
var regf = [...]string{
	"f0",
	"f1",
	"f2",
	"f3",
	"f4",
	"f5",
	"f6",
	"f7",
	"f8",
	"f9",
	"f10",
	"f11",
	"f12",
	"f13",
	"f14",
	"f15",
	"f16",
	"f17",
	"f18",
	"f19",
	"f20",
	"f21",
	"f22",
	"f23",
	"f24",
	"f25",
	"f26",
	"f27",
	"f28",
	"f29",
	"f30",
	"f31",
}

// ---------------------
// ----- Functions -----
// ---------------------

// GenRiscv recursively generates RISC-V assembler code from the intermediate representation.
func GenRiscv(opt util.Options) error {
	// Create and initialise register file representation.
	regFile := registerFile{
		i: make([]register, 32),
		f: make([]register, 32),
	}
	for i1, e1 := range regFile.i {
		e1.typ = integer
		e1.id = i1
		regFile.f[i1].typ = float
		regFile.f[i1].id = i1
	}

	// Global data and constants.

	// Write strings and float constants to data segment.
	wr := util.NewWriter()
	wr.Write(".data\n# Strings.\n")
	for i1, e1 := range ir.Strings.St {
		wr.Write("%s%d:\n\t.asciz\t%q\n", labelString, i1, e1)
	}
	wr.Write("\n# Floating point constants.\n")
	for i1, e1 := range ir.Floats.Ft {
		wr.Write("%s%d:\n\t.word\t%x\n", labelFloat, i1, math.Float32bits(e1))
	}
	// TODO: generate global variables.

	wr.Flush()
	wr.Close()

	// Generate functions.
	if opt.Threads > 1 {
		// Parallel.
		wg := sync.WaitGroup{} // Used for synchronising worker threads with main thread.

		// Initiate worker threads.
		t := opt.Threads                       // Max number of threads to initiate.
		l := len(ir.Root.Children[0].Children) // Number of functions defined in program.
		if t > l {
			t = l // Cannot launch more threads than functions.
		}
		n := l / t   // Number of jobs per worker thread.
		res := l % t // Residual work for res first threads.

		// errs handles error reporting during parallel generation.
		// Since errors can occur at multiple worker threads there may be multiple
		// errors to report when main thread resumes control.
		var errs struct {
			err []error    // Slice of errors.
			mx  sync.Mutex // For synchronising worker threads.
		}

		// Allocate memory for errors; one per worker thread.
		errs.err = make([]error, 0, t)

		// Launch t threads.
		for i1 := 0; i1 < l; i1 += n {
			m := n
			if i1 < res {
				// Indicate that this worker thread should do one more job.
				m++
				i1++
			}
			wg.Add(1) // Tell main thread to wait for new thread to finish.
			go func(i, j int, wg *sync.WaitGroup) {
				defer wg.Done() // Alert main thread that this worker is done when returning.

				// Validate function body.
				for i2 := 0; i2 < j; i2++ {
					// TODO: Create a global list of global functions?
					if ir.Root.Children[0].Children[i+i2].Typ == ir.FUNCTION {
						w := util.NewWriter()                   // Create output handler.
						rf := regFile                           // Copy register file.
						f := ir.Root.Children[0].Children[i+i2] // Function to generate.
						st := util.Stack{}
						st.Push(&ir.Global)
						st.Push(&(f.Entry.Locals))
						if err := genFunction(f, &w, &st, &rf); err != nil {
							errs.mx.Lock()
							errs.err = append(errs.err, err)
							errs.mx.Unlock()
						}

						// Deallocate stack. Can be omitted?
						st.Pop()
						st.Pop()

						// Burst write function assembly to output and close writer.
						w.Flush()
						w.Close()
					}
				}
			}(i1, m, &wg)
		}

		// Wait for worker threads to finish.
		wg.Wait()

		// Check for errors.
		if len(errs.err) > 0 {
			return errors.New("multiple errors during parallel assembly generation")
		}
	} else {
		// Sequential.
		st := util.Stack{}    // Stack used for identifier lookup.
		st.Push(&ir.Global)   // Push global symbol table on stack.
		w := util.NewWriter() // Create output handler.
		rf := regFile         // Copy register file.
		for _, e1 := range ir.Root.Children[0].Children {
			if e1.Typ == ir.FUNCTION {
				st.Push(&(e1.Entry.Locals))
				if err := genFunction(e1, &w, &st, &rf); err != nil {
					return err
				}
				st.Pop()

				// Burst write function assembly to output.
				w.Flush()
			}
		}

		// Close writer.
		w.Close()
	}
	return nil
}

// String returns a print friendly string of the register r.
func (r *register) String() string {
	if r.typ == integer {
		return regi[r.id]
	}
	return regf[r.id]
}

// loadIdentifierToReg loads identifier with name s to a register. The register type and index is returned.
func (rf *registerFile) loadIdentifierToReg(name string, f *ir.Symbol, wr *util.Writer, st *util.Stack) *register {
	// TODO: implement load instructions.
	s, _ := ir.GetEntry(name, st) // Safe, exceptions are caught in intermediate validate stage.

	if s.DataTyp == ir.DataInteger {
		// Check if identifier is in register file already.
		if idx, exist := rf.iht[name]; exist {
			rf.i[idx].seq = rf.seq
			rf.seq++
			return &(rf.i[idx])
		}

		// Not in register file, load from memory.

		// Allocate register.
		reg := rf.lruI()

		switch s.Typ {
		case ir.SymLocal:
		case ir.SymParam:
			if s.Seq < 8 {
				// Load from argument register. Just reference the parameter register itself.
				return &(rf.i[s0+s.Seq])
			}
			// Load from stack.
			wr.Write("\tlw\t%s, %s\n", reg.String()) // TODO: fiks!
		case ir.SymGlobal:
		}

		// Update integers.
		reg.entry = s
		reg.use = true
		reg.seq = rf.seq
		rf.iht[name] = reg.id
		rf.seq++
		return reg
	} else {
		// Check if identifier is in register file already.
		if idx, exist := rf.fht[name]; exist {
			rf.f[idx].seq = rf.seq
			rf.seq++
			return &(rf.i[idx])
		}

		// Not in register file, load from memory.

		// Allocate register.
		reg := rf.lruF()

		// Update floats.
		reg.entry = s
		reg.use = true
		reg.seq = rf.seq
		rf.fht[name] = reg.id
		rf.seq++
		return reg
	}
}

// saveRegToIdentifier takes the contents of the register src and saves it to the memory space allocated to
// identifier with the given name.
func (rf *registerFile) saveRegToIdentifier(name string, src int, wr *util.Writer, st *util.Stack) {
	// TODO: implement store instructions.
	s, _ := ir.GetEntry(name, st) // Safe, exceptions are caught in intermediate validate stage.

	if s.DataTyp == ir.DataInteger {
		wr.Ins2("sw", regi[src], "") // TODO: fix!
		delete(rf.iht, name)
	} else {
		delete(rf.fht, name)
	}
}

// lruF returns the least recently used floating point register of registerFile regf.
func (rf *registerFile) lruF() *register {
	low := int((^uint(0)) >> 1) // Max integer.
	idx := 0
	for i1, e1 := range rf.f[:fa0] {
		if e1.seq < low {
			low = e1.seq
			idx = i1
		}
	}
	for i1, e1 := range rf.f[fa7+1:] {
		if e1.seq < low {
			low = e1.seq
			idx = i1
		}
	}
	return &(rf.f[idx])
}

// lruI returns the least recently used usable integer register of registerFile regf.
func (rf *registerFile) lruI() *register {
	low := int((^uint(0)) >> 1) // Max integer.
	idx := 0
	// Use any registers in range [x18, x31]
	for i1 := a7 + 1; i1 < x31+1; i1++ {
		if rf.i[i1].seq < low {
			low = rf.i[i1].seq
			idx = i1
		}
	}
	return &(rf.i[idx])
}

// useI updates the sequence number of the integer register reg in registerFile rf.
// If an identifier Symbol is provided the register is updated to be holding the value of the identifier.
func (rf *registerFile) useI(idx int, ident *ir.Symbol) {
	rf.i[idx].seq = rf.seq
	rf.seq++
	if ident != nil {
		rf.iht[ident.Name] = idx
	}
}

// useF updates the sequence number of the float register reg in registerFile rf.
// If an identifier Symbol is provided the register is updated to be holding the value of the identifier.
func (rf *registerFile) useF(idx int, ident *ir.Symbol) {
	rf.f[idx].seq = rf.seq
	rf.seq++
	if ident != nil {
		rf.fht[ident.Name] = idx
	}
}

// genAsm generates assembly code recursively from the ir.Node n.
func genAsm(n *ir.Node, f *ir.Symbol, wr *util.Writer, st, ls *util.Stack, rf *registerFile) error {
	return nil
}