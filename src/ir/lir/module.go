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

// Module defines a program that contains globals and functions.
type Module struct {
	Name       string               // Name of module. Not important.
	globals    []*Global            // Global variables.
	functions  map[string]*Function // All functions defined in module.
	strings    []*Global            // Globally declared strings.
	seq        int                  // Sequence number used for assigning unique identifiers to every child of module.
	sync.Mutex                      // Mutex for synchronising access to the module during parallel execution.
}

// ---------------------
// ----- Constants -----
// ---------------------

// labelFunctionPrefix is used when assigning names to Function when no name is given.
const labelFunctionPrefix = "func"

// -------------------
// ----- globals -----
// -------------------

// ---------------------
// ----- functions -----
// ---------------------

// CreateModule creates a new empty module with the given optional name.
func CreateModule(name string) *Module {
	m := Module{
		globals:   make([]*Global, 0, 16),
		functions: make(map[string]*Function, 16),
		strings:   make([]*Global, 0, 16),
	}
	if len(name) > 0 {
		m.Name = name
	} else {
		m.Name = "LIR Module"
	}
	return &m
}

// String returns a textual representation of the module.
func (m *Module) String() string {
	sb := strings.Builder{}
	sb.WriteString(fmt.Sprintf("Module: %s\n\n", m.Name))

	// Add globals.
	for _, e1 := range m.globals {
		sb.WriteString(e1.String())
		sb.WriteRune('\n')
	}

	if len(m.globals) > 0 {
		sb.WriteRune('\n')
	}

	// Add functions.
	for _, e1 := range m.functions {
		sb.WriteString(e1.String())
		sb.WriteRune('\n')
	}
	return sb.String()
}

// CreateGlobalInt creates a global variable of the given type and optional name.
func (m *Module) CreateGlobalInt(name string) *Global {
	m.Lock()
	defer m.Unlock()
	g := &Global{
		id:  m.seq,
		typ: types.Int,
		val: nil,
	}
	m.seq++
	if len(g.name) > 0 {
		g.name = name
	} else {
		g.name = fmt.Sprintf("%s%d", globalLabelPrefix, g.id)
	}
	m.globals = append(m.globals, g)
	return g
}

// CreateGlobalFloat creates a global variable of the given type and optional name.
func (m *Module) CreateGlobalFloat(name string) *Global {
	m.Lock()
	defer m.Unlock()
	g := &Global{
		id:  m.seq,
		typ: types.Float,
		val: nil,
	}
	m.seq++
	if len(g.name) > 0 {
		g.name = name
	} else {
		g.name = fmt.Sprintf("%s%d", globalLabelPrefix, g.id)
	}
	m.globals = append(m.globals, g)
	return g
}

// CreateString creates a string that's stored in the module's global data. The returned value is a pointer to the
// string, similar to a C-style char pointer.
func (m *Module) CreateString(s string) *Global {
	m.Lock()
	defer m.Unlock()
	g := &Global{
		id:   m.seq,
		name: "",
		typ:  types.String,
		val:  s,
	}
	m.seq++
	m.strings = append(m.strings, g)
	return g
}

// CreateFunction creates a new empty function given return data type rtyp, function parameters params and
// function name.
//
// From the below C-style function we have the following attributes.
// name: foo
// rtyp: int
// params: [a int, b float, x int]
//
// int foo(int a, float b, int x){
// 	   ...
// }
func (m *Module) CreateFunction(rtyp types.DataType, name string) (*Function, error) {
	if rtyp < types.Int || rtyp > types.Float {
		return nil, fmt.Errorf("cannot create function because the provided return datatype is neither %s nor %s",
			types.Int.String(), types.Float.String())
	}

	m.Lock()
	defer m.Unlock()
	f := &Function{
		m:         m,
		id:        m.seq,
		typ:       rtyp,
		params:    make([]*Param, 0, 8), // Assume 8 parameters.
		variables: make([]Value, 0, 8),  // Assume 8 local variables.
		blocks:    make([]*Block, 0, 8), // Assume at most 8 basic blocks. It' a reasonable amount for a simple function.
	}
	m.seq++
	if len(name) > 0 {
		f.name = name
	} else {
		f.name = fmt.Sprintf("%s%d", labelFunctionPrefix, f.id)
	}
	m.functions[f.name] = f
	return f, nil
}

// Globals returns a slice of all functions declared in Module m.
func (m *Module) Globals() []*Global {
	return m.globals
}

// Functions returns a slice of all functions declared in Module m.
func (m *Module) Functions() []*Function {
	res := make([]*Function, len(m.functions))
	i1 := 0
	for _, e1 := range m.functions {
		res[i1] = e1
	}
	return res
}

// Strings returns a slice of all strings declared in Module m.
func (m *Module) Strings() []*Global {
	res := make([]*Global, len(m.strings))
	i1 := 0
	for _, e1 := range m.strings {
		res[i1] = e1
	}
	return res
}

// getId returns a unique sequence number that can be assigned to any data object in the Module m.
func (m *Module) getId() int {
	m.Lock()
	defer m.Unlock()
	res := m.seq
	m.seq++
	return res
}

// GetGlobal returns a named global variable of Module m, if it exits. If no global with the given
// name exits, nil is returned.
func (m *Module) GetGlobal(name string) *Global {
	for _, e1 := range m.globals {
		if e1.name == name {
			return e1
		}
	}
	return nil
}

// GetFunction returns a named function of Module m, if it exits. If no function with the given
// name exits, nil is returned.
func (m *Module) GetFunction(name string) *Function {
	for _, e1 := range m.functions {
		if e1.name == name {
			return e1
		}
	}
	return nil
}
