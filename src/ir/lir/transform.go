package lir

import (
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"vslc/src/ir"
	"vslc/src/util"
)

// ----------------------------
// ----- Type definitions -----
// ----------------------------

type funcWrapper struct {
	node  *ir.Node
	entry *Function
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

// GenLIR generates lightweight intermediate representation from the syntax tree.
func GenLIR(opt util.Options, root *ir.Node, globals *ir.SymTab) (*Module, error) {
	m := CreateModule(filepath.Base(opt.Src)) // The LIR module.
	if opt.Threads > 1 {
		// Parallel.
		t := opt.Threads
		l := len(root.Children)
		if t > l {
			t = l
		}
		n := l / t
		res := l % t

		start := 0
		end := n

		wg := sync.WaitGroup{}
		wg.Add(t)

		perr := util.NewPerror(t)

		// funcs hold LIR function wrappers.
		funcs := make([]funcWrapper, 0, t)

		// Spawn t worker go routines.
		for i1 := 0; i1 < t; i1++ {
			if i1 < res {
				// This worker go routine should perform one residual job.
				end++
			}

			// Spawn go routine.
			go func(start, end int, wg *sync.WaitGroup) {
				for _, e1 := range root.Children[start:end] {
					if err := genGlobals(e1); err != nil {
						perr.Append(err)
						continue
					}
					// TODO: Continue here.
				}
			}(start, end, &wg)

			start = end
			end += n
		}

		// Check for errors.
		if perr.Len() > 0 {
			for e1 := range perr.Errors() {
				fmt.Println(e1)
			}
			return nil, fmt.Errorf("%d errors during parallel LIR generation", perr.Len())
		}
		perr.Flush()

		// Generate LIR function bodies.
		t = opt.Threads
		l = len(funcs)
		if t > l {
			t = l
		}
		n = l / t
		res = l % t

		start = 0
		end = n

		// Spawn t worker go routines.
		for i1 := 0; i1 < t; i1++ {

		}

		// Check for errors.
		if perr.Len() > 0 {
			for e1 := range perr.Errors() {
				fmt.Println(e1)
			}
			return nil, fmt.Errorf("%d errors during parallel LIR generation", perr.Len())
		}

		// Generate function bodies.

		perr.Stop()
	} else {
		// Sequential.
	}
	return m, errors.New("lightweight intermediate representation is not implemented yet")
}

func genGlobals(n *ir.Node) error {
	return nil
}

func genFunctionHeader(n *ir.Node, m *Module) (*Function, error) {
	return nil, errors.New("LIR function header generation not implemented yet")
}

func genFunctionBody(n *ir.Node, f *Function) error {
	return errors.New("LIR function body generation not implemented yet")
}
