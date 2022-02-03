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
	Src         string // Path to source file.
	Out         string // Path to output file.
	Threads     int    // Thread count.
	Verbose     bool   // Set true if compiler should log statistical data to stdout.
	TokenStream bool   // Set true if compiler should output token stream and exit.
	Target      int    // Output target architecture.
}

// ---------------------
// ----- Constants -----
// ---------------------

const maxThreads = 64 // Maximum threads allowed executing in parallel.
const appVersion = "vsl compiler 1.0"

const (
	Aarch64 = iota
	Riscv64
	Riscv32
)

// ---------------------
// ----- Functions -----
// ---------------------

// ParseArgs parses command line arguments.
func ParseArgs() (Options, error) {
	opt := Options{}
	if len(os.Args) < 2 {
		return opt, nil
	}
	args := os.Args[1:]
	for i1 := 0; i1 < len(args); i1++ {
		switch args[i1] {
		case "-h", "--h", "-help", "--help":
			// Help and usage.
			printHelp()
			os.Exit(0)
		case "-o", "-s", "-t":
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
			case "-s":
				// Source file.
				opt.Src = args[i1+1]
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
		case "-target":
			// Output architecture.
			if i1+1 >= len(args) {
				return opt, fmt.Errorf("got flag %s but no argument", args[i1])
			}
			if strings.HasPrefix(args[i1+1], "-") {
				return opt, fmt.Errorf("expected architecture identifier, got new flag %s", args[i1+1])
			}
			switch args[i1+1] {
			case "aarch64":
				opt.Target = Aarch64
			case "riscv64":
				opt.Target = Riscv64
			case "riscv32":
				opt.Target = Riscv32
			default:
				return opt, fmt.Errorf("unexpected architecture identifier: %s", args[i1+1])
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
		default:
			return opt, fmt.Errorf("unexpected flag: %s", args[i1])
		}
	}
	return opt, nil
}

// printHelp prints a helpful usage message to stdout.
func printHelp() {
	w := tabwriter.NewWriter(os.Stdout, 6, 1, 1, 0, 0)
	_, _ = fmt.Fprintln(w, "-h, -help\tPrints this help message and exits the application.")
	_, _ = fmt.Fprintln(w, "--h, --help")
	_, _ = fmt.Fprintln(w, "-o\tOutput file.")
	_, _ = fmt.Fprintln(w, "-s\tPath to source VSL file.")
	_, _ = fmt.Fprintln(w, "-t\tNumber of threads to run in parallel. Must be in range [1, %d].", maxThreads)
	_, _ = fmt.Fprintln(w, "-target\tOutput architecture type. Can be either 'Aarch64' or 'Riscv64'. Defaults to 'Aarch64'.")
	_, _ = fmt.Fprintln(w, "-ts\tOutput the tokens of the source code and exit.")
	_, _ = fmt.Fprintln(w, "-v, -version\tPrints application version and exits the application.")
	_, _ = fmt.Fprintln(w, "--v, --version")
	_, _ = fmt.Fprintln(w, "-vb\tVerbose mode: print compiler statistics to stdout.")
	_ = w.Flush()
}
