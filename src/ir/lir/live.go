package lir

import (
	"fmt"
	"strings"
	"sync"
	"vslc/src/ir/lir/types"
	"vslc/src/util"
)

// ----------------------------
// ----- Type definitions -----
// ----------------------------

// LiveNode wraps an ir.Value instruction and its dependencies.
type LiveNode struct {
	Val     Value       // Val is the ir.Value instruction that is wrapped by the LiveNode.
	Dep     []*LiveNode // Dep is the dependencies of the wrapped ir.Value node Val.
	Enabled bool        // Set to true if the LiveNode is present in the graph. Set to false if it should be disabled.
	Spill   bool        // Set to true if the hardware register has to be spilled.
	Reg     interface{} // Hardware register assigned to Value Val.
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

// CalcLiveness calculates the virtual register liveness of Module m.
// The parameter p is the maximum number of threads allowed to run in parallel.
func CalcLiveness(opt util.Options, m *Module) [][]*LiveNode {
	// Initiate global wrappers used for load and store instructions.
	for _, e1 := range m.globals {
		e1.SetHW(&LiveNode{
			Val: e1,
		})
	}
	for _, e1 := range m.strings {
		e1.SetHW(&LiveNode{
			Val: e1,
		})
	}

	// Calculate liveness per function.
	rigs := make([][]*LiveNode, len(m.Functions()))
	if opt.Threads > 1 {
		// Parallel.
		t := opt.Threads
		l := len(m.Functions())
		if t > l {
			t = l
		}
		n := l / t
		res := l % t

		start := 0
		end := n

		wg := sync.WaitGroup{}

		// Spawn t worker go routines.
		wg.Add(t)
		for i1 := 0; i1 < t; i1++ {
			if i1 < res {
				end++
			}

			// Spawn worker go routine.
			go func(start, end int, wg *sync.WaitGroup) {
				defer wg.Done()
				i2 := start
				for _, e2 := range m.Functions()[start:end] {
					rigs[i2] = calcLivenessFunction(e2)
					i2++
				}
			}(start, end, &wg)

			start = end
			end += n
		}

		// Wait for worker go routines to finish.
		wg.Wait()
	} else {
		// Sequential.
		for i1, e1 := range m.Functions() {
			rig := calcLivenessFunction(e1)
			rigs[i1] = rig
		}
	}
	return rigs
}

// String creates a print friendly string representing this node. It returns a string of the instruction
// ln.val and the live/neighbour variables at the instructions point in the program.
func (n *LiveNode) String() string {
	if len(n.Dep) > 0 {
		sb := strings.Builder{}
		for i1, e1 := range n.Dep {
			sb.WriteString(e1.Val.Name())
			if i1 < len(n.Dep)-1 {
				sb.WriteString(", ")
			}
		}
		return fmt.Sprintf("%s\tLive: {%s}", n.Val.String(), sb.String())
	}
	return fmt.Sprintf("%s\tLive: {}", n.Val.String())
}

// GetNumberOfNeighbours returns the number of enabled neighbours of val n.
func (n *LiveNode) GetNumberOfNeighbours() int {
	count := 0
	for _, e1 := range n.Dep {
		if e1.Enabled {
			count++
		}
	}
	return count
}

// GetEnabledNeighbours returns all neighbours of val n that are enabled.
func (n *LiveNode) GetEnabledNeighbours() []*LiveNode {
	res := make([]*LiveNode, 0, len(n.Dep))
	for _, e1 := range n.Dep {
		if e1.Enabled {
			res = append(res, e1)
		}
	}
	return res
}

// calcLivenessFunction calculates virtual register liveness throughout the function body.
func calcLivenessFunction(f *Function) []*LiveNode {
	l := 0
	for _, e1 := range f.Blocks() {
		l += len(e1.Instructions())
	}
	vars := make([]*LiveNode, 0, l)
	live := make([]*LiveNode, 0, l)

	// Bind parameters.
	for _, e1 := range f.params {
		e1.SetHW(&LiveNode{
			Val: e1,
		})
	}

	// Bind locally declared variables.
	for _, e1 := range f.variables {
		e1.SetHW(&LiveNode{
			Val: e1,
		})
	}

	// Fill instructions.
	for _, e1 := range f.Blocks() {
		for _, e2 := range e1.Instructions() {
			n := &LiveNode{
				Val:     e2,
				Enabled: true,
			}
			e2.SetHW(n)
			vars = append(vars, n)
		}
	}

	// Fill live.
	for i1 := len(vars) - 1; i1 >= 0; i1-- {
		// Reverse order; from end of function to top of function.
		e1 := vars[i1]

		// Check for virtual registers referenced by instruction.
		for _, e2 := range ref(e1) {
			for _, e3 := range live {
				if e3.Val.Id() == e2.Val.Id() {
					// Already live.
					goto cont
				}
			}
			// Append unreferenced variable to live variables.
			live = append(live, e2)
		cont:
		}

		// Check for virtual registers defined by instruction.
		if def := def(e1); def != nil {
			// Variable declared. Remove from live slice.
			for i2, e2 := range live {
				if def.Val.Id() == e2.Val.Id() {
					// Delete from live. Order is unimportant. Use fast method.
					// https://stackoverflow.com/questions/37334119/how-to-delete-an-element-from-a-slice-in-golang
					live[i2] = live[len(live)-1]
					live = live[:len(live)-1]
					break
				}
			}
		}

		e1.Dep = make([]*LiveNode, 0, len(live))
		e1.Dep = append(e1.Dep, live...)
	}
	return vars
}

// ref returns a slice of operands that are referenced by the ir.Value instruction wrapped by LiveNode n.
// If no ir.Value instructions are referenced, <nil> is returned.
func ref(n *LiveNode) []*LiveNode {
	v := n.Val

	// Loads reference external data: no dependencies.
	if v.Type() == types.LoadInstruction {
		return nil
	}

	// Function calls have multiple dependencies.
	if v.Type() == types.FunctionCallInstruction {
		res := make([]*LiveNode, len(v.(*FunctionCallInstruction).arguments))
		for i1, e1 := range v.(*FunctionCallInstruction).arguments {
			res[i1] = e1.GetHW().(*LiveNode)
		}
		return res
	}

	// VaLists have multiple dependencies.
	if v.Type() == types.DataInstruction && v.DataType() == types.VaList {
		res := make([]*LiveNode, len(v.(*VaList).vars))
		for i1, e1 := range v.(*VaList).vars {
			res[i1] = e1.GetHW().(*LiveNode)
		}
		return res
	}

	// Remaining instructions are two or three address code instructions.
	if op1 := v.Operand1(); op1 != nil {
		res := make([]*LiveNode, 1, 2)
		res[0] = op1.GetHW().(*LiveNode)
		if op2 := v.Operand2(); op2 != nil {
			res = append(res, op2.GetHW().(*LiveNode))
		}
		return res
	}
	return nil
}

// def returns the virtual register ir.Value that is defined by instruction wrapped by LiveNode n, if it
// generates a new virtual register ir.Value.
func def(n *LiveNode) *LiveNode {
	v := n.Val
	if v.Type() == types.DataInstruction ||
		v.Type() == types.LoadInstruction ||
		v.Type() == types.FunctionCallInstruction ||
		v.Type() == types.Constant ||
		v.Type() == types.CastInstruction {
		return v.GetHW().(*LiveNode)
	}
	return nil
}
