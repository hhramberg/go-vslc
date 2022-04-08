package util

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"
)

// ----------------------------
// ----- Type definitions -----
// ----------------------------

type Options struct {
	Src          string // Path to source file.
	Out          string // Path to output file.
	Threads      int    // Thread count.
	Verbose      bool   // Set true if compiler should log statistical data to stdout.
	TokenStream  bool   // Set true if compiler should output token stream and exit.
	LLVM         bool   // Set true if compiler should use the LLVM framework to issue optimisations and code generaton.
	TargetArch   int    // Output target architecture.
	TargetVendor int    // Output target vendor type. 0 = unknown.
	TargetCPU    int    // Output target CPU. 0 = generic CPU.
	TargetOS     int    // Output target operating system type.
}

// ---------------------
// ----- Constants -----
// ---------------------

const maxThreads = 64 // Maximum threads allowed executing in parallel.
const appVersion = "vsl compiler 1.0"

// Target machine architectures.
const (
	UnknownArch = iota
	X86_64
	X86_32
	Aarch64
	Riscv64
	Riscv32
)

// Target operating system.
const (
	UnknownOS = iota
	Linux
	Windows
	MAC
)

// Target vendor.
const (
	UnknownVendor = iota
	Apple
	PC
	MIPS
	IBM
	SUSE
	AMD
)

// Target CPU.
const (
	CPUGeneric = iota
)

// ---------------------
// ----- functions -----
// ---------------------

// ParseArgs parses command line arguments.
func ParseArgs() (Options, error) {
	opt := Options{}
	if len(os.Args) < 2 {
		return opt, nil
	}
	args := os.Args[1:]
	for i1 := 0; i1 < len(args)-1; i1++ {
		switch args[i1] {
		case "-h", "--h", "-help", "--help":
			// Help and usage.
			printHelp()
			os.Exit(0)
		case "-ll":
			// Use LLVM IR and LLVM code generator.
			opt.LLVM = true
		case "-o", "-t":
			if i1+1 >= len(args) {
				return opt, fmt.Errorf("got flag %s but no argument", args[i1])
			}
			if strings.HasPrefix(args[i1+1], "-") {
				return opt, fmt.Errorf("expected path to source file, got new flag %s", args[i1+1])
			}
			switch args[i1] {
			case "-o":
				// Output file.
				opt.Out = args[i1+1]
			case "-t":
				// Thread count.
				if t, err := strconv.Atoi(args[i1+1]); err == nil {
					if t > 0 && t <= maxThreads {
						opt.Threads = t
					} else {
						return opt, fmt.Errorf("thread count must be integer in range [1, %d]", maxThreads)
					}
				} else {
					return opt, fmt.Errorf("expected integer thread count, got: %s", args[i1+1])
				}
			}
			i1++
		case "-arch":
			// Output architecture.
			if i1+1 >= len(args) {
				return opt, fmt.Errorf("got flag %s but no argument", args[i1])
			}
			if strings.HasPrefix(args[i1+1], "-") {
				return opt, fmt.Errorf("expected architecture identifier, got new flag %s", args[i1+1])
			}
			switch args[i1+1] {
			case "aarch64":
				opt.TargetArch = Aarch64
			case "riscv64":
				opt.TargetArch = Riscv64
			case "riscv32":
				opt.TargetArch = Riscv32
			case "x86_64":
				opt.TargetArch = X86_64
			case "x86_32":
				opt.TargetArch = X86_32
			default:
				return opt, fmt.Errorf("unexpected architecture identifier: %s", args[i1+1])
			}
			i1++
		case "-os":
			// Output operating system type.
			if i1+1 >= len(args) {
				return opt, fmt.Errorf("got flag %s but no argument", args[i1])
			}
			if strings.HasPrefix(args[i1+1], "-") {
				return opt, fmt.Errorf("expected operating system identifier, got new flag %s", args[i1+1])
			}
			switch args[i1+1] {
			case "linux":
				opt.TargetOS = Linux
			case "windows":
				opt.TargetOS = Windows
			case "mac":
				opt.TargetOS = MAC
			default:
				return opt, fmt.Errorf("unexpected operating system identifier: %s", args[i1+1])
			}
			i1++
		case "-vendor":
			// Output vendor type.
			if i1+1 >= len(args) {
				return opt, fmt.Errorf("got flag %s but no argument", args[i1])
			}
			if strings.HasPrefix(args[i1+1], "-") {
				return opt, fmt.Errorf("expected vendor identifier, got new flag %s", args[i1+1])
			}
			switch args[i1+1] {
			case "pc":
				opt.TargetVendor = PC
			case "apple":
				opt.TargetOS = Apple
			case "ibm":
				opt.TargetVendor = IBM
			default:
				return opt, fmt.Errorf("unexpected vendor identifier: %s", args[i1+1])
			}
			i1++
		case "-ts":
			// Output token stream
			opt.TokenStream = true
		case "-v", "--v", "-version", "--version":
			// Application version.
			fmt.Println(appVersion)
			os.Exit(0)
		case "-vb":
			// Verbose mode.
			opt.Verbose = true
		default:
			return opt, fmt.Errorf("unexpected flag: %s", args[i1])
		}
	}
	if len(args) > 0 {
		opt.Src = args[len(args)-1]
	}
	return opt, nil
}

// printHelp prints a helpful usage message to stdout.
func printHelp() {
	w := tabwriter.NewWriter(os.Stdout, 6, 1, 1, 0, 0)
	_, _ = fmt.Fprintln(w, "-h, -help\tPrints this help message and exits the application.")
	_, _ = fmt.Fprintln(w, "--h, --help")
	_, _ = fmt.Fprintln(w, "-ll\tUse LLVM to optimise and generate output code.")
	_, _ = fmt.Fprintln(w, "-o\tPath and name of the output file.")
	_, _ = fmt.Fprintf(w, "-t\tNumber of threads to run in parallel. Must be in range [1, %d].\n", maxThreads)
	_, _ = fmt.Fprintln(w, "-target\tOutput architecture type. Can be either 'Aarch64', 'Riscv32' or 'Riscv64'. Defaults to 'Aarch64'.")
	_, _ = fmt.Fprintln(w, "-ts\tOutput the tokens of the source code and exit.")
	_, _ = fmt.Fprintln(w, "-v, -version\tPrints application version and exits the application.")
	_, _ = fmt.Fprintln(w, "--v, --version")
	_, _ = fmt.Fprintln(w, "-vb\tVerbose mode: print compiler statistics to stdout.")
	_ = w.Flush()
}
