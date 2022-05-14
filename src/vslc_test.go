package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"vslc/src/backend"
	lir2 "vslc/src/backend/lir"
	"vslc/src/frontend"
	"vslc/src/ir"
	"vslc/src/ir/lir"
	"vslc/src/ir/llvm"
	"vslc/src/util"
)

// -----------------------------
// ----- Type definitions ------
// -----------------------------

// benchType defines a benchmark with pre-defined benchmark parameters.
type benchType struct {
	name string // Informative name of benchmark.
	src  string // The VSL source file as a string.
	out  string // Destination assembler file of benchmark.
}

// ----------------------
// ----- Constants ------
// ----------------------

// p defines the maximum number of parallel threads to pass to the compiler.
const p = 4

// --------------------
// ----- Globals ------
// --------------------

// srcPath defines the relative path from working directory of vslc/src project to the typed VSL source files.
var srcPath = "/resources/vsl_typed/"

// dstPath defines the relative path from working directory of vslc/src project to the build folder.
var dstPath = "/build/"

// ----------------------
// ----- Functions ------
// ----------------------

// BenchmarkAarch64 benchmarks compiling all bundled project typed VSL source files into assembler.
func BenchmarkAarch64(b *testing.B) {
	wd, err := os.Getwd()
	if err != nil {
		b.Fatal(err)
	}
	srcp := filepath.Join(wd, "../", srcPath)
	dstp := filepath.Join(wd, "../", dstPath)

	src, files := helperReadFiles(srcp, b)

	// Compiler configuration:
	// Target architecture: aarch64
	opt := util.Options{
		Threads:    1,
		TargetArch: util.Aarch64,
	}

	benchmarks := make([]benchType, len(files))
	for i1, e1 := range files {
		benchmarks[i1] = benchType{
			name: e1.Name(),
			src:  src[i1],
			out:  filepath.Join(dstp, strings.Split(files[i1].Name(), ".")[0]+".s"),
		}
	}

	// Run benchmarks for all VSL source files.
	for _, e1 := range benchmarks {
		opt.Out = e1.out

		// Test for 1 to p parallel worker go routines.
		for i2 := 1; i2 <= p; i2++ {
			opt.Threads = i2

			// Attempt to open output file. Create new file if necessary.
			f, err := os.OpenFile(opt.Out, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				b.Fatalf("I/O error, could not open/create destination file: %s\n", err)
			}
			b.Run(fmt.Sprintf("%s-threads=%d", e1.name, i2), func(b *testing.B) {
				for n := 0; n < b.N; n++ {
					util.ListenWriteBench(opt)
					if err := benchRun(e1.src, opt); err != nil {
						b.Fatalf("Compiler error: %s\n", err)
					}
					util.Close()
				}
			})

			// Close the destination file.
			err = f.Close()
			if err != nil {
				b.Fatalf("I/O error, could not close destination file: %s\n", err)
			}
		}
	}

	b.Cleanup(func() {
		helperDeleteFiles(dstp, files, b)
	})
}

// BenchmarkASTOptimisation benchmarks the frontend.Optimise function that optimises the parse tree.
// The scanning process cannot be decoupled from the benchmark because the parse tree has to be regenerated for every
// optimisation pass.
func BenchmarkASTOptimisation(b *testing.B) {
	wd, err := os.Getwd()
	if err != nil {
		b.Fatal(err)
	}
	srcp := filepath.Join(wd, "../", srcPath)
	src, files := helperReadFiles(srcp, b)

	// Compiler configuration:
	// None needed, parse tree optimisation is target independent.
	// The benchmark relies solely on maximum thread count.
	opt := util.Options{
		Threads: 1, // Re-configured in benchmark inner loop.
	}

	benchmarks := make([]benchType, len(files))
	for i1, e1 := range files {
		benchmarks[i1] = benchType{
			name: e1.Name(),
			src:  src[i1],
			out:  filepath.Join(dstPath, strings.Split(files[i1].Name(), ".")[0]+".s"),
		}
	}

	// Run benchmarks for all VSL source files.
	for _, e1 := range benchmarks {
		opt.Out = e1.out

		// Test for 1 to p parallel worker go routines.
		for i2 := 1; i2 <= p; i2++ {
			opt.Threads = i2
			b.Run(fmt.Sprintf("%s-threads=%d", e1.name, i2), func(b *testing.B) {
				for n := 0; n < b.N; n++ {
					if err := frontend.Parse(e1.src); err != nil {
						b.Fatalf("Could not parse syntax tree: %s\n", err)
					}
					if err := ir.Optimise(opt); err != nil {
						b.Fatalf("Could not optimise syntax tree: %s\n", err)
					}
				}
			})
		}
	}
}

// BenchmarkLIRGeneration measures the performance of transforming the optimised syntax tree into LIR SSA.
func BenchmarkLIRGeneration(b *testing.B) {
	wd, err := os.Getwd()
	if err != nil {
		b.Fatal(err)
	}
	srcp := filepath.Join(wd, "../", srcPath)
	src, files := helperReadFiles(srcp, b)

	// Compiler configuration:
	// None needed, parse tree optimisation is target independent.
	// The benchmark relies solely on maximum thread count.
	opt := util.Options{
		Threads: 1, // Re-configured in benchmark inner loop.
	}

	benchmarks := make([]benchType, len(files))
	for i1, e1 := range files {
		benchmarks[i1] = benchType{
			name: e1.Name(),
			src:  src[i1],
			out:  filepath.Join(dstPath, strings.Split(files[i1].Name(), ".")[0]+".s"),
		}
	}

	// Run benchmarks for all VSL source files.
	for _, e1 := range benchmarks {
		opt.Out = e1.out

		// Test for 1 to p parallel worker go routines.
		for i2 := 1; i2 <= p; i2++ {
			opt.Threads = i2
			if err := frontend.Parse(e1.src); err != nil {
				b.Fatalf("Could not parse syntax tree: %s\n", err)
			}
			if err := ir.Optimise(opt); err != nil {
				b.Fatalf("Could not optimise syntax tree: %s\n", err)
			}
			b.Run(fmt.Sprintf("%s-threads=%d", e1.name, i2), func(b *testing.B) {
				for n := 0; n < b.N; n++ {
					if _, err := lir.GenLIR(opt, ir.Root); err != nil {
						b.Fatalf("Could not generate LIR: %s\n", err)
					}
				}
			})
		}
	}
}

// BenchmarkRegisterAllocation measures the performance of allocating hardware registers to the LIR SSA virtual
// registers. The target architecture is aarch64.
func BenchmarkRegisterAllocation(b *testing.B) {
	wd, err := os.Getwd()
	if err != nil {
		b.Fatal(err)
	}
	srcp := filepath.Join(wd, "../", srcPath)
	src, files := helperReadFiles(srcp, b)

	// Compiler configuration:
	// None needed, parse tree optimisation is target independent.
	// The benchmark relies solely on maximum thread count.
	opt := util.Options{
		Threads:    1, // Re-configured in benchmark inner loop.
		TargetArch: util.Aarch64,
	}

	benchmarks := make([]benchType, len(files))
	for i1, e1 := range files {
		benchmarks[i1] = benchType{
			name: e1.Name(),
			src:  src[i1],
			out:  filepath.Join(dstPath, strings.Split(files[i1].Name(), ".")[0]+".s"),
		}
	}

	// Run benchmarks for all VSL source files.
	for _, e1 := range benchmarks {
		opt.Out = e1.out

		// Test for 1 to p parallel worker go routines.
		for i2 := 1; i2 <= p; i2++ {
			opt.Threads = i2
			if err := frontend.Parse(e1.src); err != nil {
				b.Fatalf("Could not parse syntax tree: %s\n", err)
			}
			if err := ir.Optimise(opt); err != nil {
				b.Fatalf("Could not optimise syntax tree: %s\n", err)
			}
			m, err := lir.GenLIR(opt, ir.Root)
			if err != nil {
				b.Fatalf("Could not generate LIR: %s\n", err)
			}
			b.Run(fmt.Sprintf("%s-threads=%d", e1.name, i2), func(b *testing.B) {
				for n := 0; n < b.N; n++ {
					if err := lir2.AllocateRegisters(opt, m); err != nil {
						b.Fatalf("Could not allocate registers for target architecture %d: %s\n", opt.TargetArch, err)
					}
				}
			})
		}
	}
}

// BenchmarkAssemblerGeneration benchmarks transforming LIR SSA into assembler. The target architecture is aarch64.
func BenchmarkAssemblerGeneration(b *testing.B) {
	wd, err := os.Getwd()
	if err != nil {
		b.Fatal(err)
	}
	srcp := filepath.Join(wd, "../", srcPath)
	dstp := filepath.Join(wd, "../", dstPath)

	src, files := helperReadFiles(srcp, b)

	// Compiler configuration:
	// Target architecture: aarch64
	opt := util.Options{
		Threads:    1,
		TargetArch: util.Aarch64,
	}

	benchmarks := make([]benchType, len(files))
	for i1, e1 := range files {
		benchmarks[i1] = benchType{
			name: e1.Name(),
			src:  src[i1],
			out:  filepath.Join(dstp, strings.Split(files[i1].Name(), ".")[0]+".s"),
		}
	}

	// Run benchmarks for all VSL source files.
	for _, e1 := range benchmarks {
		opt.Out = e1.out

		// Test for 1 to p parallel worker go routines.
		for i2 := 1; i2 <= p; i2++ {
			opt.Threads = i2

			// Attempt to open output file. Create new file if necessary.
			f, err := os.OpenFile(opt.Out, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				b.Fatalf("I/O error, could not open/create destination file: %s\n", err)
			}
			if err := frontend.Parse(e1.src); err != nil {
				b.Fatalf("Could not parse syntax tree: %s\n", err)
			}
			if err := ir.Optimise(opt); err != nil {
				b.Fatalf("Could not optimise syntax tree: %s\n", err)
			}
			m, err := lir.GenLIR(opt, ir.Root)
			if err != nil {
				b.Fatalf("Could not generate LIR: %s\n", err)
			}
			if err := lir2.AllocateRegisters(opt, m); err != nil {
				b.Fatalf("Could not allocate registers for target architecture %d: %s\n", opt.TargetArch, err)
			}
			b.Run(fmt.Sprintf("%s-threads=%d", e1.name, i2), func(b *testing.B) {
				for n := 0; n < b.N; n++ {
					util.ListenWriteBench(opt)
					if err := backend.GenerateAssembler(opt, m, ir.Root); err != nil {
						b.Fatalf("Could not generate assembler: %s\n", err)
					}
					util.Close()
				}
			})

			// Close the destination file.
			err = f.Close()
			if err != nil {
				b.Fatalf("I/O error, could not close destination file: %s\n", err)
			}
		}
	}
	b.Cleanup(func() {
		helperDeleteFiles(dstp, files, b)
	})
}

// benchRun runs the compiler, exactly like the run function, but without reading the source code.
func benchRun(src string, opt util.Options) error {
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
		if err := llvm.GenLLVM(opt, ir.Root); err != nil {
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

// helperReadFiles reads all typed VSL source files into memory. This helper function is ignored by the test metric
// counters; time spent executing this function isn't included in the benchmark results.
func helperReadFiles(srcp string, b *testing.B) ([]string, []os.FileInfo) {
	b.Helper()
	files, err := ioutil.ReadDir(srcp)
	if err != nil {
		b.Fatalf("Could not read VSL source files: %s", err)
	}
	src := make([]string, len(files))
	for i1, e1 := range files {
		data, err := ioutil.ReadFile(filepath.Join(srcp, e1.Name()))
		if err != nil {
			b.Fatal(err)
		}
		src[i1] = string(data)
	}
	return src, files
}

// helperDeleteFiles deletes the files in dstPath directory pointed to by the []os.FileInfo files.
func helperDeleteFiles(dstp string, files []os.FileInfo, b *testing.B) {
	b.Helper()
	for _, e1 := range files {
		if err := os.Remove(filepath.Join(dstp, strings.Split(e1.Name(), ".")[0]+".s")); err != nil {
			fmt.Println(err)
			b.Fail()
		}
	}
}
