//go:generate goyacc -o frontend/parser.yy.go frontend/parser-typed.y

package main

import (
	"fmt"
	"os"
	"vslc/src/backend"
	lir2 "vslc/src/backend/lir"
	"vslc/src/ir/lir"
)

import (
	"vslc/src/frontend"
	"vslc/src/ir"
	"vslc/src/ir/llvm"
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
		return err
	}

	// Optimise syntax tree.
	if err := ir.Optimise(opt); err != nil {
		return fmt.Errorf("syntax tree error: %s\n", err)
	}

	if opt.Verbose {
		fmt.Println("Syntax tree:")
		ir.Root.Print(0, true)
	}

	// Gen LLVM and exit, if flag is passed.
	if opt.LLVM {
		if err = llvm.GenLLVM(opt, ir.Root); err != nil {
			return fmt.Errorf("error reported by LLVM: %s", err)
		}
		return nil
	}

	// Generate SSA from optimised and validated parse tree.
	m, err := lir.GenLIR(opt, ir.Root)
	if err != nil {
		return err
	}

	if opt.Verbose {
		fmt.Println("\nLIR intermediate representation:")
		fmt.Println(m.String())
	}

	// Allocate hardware registers to LIR virtual registers.
	if err := lir2.AllocateRegisters(opt, m); err != nil {
		return err
	}

	// Generate assembler.
	if err := backend.GenerateAssembler(opt, m, ir.Root); err != nil {
		return err
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
	if opt.LLVM && opt.TokenStream {
		fmt.Println("Error: cannot run token stream and LLVM generation at the same time.")
		os.Exit(1)
	}
	if !opt.LLVM {
		// Writing LLVM generated object code in parallel is outside the scope of this project.
		if len(opt.Out) > 0 {
			// Attempt to open output file. Create new file if necessary.
			if f, err := os.OpenFile(opt.Out, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0644); err == nil {
				defer func(f *os.File) {
					err := f.Close()
					if err != nil {
						fmt.Println(err)
					}
				}(f)
				util.ListenWrite(opt, f)
			} else {
				fmt.Println(err)
				os.Exit(1)
			}
		} else {
			// Write results to stdout.
			util.ListenWrite(opt, nil)
		}
	}

	ret := 0
	if err := run(opt); err != nil {
		fmt.Printf("Error: %s\n", err)
		ret = 1
	}

	if !opt.LLVM {
		util.Close()
	}

	// Wait for code generation to complete.
	os.Exit(ret)
}
