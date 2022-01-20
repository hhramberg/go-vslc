package ir

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"vslc/src/util"
)

// ----------------------------
// ----- Type definitions -----
// ----------------------------

// symType differentiate different types of symbols, e.g. global variable, function and parameters.
type symType int

// dataType differentiates data types, such as integer, floating point and boolean.
type dataType int

// Symbol refers to a variable/identifier's entry in the global symbol table.
type Symbol struct {
	Typ     symType  // Type of symbol.
	Name    string   // Name of symbol.
	Seq     int      // Sequence number of variable/function.
	Node    *Node    // Pointer to Symbol's definition node in syntax tree.
	DataTyp dataType // Data type of variable.
	Nparams int      // Number of parameters defined for function.
	Nlocals int      // Number of local variables defined for function, excluding parameters.
	Leaf    bool     // Set true if this function does not call another function.
	Locals  SymTab   // Locally defined function variables and parameters.
}

// SymTab wraps a hash table that can be accessed by multiple threads using a mutex.
type SymTab struct {
	HT map[string]*Symbol // Hash table holding Symbol entries.
	mx sync.Mutex         // Used for synchronising worker threads.
}

// ---------------------
// ----- Constants -----
// ---------------------

const hTabSize = 16 // It is unlikely that we need to store more than 16 global variables in this project.
const sSize = 16    // It is unlikely that we need to store more than 16 strings in this project.

const (
	SymFunc symType = iota
	SymParam
	SymLocal
	SymGlobal
	SymBlock
)

const (
	DataInteger dataType = iota
	DataFloat
)

// sTyp defines strings for print friendly output of symType.
var sTyp = []string{
	"Function",
	"Parameter",
	"Local",
	"Global",
	"Block NODE",
}

// dTyp defines string for print friendly output of dataType.
var dTyp = []string{
	"integer",
	"float",
	"bool",
}

// -------------------
// ----- Globals -----
// -------------------

// Global symbol table.
var Global SymTab

// Strings contains all strings defined in program being compiled.
var Strings struct {
	st []string   // Slice of strings defined in program.
	mx sync.Mutex // Mutex for synchronising worker threads during string insertion.
}

// seqCtrl manages sequence numbers for parallel worker threads.
var seqCtrl struct {
	gFun int        // Sequence number for functions.
	gVar int        // Sequence number for global variables.
	mx   sync.Mutex // Synchronise worker threads using mutex.
}

// ----------------------
// ----- Functions ------
// ----------------------

// GenerateSymTab populates the symbol table for the VSL program.
func GenerateSymTab(opt util.Options) error {
	// Initiate global symbol table and string table.
	Global = SymTab{
		HT: make(map[string]*Symbol, hTabSize),
		mx: sync.Mutex{},
	}
	Strings = struct {
		st []string
		mx sync.Mutex
	}{st: make([]string, sSize), mx: sync.Mutex{}}

	if opt.Threads > 1 {
		// Parallel.
		wg := sync.WaitGroup{} // Used for synchronising worker threads with main thread.
		ts := sync.WaitGroup{} // Used for synchronising worker threads before doing recursive binding.

		// Initiate worker threads.
		t := opt.Threads                    // Max number of threads to initiate.
		l := len(Root.Children[0].Children) // Number of functions defined in program.
		if t > l {
			t = l // Cannot launch more threads than functions.
		}
		n := l / t   // Number of jobs per worker thread.
		res := l % t // Residual work for res first threads.

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
			go func(i, j int, wg, ts *sync.WaitGroup) {
				defer wg.Done() // Alert main thread that this worker is done when returning.

				// Bind function to global symbol table.
				ts.Add(1)
				for i2 := 0; i2 < j; i2++ {
					if err := Root.Children[0].Children[i+i2].bindGlobal(opt); err != nil {
						errs.mx.Lock()
						errs.err = append(errs.err, err)
						errs.mx.Unlock()
					}
				}
				ts.Done()

				// Wait for other worker threads to finish adding functions to global symbol table.
				ts.Wait()

				// Bind function's local variables.
				for i2 := 0; i2 < j; i2++ {
					if Root.Children[0].Children[i+i2].Typ != FUNCTION {
						continue
					}
					f := Root.Children[0].Children[i+i2]
					st := util.Stack{}
					st.Push(&Global)
					st.Push(&f.Entry.Locals)
					f.Children[2].Entry = f.Entry // Link FUNCTION body BLOCK to symbol table entry of function.
					for _, e1 := range f.Children[2].Children {
						if err := e1.bind(&st, f.Entry); err != nil {
							errs.mx.Lock()
							errs.err = append(errs.err, err)
							errs.mx.Unlock()
						}
					}
				}
			}(i1, m, &wg, &ts)
		}

		// Wait for worker threads to finish.
		wg.Wait()

		// Check for errors.
		if len(errs.err) > 0 {
			return errors.New("multiple errors during parallel optimisation")
		}
	} else {
		// Sequential.
		for _, e1 := range Root.Children[0].Children {
			if err := e1.bindGlobal(opt); err != nil {
				return err
			}
		}
		for _, v := range Global.HT {
			st := util.Stack{}
			st.Push(&Global) // Push global symbol table to bottom of stack.
			if v.Typ == SymFunc {
				st.Push(&v.Locals)           // Push function's local definitions to top of stack.
				v.Node.Children[2].Entry = v // Link FUNCTION body BLOCK to symbol table entry of function.
				for _, e1 := range v.Node.Children[2].Children {
					// Iterate over all children of FUNCTION's BLOCK.
					// It is already defined and put on stack.
					if err := e1.bind(&st, v); err != nil {
						return fmt.Errorf("error in body of function %q: %s", v.Name, err)
					}
				}
			}
		}
	}
	return nil
}

// bindGlobal binds global definitions to the global symbol table.
func (n *Node) bindGlobal(opt util.Options) error {
	var seq int
	switch n.Typ {
	case FUNCTION:
		// Get sequence number.
		if opt.Threads > 1 {
			seqCtrl.mx.Lock()
			seq = seqCtrl.gFun
			seqCtrl.gFun++
			seqCtrl.mx.Unlock()
		} else {
			seq = seqCtrl.gFun
			seqCtrl.gFun++
		}

		// Create Symbol.
		s := Symbol{
			Typ:     SymFunc,
			Name:    n.Children[0].Data.(string),
			Seq:     seq,
			Node:    n,
			Nparams: 0,
			Nlocals: 0,
			Leaf:    true,                                                                // Assume function is leaf until otherwise disproved.
			Locals:  SymTab{HT: make(map[string]*Symbol, len(n.Children[1].Children)+8)}, // Leave space for locals.
		}

		// Set return data type.
		if err := s.setDataType(n.Children[1]); err != nil {
			return fmt.Errorf("compiler error: %s", err)
		}

		// Check for duplicates.
		Global.mx.Lock()
		if _, ok := Global.HT[s.Name]; ok {
			Global.mx.Unlock()
			return fmt.Errorf("duplicte declaration for global identifier %q", s.Name)
		}
		Global.mx.Unlock()

		// Iterate over all function parameters.
		seq = 0 // Parameter sequence numbers.
		for _, e1 := range n.Children[2].Children {
			// All typed variable lists defined as parameters.

			for _, e2 := range e1.Children {
				param := Symbol{
					Typ:  SymParam,
					Name: e2.Data.(string),
					Seq:  seq,
					Node: e2,
				}

				// Set return data type.
				if err := param.setDataType(e1); err != nil {
					return fmt.Errorf("compiler error: %s", err)
				}

				// Check for duplicates.
				if _, ok := s.Locals.Get(param.Name); ok {
					return fmt.Errorf("duplicte declaration of parameter %q found in function %q",
						param.Name, s.Name)
				}

				// Add parameter to function's local variables.
				s.Locals.Add(&param)
				seq++
				s.Nparams++
			}
		}

		// Add function symbol to global symbol table.
		Global.Add(&s)
	case DECLARATION:
		// Global variable declaration.
		for _, e1 := range n.Children[0].Children {
			// Get sequence number.
			if opt.Threads > 1 {
				seqCtrl.mx.Lock()
				seq = seqCtrl.gVar
				seqCtrl.gVar++
				seqCtrl.mx.Unlock()
			} else {
				seq = seqCtrl.gVar
				seqCtrl.gVar++
			}

			// Create Symbol.
			s := Symbol{
				Typ:  SymGlobal,
				Name: e1.Data.(string),
				Seq:  seq,
				Node: e1,
			}

			// Set datatype of symbol.
			if err := s.setDataType(n); err != nil {
				return err
			}

			// Check for duplicates.
			if dup, ok := Global.Get(s.Name); ok {
				return fmt.Errorf("duplicte declaration for global identifier %q, already declared at line %d:%d",
					dup.Name, dup.Node.Line, dup.Node.Pos)
			}

			// Add global variable to global symbol table.
			Global.Add(&s)
		}
	case DECLARATION_LIST:
		for _, e1 := range n.Children {
			if err := e1.bindGlobal(opt); err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("expected node of type %s, %s or %s, got: %s",
			nt[FUNCTION], nt[DECLARATION], nt[DECLARATION_LIST], nt[n.Typ])
	}
	return nil
}

// bind populates local scoped symbol tables recursively.
func (n *Node) bind(st *util.Stack, f *Symbol) error {
	switch n.Typ {
	case EXPRESSION_LIST, EXPRESSION, ASSIGNMENT_STATEMENT:
		// The above Nodes don't declare variables, but references variables.
		// Do not check children of these nodes.
		return nil
	case DECLARATION:
		for _, e1 := range n.Children {
			// Slice of VARIABLE_LIST nodes.
			for _, e2 := range e1.Children {
				// Declared IDENTIFIERS.

				// Add local variable to scope.
				name := e2.Data.(string)
				scope := st.Peek().(*SymTab)

				// Check for duplicate declaration in current scope.
				if s, ok := scope.Get(name); ok {
					return fmt.Errorf("variable %q referenced at line %d:%d was previously declared at line %d:%d",
						name, e2.Line, e2.Pos, s.Node.Line, s.Node.Pos)
				}

				// Create Symbol and push to stack.
				s := Symbol{
					Typ:  SymLocal,
					Name: name,
					Seq:  f.Nlocals,
					Node: e2,
				}
				f.Nlocals++
				scope.Add(&s)
			}
		}
	case BLOCK:
		// Add new scope to stack.
		s := Symbol{
			Typ:    SymBlock,
			Node:   n,
			Locals: SymTab{HT: make(map[string]*Symbol, 8), mx: sync.Mutex{}},
		}
		n.Entry = &s // Save local scope symbol table to node. We can use it later when needed.
		st.Push(&(s.Locals))
	case STRING_DATA:
		// Take string data from node and put it in global string table.
		// Replace STRING_DATA node's data with the index of string in string table.
		AddString(n)
	}

	// Recursively bind identifiers declared in children.
	for _, e1 := range n.Children {
		if err := e1.bind(st, f); err != nil {
			return err
		}
	}
	return nil
}

// Add safely adds a new Symbol to the symbol table st.
func (st *SymTab) Add(s *Symbol) {
	st.mx.Lock()
	defer st.mx.Unlock()
	st.HT[s.Name] = s
}

// Get safely retrieves the Symbol with given key if it exists.
// If the Symbol does not exist, the returned bool will be false.
func (st *SymTab) Get(key string) (*Symbol, bool) {
	st.mx.Lock()
	defer st.mx.Unlock()
	s, ok := st.HT[key]
	return s, ok
}

// String returns a print friendly string of SymTab st.
func (st *SymTab) String() string {
	sb := strings.Builder{}
	st.mx.Lock()
	for _, v := range st.HT {
		sb.WriteString(v.String())
		sb.WriteRune('\n')
	}
	st.mx.Unlock()
	return sb.String()
}

// String returns a print friendly string of Symbol s.
func (s *Symbol) String() string {
	if s.Typ == SymFunc {
		return fmt.Sprintf("%s [%q], params: %d, locals: %d", sTyp[s.Typ], s.Name, s.Nparams, s.Nlocals)
	} else {
		return fmt.Sprintf("%s [%q]", sTyp[s.Typ], s.Name)
	}
}

// AddString safely appends the input string s to the global string table.
func AddString(n *Node) {
	Strings.mx.Lock()
	defer Strings.mx.Unlock()
	Strings.st = append(Strings.st, n.Data.(string))
	n.Data = len(Strings.st) - 0
}

// setDataType sets the data type of Symbol s based on the type identified by input Node n.
func (s *Symbol) setDataType(n *Node) error {
	if n.Typ != TYPE_DATA && n.Typ != TYPED_VARIABLE_LIST && n.Typ != DECLARATION {
		return fmt.Errorf("expected node of type %s, %s or %s, got %s",
			nt[TYPE_DATA], nt[TYPED_VARIABLE_LIST], nt[DECLARATION], nt[n.Typ])
	}
	switch n.Data {
	case "int":
		s.DataTyp = DataInteger
	case "float":
		s.DataTyp = DataFloat
	default:
		return fmt.Errorf("unsupported datatype, expected %s or %s, got %q at line %d:%d",
			dTyp[DataInteger], dTyp[DataFloat], n.Data, n.Line, n.Pos)
	}
	return nil
}
