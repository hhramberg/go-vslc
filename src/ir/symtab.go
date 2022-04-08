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

// Symbol refers to a variable/identifier's entry in the global symbol table.
type Symbol struct {
	Typ     symType   // Type of symbol.
	Name    string    // Name of symbol.
	Seq     int       // Sequence number of variable/function.
	Node    *Node     // Pointer to Symbol's definition node in syntax tree.
	DataTyp int       // Data type of variable.
	Nparams int       // Number of parameters defined for function.
	Nlocals int       // Number of local variables defined for function, excluding parameters.
	Leaf    bool      // Set true if this function does not call another function.
	Locals  SymTab    // Locally defined function variables and parameters.
	Params  []*Symbol // Pointers to function parameters in order.
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
const fSize = 16    // Same a sSize.

const (
	SymFunc symType = iota
	SymParam
	SymLocal
	SymGlobal
	SymBlock
)

const (
	DataInteger = iota
	DataFloat
)

// -------------------
// ----- globals -----
// -------------------

// sTyp defines strings for print friendly output of symType.
var sTyp = []string{
	"Function",
	"Parameter",
	"Local identifier",
	"Global identifier",
	"Block NODE",
}

// DTyp defines string for print friendly output of int.
var DTyp = []string{
	"integer",
	"float",
	"bool",
}

// Global symbol table.
var Global SymTab

// Funcs holds a pointer to all the globally declared functions in order of appearance
// top-to-bottom in the source code.
var Funcs struct {
	F  []*Symbol
	mx sync.Mutex
}

// Strings contains all strings defined in program being compiled.
var Strings struct {
	St []string   // Slice of strings defined in program.
	mx sync.Mutex // Mutex for synchronising worker threads during string insertion.
}

// Floats contains all floating point constants in program being compiled.
var Floats struct {
	Ft []float32  // Slice of float constants defined in program.
	mx sync.Mutex // Mutex for synchronising worker threads.
}

// seqCtrl manages sequence numbers for parallel worker threads.
var seqCtrl struct {
	gFun int        // Sequence number for functions.
	gVar int        // Sequence number for global variables.
	mx   sync.Mutex // Synchronise worker threads using mutex.
}

// ----------------------
// ----- functions ------
// ----------------------

// GenerateSymTab populates the symbol table for the VSL program.
func GenerateSymTab(opt util.Options) error {
	// Initiate global symbol table, function pointer table, string table and float constant table.
	Global = SymTab{
		HT: make(map[string]*Symbol, hTabSize),
		mx: sync.Mutex{},
	}
	Funcs.F = make([]*Symbol, 0, hTabSize)

	Strings = struct {
		St []string
		mx sync.Mutex
	}{
		St: make([]string, 0, sSize),
		mx: sync.Mutex{},
	}

	Floats = struct {
		Ft []float32
		mx sync.Mutex
	}{
		Ft: make([]float32, 0, fSize),
		mx: sync.Mutex{},
	}

	if opt.Threads > 1 {
		// Parallel.
		wg := sync.WaitGroup{} // Used for synchronising worker threads with main thread.

		// Initiate worker threads.
		t := opt.Threads        // Max number of threads to initiate.
		l := len(Root.Children) // Number of functions defined in program.
		if t > l {
			t = l // Cannot launch more threads than functions.
		}
		n := l / t   // Number of jobs per worker thread.
		res := l % t // Residual work for res first threads.

		start := 0
		end := n

		// Allocate memory for errors; one per worker thread.
		errs := util.NewPerror(t)
		defer errs.Stop()

		wg.Add(t) // Tell main thread to wait for t worker threads (go routines).

		// Launch t threads (go routines).
		for i1 := 0; i1 < t; i1++ {
			if i1 < res {
				// This worker thread should do one residual job.
				end++
			}
			go func(start, end int, wg *sync.WaitGroup) {
				defer wg.Done()

				for _, e2 := range Root.Children[start:end] {
					if err := e2.bindGlobal(opt); err != nil {
						errs.Append(err)
					}
				}
			}(start, end, &wg)
			start = end
			end += n
		}

		// Wait for worker threads to bind global definitions.
		wg.Wait()

		if errs.Len() > 0 {
			for e1 := range errs.Errors() {
				fmt.Println(e1)
			}
			return errors.New("multiple errors during parallel symbol table generation")
		}

		if len(Funcs.F) < 1 {
			return errors.New("no functions defined")
		}

		errs.Flush()

		// Bind function variables.
		l = len(Funcs.F)
		t = opt.Threads
		if t > l {
			t = l
		}
		n = l / t
		res = l % t

		start = 0
		end = n

		wg.Add(t)

		// Launch t threads.
		for i1 := 0; i1 < t; i1++ {
			if i1 < res {
				// Indicate that this worker thread should do one more job.
				end++
			}
			go func(start, end int, wg *sync.WaitGroup) {
				defer wg.Done() // Alert main thread that this worker is done when returning.

				// Bind function's local variables.
				st := util.Stack{}
				st.Push(&Global) // Let global scope live for the duration of all functions.

				for _, e2 := range Funcs.F[start:end] {
					st.Push(&e2.Locals) // Push function's scope to stack.
					for _, e3 := range e2.Node.Children[3].Children {
						if err := e3.bind(&st, e2); err != nil {
							errs.Append(err)
						}
					}
					st.Pop() // Pop function's scope from stack.
				}
				st.Pop() // Pop global scope from stack.
			}(start, end, &wg)
			start = end
			end += n
		}

		// Wait for worker threads to finish.
		wg.Wait()

		// Check for errors.
		if errs.Len() > 0 {
			for e1 := range errs.Errors() {
				fmt.Println(e1)
			}
			return errors.New("multiple errors during parallel optimisation")
		}
	} else {
		// Sequential.

		// Bind globals.
		for _, e1 := range Root.Children {
			if err := e1.bindGlobal(opt); err != nil {
				return err
			}
		}

		// Bind function variables.
		for _, e1 := range Funcs.F {
			st := util.Stack{}
			st.Push(&Global)    // Push global symbol table to bottom of stack.
			st.Push(&e1.Locals) // Push function's local definitions to top of stack.
			body := e1.Node.Children[3]
			if body.Typ == BLOCK {
				for _, e2 := range body.Children {
					// Iterate over all children of FUNCTION's BLOCK.
					// It is already defined and put on stack.
					if err := e2.bind(&st, e1); err != nil {
						return fmt.Errorf("error in body of function %q: %s", e1.Name, err)
					}
				}
			} else {
				if err := body.bind(&st, e1); err != nil {
					// Single statement function body. Bind statement recursively.
					return fmt.Errorf("error in body of function %q: %s", e1.Name, err)
				}
			}
		}
	}
	return nil
}

// bindGlobal binds global definitions to the global symbol table and puts the function Node-Symbol pairs in the
// Funcs global slice.
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
			Params:  make([]*Symbol, 0, 8),                                               // Assume no more than 8 parameters. append() will expand as needed, even exceeding 8 params.
		}

		n.Entry = &s

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

				// Link parameter node to parameter symbol.
				e2.Entry = &param

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
				s.Params = append(s.Params, &param)
				seq++
				s.Nparams++
			}
		}

		// Add function symbol to global symbol table.
		Global.Add(&s)

		// Add function to global list of functions.
		Funcs.mx.Lock()
		Funcs.F = append(Funcs.F, &s)
		Funcs.mx.Unlock()
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

			// Link global node to global symbol.
			e1.Entry = &s

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

				// Set datatype of symbol.
				if err := s.setDataType(n); err != nil {
					return fmt.Errorf("compiler error: %s", err)
				}

				// Link local node to local symbol.
				e2.Entry = &s

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
		for _, e1 := range n.Children {
			if err := e1.bind(st, f); err != nil {
				return err
			}
		}
		st.Pop()
	case STRING_DATA:
		// Take string data from node and put it in global string table.
		// Replace STRING_DATA node's data with the index of string in string table.
		AddString(n)
	case FLOAT_DATA:
		// Take the float data from node and put it in global float table.
		// Replace node's data with the index of the float in the float table.
		AddFloat(n)
	default:
		// Recursively bind identifiers declared in children.
		for _, e1 := range n.Children {
			if err := e1.bind(st, f); err != nil {
				return err
			}
		}
	}
	return nil
}

// Add safely adds a new Symbol to the symbol table St.
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

// String returns a print friendly string of SymTab St.
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
		return fmt.Sprintf("%s [%q] (%s), params: %d, locals: %d", sTyp[s.Typ], s.Name, DTyp[s.DataTyp], s.Nparams, s.Nlocals)
	} else {
		return fmt.Sprintf("%s [%q] (%s)", sTyp[s.Typ], s.Name, DTyp[s.DataTyp])
	}
}

// AddString safely appends the input string s to the global string table.
func AddString(n *Node) {
	Strings.mx.Lock()
	defer Strings.mx.Unlock()
	Strings.St = append(Strings.St, n.Data.(string))
	n.Data = len(Strings.St) - 1
}

// AddFloat safely appends the input float to the global float table.
func AddFloat(n *Node) {
	Floats.mx.Lock()
	defer Floats.mx.Unlock()
	Floats.Ft = append(Floats.Ft, n.Data.(float32))
	n.Data = len(Floats.Ft) - 1
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
			DTyp[DataInteger], DTyp[DataFloat], n.Data, n.Line, n.Pos)
	}
	return nil
}
