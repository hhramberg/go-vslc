// Package lir provides functions for transforming the lightweight intermediate representation (LIR) into
// either ARMv8 or RISC-V assembly.
package lir

import (
	"errors"
	"fmt"
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
//type node struct {
//	val        *lir.Value // LIR Value being wrapped.
//	neighbours []*node    // Neighbours of val in register interference graph (RIG).
//	enabled    bool       // Set to true if "present" in RIG. Set to false if disabled, "not present", in RIG.
//	spill      bool       // Set to true if this variable should be spilled to memory.
//}

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

// AllocateRegisters uses the graph colouring algorithm to assign virtual values a physical register, based on
// the target type provided by the util.Options configuration file opt.
func AllocateRegisters(opt util.Options, m *lir.Module) error {
	// Procedure from: http://web.cecs.pdx.edu/~mperkows/temp/register-allocation.pdf

	// Create virtual register file.
	var rf regfile.RegisterFile
	if opt.TargetArch == util.Aarch64 {
		rf = arm.CreateRegisterFile()
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
	rigs := lir.CalcLiveness(opt, m)

	// Allocate hardware registers to the lir.LiveNodes wrapping the lir.Value.
	if opt.Threads > 1 {
		// Parallel.
		t := opt.Threads
		l := len(rigs)
		if t > l {
			t = l
		}
		n := l / t
		res := l % t

		start := 0
		end := n

		// Create error listener.
		perr := util.NewPerror(t)

		// Create wait group for main go routine to wait for worker go routines.
		wg := sync.WaitGroup{}
		wg.Add(t)

		// Spawn t worker go routines.
		for i1 := 0; i1 < t; i1++ {
			if i1 < res {
				end++
			}

			// Spawn worker go routine.
			go func(start, end int, wg *sync.WaitGroup) {
				defer wg.Done()
				for i2, e2 := range rigs[start:end] {
					// Pass register file rf by value, not pointer, such that every go routine gets its very own copy.
					if err := allocateRegisterFunc(opt, m.Functions()[start:end][i2], rf, e2); err != nil {
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
		for i1, e1 := range rigs {
			if err := allocateRegisterFunc(opt, m.Functions()[i1], rf, e1); err != nil {
				return nil
			}
		}
	}
	return nil
}

// allocateRegisterFunc allocates physical registers to an lir.Function's virtual registers. An error is returned
// if something wen't wrong.
func allocateRegisterFunc(opt util.Options, f *lir.Function, rf regfile.RegisterFile, rig []*lir.LiveNode) error {
	// Assign physical registers to virtual registers using the virtual register file.

	if opt.TargetArch != util.Riscv32 && opt.TargetArch != util.Riscv64 && opt.TargetArch != util.Aarch64 {
		return fmt.Errorf("register allocation for target architecture %d not supported", opt.TargetArch)
	}

	// "Remove" nodes from RIG and put them on stack.
	stack := util.Stack{}
	rt := retry // Retry removing nodes this many times before reporting failure.
	for stack.Size() < len(rig) && rt > 0 {
		// Keep removing nodes until all nodes are removed.
		// Bottom-up to preserve result from live variable analysis.
		for i2 := len(rig) - 1; i2 >= 0; i2-- {
			e2 := rig[i2]
			if e2.Enabled {
				var k int
				if e2.Val.DataType() == types.Int {
					// Integer data.
					k = rf.Ki()
				} else {
					// Floating point data.
					k = rf.Kf()
				}

				// If the below check fails, we'll hope to catch it in some later retry iteration.
				// That's why we have the outer loop that checks rt against the constant retry.
				if len(e2.GetEnabledNeighbours()) < k {
					e2.Enabled = false // "Remove" val from RIG.
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
		n.(*lir.LiveNode).Enabled = true

		// Exclusively assign d0 or x0 to return statement.
		if n.(*lir.LiveNode).Val.Type() == types.ReturnInstruction {
			typ := n.(*lir.LiveNode).Val.DataType()
			if typ == types.Int || typ == types.String {
				// Strings are addresses stored in register.
				n.(*lir.LiveNode).Val.GetHW().(*lir.LiveNode).Reg = rf.GetI(0)
			} else {
				n.(*lir.LiveNode).Val.GetHW().(*lir.LiveNode).Reg = rf.GetF(0)
			}
			continue
		}

		// Check for datatype of Value. No need to assign physical register to branch instructions etc.
		if n.(*lir.LiveNode).Val.Type() != types.DataInstruction &&
			n.(*lir.LiveNode).Val.Type() != types.LoadInstruction &&
			n.(*lir.LiveNode).Val.Type() != types.FunctionCallInstruction &&
			n.(*lir.LiveNode).Val.Type() != types.Constant &&
			n.(*lir.LiveNode).Val.Type() != types.CastInstruction {
			continue
		}

		var r regfile.Register // Physical register to be allocated to val n's Value.

		// Check neighbours for allocated registers.
		en := n.(*lir.LiveNode).GetEnabledNeighbours() // Enabled neighbours.
		excl := make([]regfile.Register, len(en))      // Exclusion slice.
		for i1, e1 := range en {
			excl[i1] = e1.Val.GetHW().(*lir.LiveNode).Reg.(regfile.Register)
		}

		typ := n.(*lir.LiveNode).Val.DataType()
		if typ == types.Int || typ == types.String {
			// Strings are addresses stored in register.
			r = rf.GetNextTempIExclude(excl)
		} else {
			r = rf.GetNextTempFExclude(excl)
		}

		// Check for registering spilling.
		if r == nil {
			// TODO: Implement register spilling.
			n.(*lir.LiveNode).Spill = true
			return errors.New("register spilling not implemented yet")
		} else {
			// Allocate physical register to virtual register.
			n.(*lir.LiveNode).Val.GetHW().(*lir.LiveNode).Reg = r
		}
	}

	// Assign registers for function's parameters.
	l := len(f.Params())
	if l > 0 {
		if l > 8 {
			l = 8
		}
		ii := 0
		fi := 0
		for _, e1 := range f.Params()[:l] {
			if e1.DataType() == types.Int {
				e1.GetHW().(*lir.LiveNode).Reg = rf.GetI(ii)
				ii++
			} else {
				e1.GetHW().(*lir.LiveNode).Reg = rf.GetF(fi)
				fi++
			}
		}
	}
	return nil
}
