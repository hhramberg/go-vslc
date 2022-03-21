//go:generate goyacc -o frontend/parser.yy.go frontend/parser-typed.y

package main

import (
	"fmt"
	"os"
	"sync"
	"time"
	"vslc/src/backend"
	"vslc/src/frontend"
	"vslc/src/ir"
	ll2 "vslc/src/ir/llvm"
	"vslc/src/util"
)

// run begins reading source code and executes compiler stages.
// Behaviour is defined by the util.Options structure.
func run(opt util.Options) error {
	// Read source code.
	src, err := util.ReadSource(opt)
	if err != nil {
		return fmt.Errorf("could not read source code: %s\n", err)
	}

	// If -ts flag was passed: output token stream and exit.
	if opt.TokenStream {
		if err := frontend.TokenStream(src); err != nil {
			return fmt.Errorf("syntax error: %s\n", err)
		}
		return nil
	}

	// Generate syntax tree by lexing and parsing source code.
	if err := frontend.Parse(src); err != nil {
		return fmt.Errorf("parse error: %s\n", err)
	}

	// Optimise syntax tree.
	if err := ir.Optimise(opt); err != nil {
		if opt.Threads > 1 {
			// Print errors during parallel optimisation before exiting.
			for _, e1 := range ir.Errors() {
				fmt.Println(e1)
			}
		}
		return fmt.Errorf("syntax tree error: %s\n", err)
	}

	// TODO: Delete.
	if opt.Verbose {
		ir.Root.Print(0, true)
	}

	// Gen LLVM and exit, if flag is passed.
	if opt.LLVM {
		//if err = llvm.GenLLVM(opt, ir.Root); err != nil {
		//	return fmt.Errorf("error reported by LLVM: %s", err)
		//}
		if err = ll2.GenLLVM(opt, ir.Root); err != nil {
			return fmt.Errorf("error reported by LLVM: %s", err)
		}
		return nil
	}

	// Generate output assembler manually.

	// Generate symbol table using my implementation.
	if err = ir.GenerateSymTab(opt); err != nil {
		return err
	}

	// Validate source code.
	if err = ir.ValidateTree(opt); err != nil {
		return err
	}

	// Generate assembler.
	if err = backend.GenerateAssembler(opt); err != nil {
		return fmt.Errorf("code generation error: %s\n", err)
	}
	return nil
}

func main() {
	// Parse command line arguments.
	opt, err := util.ParseArgs()
	if err != nil {
		fmt.Printf("Command line argument error: %s\n", err)
		os.Exit(1)
	}

	// Initiate output writer.
	wg := sync.WaitGroup{}
	if len(opt.Out) > 0 {
		// Attempt to open output file. Create new file if necessary.
		if f, err := os.OpenFile(opt.Out, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0644); err != nil {
			defer func(f *os.File) {
				err := f.Close()
				if err != nil {
					fmt.Println(err)
				}
			}(f)
			util.ListenWrite(opt, f, &wg)
		} else {
			fmt.Println(err)
			os.Exit(1)
		}
	} else {
		// Write results to stdout.
		util.ListenWrite(opt, nil, &wg)
	}
	defer util.Close()

	if err := run(opt); err != nil {
		fmt.Printf("Error: %s", err)
	}

	// Wait for code generation to complete.
	wg.Wait()               // TODO: Make this such that it works.
	time.Sleep(time.Second) // TODO: Delete.
}
