//go:generate goyacc -o frontend/parser.yy.go frontend/parser-typed.y

package main

import (
	"fmt"
	"os"
	"vslc/src/backend"
	"vslc/src/frontend"
	"vslc/src/ir"
	"vslc/src/ir/llvm"
	"vslc/src/util"
)

func main() {
	// Parse command line arguments.
	opt, err := util.ParseArgs()
	if err != nil {
		fmt.Printf("Command line argument error: %s\n", err)
		os.Exit(1)
	}

	// Read source code.
	src, err := util.ReadSource(opt)
	if err != nil {
		fmt.Printf("Could not read source code: %s\n", err)
		os.Exit(1)
	}

	// If -ts flag was passed: output token stream and exit.
	if opt.TokenStream {
		if err := frontend.TokenStream(src); err != nil {
			fmt.Printf("Syntax error: %s\n", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	// Generate syntax tree by lexing and parsing source code.
	if err := frontend.Parse(src); err != nil {
		fmt.Printf("Parse error: %s\n", err)
		os.Exit(1)
	}

	// Optimise syntax tree.
	if err := ir.Optimise(opt); err != nil {
		if opt.Threads > 1 {
			// Print errors during parallel optimisation before exiting.
			for _, e1 := range ir.Errors() {
				fmt.Println(e1)
			}
		}
		fmt.Printf("Syntax tree error: %s\n", err)
		os.Exit(1)
	}

	if opt.LLVM {
		defer func(){
			if r := recover(); r != nil {
				fmt.Println(r) // TODO: delete.
			}
		}()
		if err := llvm.GenLLVM(opt, ir.Root); err != nil {
			fmt.Printf("Error reported by LLVM: %s", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	// Generate symbol table using my implementation.
	if err := ir.GenerateSymTab(opt); err != nil {
		fmt.Printf("Source code error: %s\n", err)
		os.Exit(1)
	}

	// Validate source code.
	if err := ir.ValidateTree(opt); err != nil {
		fmt.Printf("Source code error: %s", err)
		os.Exit(1)
	}

	// Initiate output writer.
	if len(opt.Out) > 0 {
		// Attempt to open output file. Create new file if necessary.
		if f, err := os.OpenFile(opt.Out, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0644); err != nil {
			defer func(f *os.File) {
				err := f.Close()
				if err != nil {
					fmt.Println(err)
					os.Exit(1)
				}
			}(f)
			util.ListenWrite(opt.Threads, f)
		} else {
			fmt.Println(err)
			os.Exit(1)
		}
	} else {
		// Write results to stdout.
		util.ListenWrite(opt.Threads, nil)
	}

	// Generate assembler.
	if err := backend.GenerateAssembler(opt); err != nil {
		fmt.Printf("Code generation error: %s\n", err)
		util.Close()
		os.Exit(1)
	}

	// Stop the output writer.
	util.Close()
}
