//go:generate goyacc -o frontend/parser.yy.go frontend/parser-typed.y

package main

import (
	"fmt"
	"os"
	lir2 "vslc/src/backend/lir"
	"vslc/src/ir/lir"
	"vslc/src/ir/lir/types"
)

import (
	"vslc/src/backend"
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
		return fmt.Errorf("parse error: %s\n", err)
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
		return fmt.Errorf("assembler generation error: %s", err)
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

	//wg := sync.WaitGroup{}
	//
	//// Initiate output writer.
	//if !opt.LLVM {
	//	// Writing LLVM generated object code in parallel is outside the scope of this project.
	//	if len(opt.Out) > 0 {
	//		// Attempt to open output file. Create new file if necessary.
	//		if f, err := os.OpenFile(opt.Out, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0644); err == nil {
	//			defer func(f *os.File) {
	//				err := f.Close()
	//				if err != nil {
	//					fmt.Println(err)
	//				}
	//			}(f)
	//			util.ListenWrite(opt, f, &wg)
	//		} else {
	//			fmt.Println(err)
	//			os.Exit(1)
	//		}
	//	} else {
	//		// Write results to stdout.
	//		util.ListenWrite(opt, nil, &wg)
	//	}
	//	defer util.Close()
	//}
	//if err := run(opt); err != nil {
	//	fmt.Printf("Error: %s", err)
	//}
	//
	//// Wait for code generation to complete.
	//wg.Wait()               // TODO: Make this such that it works.
	//time.Sleep(time.Second) // TODO: Delete.

	// TODO: Below is test only.
	m := lir.CreateModule("")
	g := m.CreateGlobalInt("n")
	_ = m.CreateGlobalFloat("x")
	foo, err := m.CreateFunction(types.Int, "foo")
	if err != nil {
		fmt.Println(err)
		return
	}
	param := foo.CreateParamInt("")
	bb := foo.CreateBlock()
	y := bb.CreateLoad(g)
	a := bb.CreateConstantInt(1)
	b := bb.CreateConstantInt(5)
	_ = bb.CreateMul(a, b)
	d := bb.CreateLoad(param)
	e := bb.CreateSub(b, d)
	x := bb.CreateDiv(e, y)
	//_ = bb.CreateLoad(g)
	bb.CreateReturn(x)
	fmt.Println(m.String())

	if err := lir2.AllocateRegisters(opt, m); err != nil {
		fmt.Printf("register allocation failed: %s\n", err)
	}
}
