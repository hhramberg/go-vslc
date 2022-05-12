package lir

import (
	"fmt"
	"strings"
	"sync"
	"vslc/src/ir/lir/types"
)

// ----------------------------
// ----- Type definitions -----
// ----------------------------

// Module defines the global scope of the lightweight intermediate representation.
type Module struct {
	name       string               // name defines the module name.
	functions  []*Function          // functions defines the globally declared functions of the program.
	globals    []*Global            // globals defines the globally declared variables of the program.
	fmap       map[string]*Function // A hash map for quickly accessing globally declared functions.
	gmap       map[string]*Global   // A hash map for quickly accessing globally declared variables.
	constants  []*Constant          // All constants are linked globally in case they need to be loaded from global data instead of immediate values.
	strings    []*String            // strings declares the string data used in the program.
	seq        int                  // seq is the global sequence number that generates unique identifiers for global LIR objects.
	sync.Mutex                      // Mutex synchronizes worker go routine access to global data.
}

// ---------------------
// ----- Constants -----
// ---------------------

// labelString defines the prefix for globally declared static strings.
const labelString = "_STR_"

// defaultModuleName defines the default name of any newly created Modules where no name was provided at time of creation.
const defaultModuleName = "LIR Module"

// gSize pre-defines a reasonable number of functions and global identifiers for elementary and small programs.
const gSize = 16

// fSize pre-defines a reasonable number of parameters, local variables and basic blocks for functions.
const fSize = 8

// -------------------
// ----- Globals -----
// -------------------

// reservedNames contains reserved function names that cannot be used..
var reservedNames = [...]string{
	"printf",
	"main",
	"atoi",
	"atof",
}

// ---------------------
// ----- Functions -----
// ---------------------

// CreateModule creates a new LIR module with the optional name.
func CreateModule(name string) *Module {
	m := &Module{
		functions: make([]*Function, 0, gSize),
		fmap:      make(map[string]*Function),
		gmap:      make(map[string]*Global),
		constants: make([]*Constant, 0, gSize),
		strings:   make([]*String, 0, gSize),
		Mutex:     sync.Mutex{},
		seq:       1 << 20, // Offset by a large number, because function's local sequence numbers start at 0.
	}
	if len(name) > 0 {
		m.name = name
	} else {
		m.name = defaultModuleName
	}
	return m
}

// Name returns the Module's name.
func (m *Module) Name() string {
	return m.name
}

// String returns the textual LIR representation of Module m.
func (m *Module) String() string {
	sb := strings.Builder{}
	sb.WriteString("module: ")
	sb.WriteString(m.name)
	sb.WriteRune('\n')
	sb.WriteRune('\n')

	// Append strings.
	if len(m.strings) > 0 {
		for _, e1 := range m.strings {
			sb.WriteString(e1.String())
			sb.WriteRune('\n')
		}
		sb.WriteRune('\n')
	}

	// Append constants.
	//for _, e1 := range m.constants {
	//
	//}

	// Append global variables.
	if len(m.globals) > 0 {
		for _, e1 := range m.globals {
			sb.WriteString(e1.String())
			sb.WriteRune('\n')
		}
		sb.WriteRune('\n')
	}

	// Append functions.
	for i1, e1 := range m.functions {
		sb.WriteString(e1.String())
		sb.WriteRune('\n')
		if i1 < len(m.functions)-1 {
			sb.WriteRune('\n')
		}
	}
	return sb.String()
}

// CreateGlobalInt creates a global variable of type integer.
func (m *Module) CreateGlobalInt(name string) *Global {
	if len(name) < 1 {
		panic("cannot create global: no name provided")
	}
	m.Lock()
	defer m.Unlock()
	if _, ok := m.fmap[name]; ok {
		panic(fmt.Sprintf("duplicate declaration: function with name %q already defined for module %s",
			name, m.name))
	}
	if _, ok := m.gmap[name]; ok {
		panic(fmt.Sprintf("duplicate declaration: global identifier %q already defined for module %s",
			name, m.name))
	}
	inst := &Global{
		m:    m,
		id:   m.seq,
		name: name,
		typ:  types.Int,
		en:   true,
	}
	m.seq++
	m.globals = append(m.globals, inst)
	m.gmap[name] = inst
	return inst
}

// CreateGlobalFloat creates a global variable of type floating point.
func (m *Module) CreateGlobalFloat(name string) *Global {
	if len(name) < 1 {
		panic("cannot create global: no name provided")
	}
	m.Lock()
	defer m.Unlock()
	if _, ok := m.fmap[name]; ok {
		panic(fmt.Sprintf("duplicate declaration: function with name %q already defined for module %s",
			name, m.name))
	}
	if _, ok := m.gmap[name]; ok {
		panic(fmt.Sprintf("duplicate declaration: global identifier %q already defined for module %s",
			name, m.name))
	}
	inst := &Global{
		m:    m,
		id:   m.seq,
		name: name,
		typ:  types.Float,
		en:   true,
	}
	m.seq++
	m.globals = append(m.globals, inst)
	m.gmap[name] = inst
	return inst
}

// createConstant appends a float or int constant to the Module m.
func (m *Module) createConstant(v Value) {
	if v.Type() != types.Constant && v.DataType() != types.Int && v.DataType() != types.Float {
		panic(fmt.Sprintf("cannot create constant: expected %s %s or %s %s, got %s %s",
			types.Constant.String(), types.Int.String(), types.Constant.String(),
			types.Float, v.Type().String(), v.DataType()))
	}
	m.Lock()
	defer m.Unlock()
	m.constants = append(m.constants, v.(*Constant))
}

// CreateGlobalString creates a global constant string.
func (m *Module) CreateGlobalString(s string) *String {
	if len(s) < 1 {
		panic("cannot create string constant: no string provided")
	}
	m.Lock()
	defer m.Unlock()
	str := &String{
		m:   m,
		id:  m.seq,
		val: s,
		en:  true,
	}
	m.seq++
	m.strings = append(m.strings, str)
	return str
}

// GetGlobalVariable returns a *Global variable if it exists. If it does not exist, <nil> is returned.
func (m *Module) GetGlobalVariable(name string) *Global {
	if g, ok := m.gmap[name]; ok {
		return g
	}
	return nil
}

// Globals returns a slice of all the globally declared variables of Module m.
func (m *Module) Globals() []*Global {
	return m.globals
}

// Strings returns a slice of all the constant string literals of Module m.
func (m *Module) Strings() []*String {
	return m.strings
}

// Constants returns a slice of all the declared constants defined for Module m.
func (m *Module) Constants() []*Constant {
	return m.constants
}

// CreateFunction creates a function header. The function body is defined when at least one basic block is added.
// Function parameters are added directly using the function's Function.CreateParam function.
func (m *Module) CreateFunction(name string, typ types.DataType) *Function {
	if typ > types.Float {
		panic(fmt.Sprintf("cannot create function: functions can only return %s or %s",
			types.Int.String(), types.Float.String()))
	}
	if len(name) < 1 {
		panic("cannot create function: no function name provided")
	}

	// Check for reserved function name.
	for _, e1 := range reservedNames {
		if e1 == name {
			panic(fmt.Sprintf("function name %q is a reserved function name", name))
		}
	}

	// Check for duplicate declarations.
	m.Lock()
	defer m.Unlock()
	if _, ok := m.fmap[name]; ok {
		panic(fmt.Sprintf("duplicate declaration: function %q already defined for module %s",
			name, m.name))
	}
	if _, ok := m.gmap[name]; ok {
		panic(fmt.Sprintf("duplicate declaration: global identifier %q already defined for module %s",
			name, m.name))
	}

	// Generate function header.
	f := &Function{
		m:         m,
		id:        m.seq,
		name:      name,
		typ:       typ,
		blocks:    make([]*Block, 0, fSize),
		params:    make([]*Param, 0, fSize),
		variables: make([]*DeclareInstruction, 0, fSize),
	}
	m.seq++
	m.functions = append(m.functions, f)
	m.fmap[name] = f
	return f
}

// GetFunction returns the named function if it exists. If it does not exist, <nil> is returned.
func (m *Module) GetFunction(name string) *Function {
	m.Lock()
	defer m.Unlock()
	if f, ok := m.fmap[name]; ok {
		return f
	}
	return nil
}

// Functions returns a slice of all the functions defined for Module m.
func (m *Module) Functions() []*Function {
	m.Lock()
	defer m.Unlock()
	return m.functions
}

// getId returns a unique sequence id from Module m.
func (m *Module) getId() int {
	m.Lock()
	defer m.Unlock()
	id := m.seq
	m.seq++
	return id
}
