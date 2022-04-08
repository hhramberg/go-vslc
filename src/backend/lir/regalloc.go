// Package lir provides functions for transforming the lightweight intermediate representation (LIR) into
// either ARMv8 or RISC-V assembly.
package lir

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"vslc/src/backend/arm"
	"vslc/src/backend/regfile"
	"vslc/src/ir/lir"
	"vslc/src/ir/lir/types"
	"vslc/src/util"
)

// ----------------------------
// ----- Type definitions -----
// ----------------------------

// node represents a register interference graph node element.
type node struct {
	val        *lir.Value // LIR Value being wrapped.
	neighbours []*node    // Neighbours of val in register interference graph (RIG).
	enabled    bool       // Set to true if "present" in RIG. Set to false if disabled, "not present", in RIG.
	spill      bool       // Set to true if this variable should be spilled to memory.
}

// ---------------------
// ----- Constants -----
// ---------------------

// Number of times register allocation will retry finding a node with fewer than k neighbours before failing.
const retry = 128

// -------------------
// ----- Globals -----
// -------------------

// ---------------------
// ----- Functions -----
// ---------------------

// calcLiveness calculates liveness of all virtual registers lir.Value of every function of lir.Module m.
func calcLiveness(opt util.Options, m *lir.Module) ([][]*node, error) {
	if m == nil {
		return nil, errors.New("lir module is <nil>")
	}

	rigs := make([][]*node, len(m.Functions()))
	if opt.Threads > 1 {
		// Parallel.
		t := opt.Threads
		l := len(m.Functions())
		if t > l {
			t = l
		}
		n := t / l
		res := t % l

		// Error listener.
		perr := util.NewPerror(t)
		cres := make(chan [][]*node, t)
		wg := sync.WaitGroup{}
		wg.Add(t)

		start := 0
		end := n

		// Spawn t worker go routines.
		for i1 := 0; i1 < t; i1++ {
			if res < i1 {
				end++
			}

			// Spawn worker go routine.
			go func(start, end int, wg *sync.WaitGroup, cres chan [][]*node) {
				defer wg.Done()
				rigs := make([][]*node, 0, end-start)
				for i1, e1 := range m.Functions()[start:end] {
					if rig, err := calcLivenessFunc(e1); err != nil {
						perr.Append(err)
					} else {
						rigs[i1] = rig
					}
				}
				cres <- rigs
			}(start, end, &wg, cres)

			start = end
			end += n
		}

		wg.Wait()
		if perr.Len() > 0 {
			for e1 := range perr.Errors() {
				fmt.Println(e1)
			}
			return nil, fmt.Errorf("%d error(s) during parallel liveness calculation", perr.Len())
		}

		// Append RIG from every worker go routine to main result RIG.
		for e1 := range cres {
			rigs = append(rigs, e1...)
		}
	} else {
		// Sequential.
		for i1, e1 := range m.Functions() {
			// Calculate separately for every function.
			if rig, err := calcLivenessFunc(e1); err != nil {
				return nil, err
			} else {
				rigs[i1] = rig
			}
		}
	}
	return rigs, nil
}

// calcLivenessFunc calculates liveness for all virtual registers lir.Value of lir.Function f.
// Procedure from: https://www.cl.cam.ac.uk/teaching/1819/OptComp/slides/lecture03.pdf
func calcLivenessFunc(f *lir.Function) ([]*node, error) {
	l := 0
	for _, e1 := range f.Blocks() {
		l += len(e1.Instructions())
	}
	vars := make([]*node, 0, l) // Variable graph that's pre-allocated with the number of instructions in function f.

	// State of instructions at any time.
	live := make([]*node, 0, l)

	for _, e1 := range f.Blocks() {
		for _, e2 := range e1.Instructions() {
			n := &node{
				val:     e2,
				enabled: true,
			}
			vars = append(vars, n)
			(*e2).SetWrapper(n)
		}
	}

	// Find liveness by backward flow.
	for i1 := len(vars) - 1; i1 >= 0; i1-- {
		e1 := &vars[i1]
		ref := ref(*e1)

		// Check for variables referenced by instruction e1.
		for _, e2 := range ref {
			for _, e3 := range live {
				if (*e2.val).Id() == (*e3.val).Id() {
					// Already live.
					goto cont
				}
			}
			// Append unreferenced variable to live variable.
			live = append(live, e2)
		cont:
		}

		// Check for variables declared by instruction e1.
		def := def(*e1)
		if len(def) > 0 {
			// Declares variable, set variable dead by removing it from the live slice.
			for i1, e1 := range live {
				if (*def[0].val).Id() == (*e1.val).Id() {
					// Delete from live.
					// Order is not important, use fast method:
					// https://stackoverflow.com/questions/37334119/how-to-delete-an-element-from-a-slice-in-golang
					live[i1] = live[len(live)-1]
					live = live[:len(live)-1]
					break
				}
			}
		}
		(*e1).neighbours = make([]*node, 0, len(live))
		(*e1).neighbours = append((*e1).neighbours, live...)
	}

	// TODO: Dev delete printout.
	for _, e1 := range vars {
		fmt.Println(e1.String())
	}
	return vars, nil
}

// AllocateRegisters uses the graph colouring algorithm to assign virtual values a physical register, based on
// the target type provided by the util.Options configuration file opt.
func AllocateRegisters(opt util.Options, m *lir.Module) error {
	// Procedure from: http://web.cecs.pdx.edu/~mperkows/temp/register-allocation.pdf

	// Create virtual register file.
	var rf regfile.RegisterFile
	if opt.TargetArch == util.Aarch64 {
		rf = arm.CreateRegisterFile2()
	} else if opt.TargetArch == util.Riscv32 || opt.TargetArch == util.Riscv64 {
		//rf = riscv.CreateRegisterFile(opt)
		return errors.New("risc-v target not implemented yet") // TODO: Implement.
	} else {
		return errors.New("unsupported target architecture")
	}
	if rf == nil {
		return errors.New("failed to initiate target virtual register file")
	}

	// Find temporaries' dependencies using live variable analysis on virtual registers.
	// calcLiveness returns the register interference graphs (RIG) of all lir.Functions in lir.Module m.
	rigs, err := calcLiveness(opt, m)
	if err != nil {
		fmt.Printf("variable liveness analysis failed: %s\n", err)
	}

	if opt.Threads > 1 {
		// Parallel.
		t := opt.Threads
		l := len(rigs)
		if t > l {
			t = l
		}
		n := l / t
		res := n % t

		start := 0
		end := n

		// Create error listener.
		perr := util.NewPerror(t)

		// Create wait group for main go routine to wait for worker go routines.
		wg := sync.WaitGroup{}
		wg.Add(t)

		// Spawn t worker go routines.
		for i1 := 0; i1 < l; i1++ {
			if res < i1 {
				end++
			}

			// Spawn worker go routine.
			go func(start, end int, wg *sync.WaitGroup) {
				defer wg.Done()
				for _, e1 := range rigs[start:end] {
					// Pass register file rf by value, not pointer, such that every go routine gets it very own copy.
					if err := allocateRegisterFunc(rf, e1); err != nil {
						perr.Append(err)
					}
				}
			}(start, end, &wg)

			start = end
			end += n
		}

		// Wait for worker go routines to finish register allocation.
		wg.Wait()

		// Check for errors from worker go routines.
		if perr.Len() > 1 {
			for e1 := range perr.Errors() {
				fmt.Println(e1)
			}
			return fmt.Errorf("%d error(s) during parallel register allocation", perr.Len())
		}
		return nil
	} else {
		// Sequential.
		for _, e1 := range rigs {
			if err := allocateRegisterFunc(rf, e1); err != nil {
				return nil
			}
		}
	}
	return nil
}

// allocateRegisterFunc allocates physical registers to an lir.Function's virtual registers. An error is returned
// if something wen't wrong.
func allocateRegisterFunc(rf regfile.RegisterFile, rig []*node) error {
	// Assign physical registers to virtual registers using the virtual register file.
	// "Remove" nodes from RIG and put them on stack.
	stack := util.Stack{}
	rt := retry // Retry removing nodes this many times before reporting failure.
	for stack.Size() < len(rig) && rt > 0 {
		// Keep removing nodes until all nodes are removed.
		// Bottom-up to preserve result from live variable analysis.
		for i2 := len(rig) - 1; i2 >= 0; i2-- {
			e2 := rig[i2]
			if e2.enabled {
				var k int
				if (*e2.val).DataType() == types.Int {
					// Integer data.
					k = rf.Ki()
				} else {
					// Floating point data.
					k = rf.Kf()
				}

				// If the below check fails, we'll hope to catch it in some later retry iteration.
				// That's why we have the outer loop that checks rt against the constant retry.
				if len(e2.GetEnabledNeighbours()) < k {
					e2.enabled = false // "Remove" val from RIG.
					stack.Push(e2)     // Push val on stack.
				}
			}
		}
		rt--
	}

	// Check for RIG node removal failure.
	if rt < 1 {
		return fmt.Errorf("could not untangle register interference graph within %d retries", retry)
	}

	// Pop nodes from stack and assign registers.
	for n := stack.Pop(); n != nil; n = stack.Pop() {
		var r regfile.Register // Physical register to be allocated to val n's Value.
		n.(*node).enabled = true

		// Check for datatype of Value. No need to assign physical register to branch instructions etc.
		if (*n.(*node).val).Type() != types.Data && (*n.(*node).val).Type() != types.Load {
			continue
		}

		// Check neighbours for allocated registers.
		en := n.(*node).GetEnabledNeighbours()    // Enabled neighbours.
		excl := make([]regfile.Register, len(en)) // Exclusion slice.
		for i1, e1 := range en {
			excl[i1] = (*e1.val).GetHW().(regfile.Register)
		}
		if (*n.(*node).val).DataType() == types.Int {
			r = rf.GetNextTempIExclude(excl)
		} else {
			r = rf.GetNextTempFExclude(excl)
		}

		// Check for registering spilling.
		if r == nil {
			// TODO: Implement register spilling.
			return errors.New("register spilling not implemented yet")
		}

		// Allocate physical register to virtual register.
		(*n.(*node).val).SetHW(r)
	}

	// TODO: Delete dev printout.
	for _, e2 := range rig {
		if (*e2.val).GetHW() == nil {
			fmt.Printf("%s was not assigned\n", (*e2.val).Name())
			continue
		}
		fmt.Printf("%s was assigned register %s\n",
			(*e2.val).Name(),
			(*e2.val).GetHW().(regfile.Register).String())
	}
	return nil
}

// GetNumberOfNeighbours returns the number of enabled (deleted) neighbours of val n.
func (n *node) GetNumberOfNeighbours() int {
	count := 0
	for _, e1 := range n.neighbours {
		if e1.enabled {
			count++
		}
	}
	return count
}

// GetEnabledNeighbours returns all neighbours of val n that are enabled.
func (n *node) GetEnabledNeighbours() []*node {
	res := make([]*node, 0, len(n.neighbours))
	for _, e1 := range n.neighbours {
		if e1.enabled {
			res = append(res, e1)
		}
	}
	return res
}

// String creates a print friendly string representing this node. It returns a string of the instruction
// ln.val and the live/neighbour variables at the instructions point in the program.
func (n *node) String() string {
	if len(n.neighbours) > 0 {
		sb := strings.Builder{}
		for i1, e1 := range n.neighbours {
			sb.WriteString((*e1.val).Name())
			if i1 < len(n.neighbours)-1 {
				sb.WriteRune(',')
				sb.WriteRune(' ')
			}
		}
		return fmt.Sprintf("%s\tLive: {%s}", (*n.val).String(), sb.String())
	}
	return fmt.Sprintf("%s\tLive: {}", (*n.val).String())
}

// ref returns the local virtual registers that are referenced by lir.Value v. If no variables are referenced, ref returns nil.
func ref(n *node) []*node {
	v := n.val
	if (*v).Has2Operands() {
		op1 := (*v).GetOperand1().GetWrapper().(*node)
		op2 := (*v).GetOperand2().GetWrapper().(*node)
		return []*node{op1, op2}
	}
	if (*v).Has1Operand() {
		op1 := (*v).GetOperand1().GetWrapper().(*node)
		return []*node{op1}
	}
	return nil
}

// def returns the local virtual registers that is defined, if any. Else it returns <nil>.
func def(n *node) []*node {
	v := n.val
	if (*v).Type() == types.Data || (*v).Type() == types.Load {
		return []*node{(*v).GetWrapper().(*node)}
	}
	return nil
}
