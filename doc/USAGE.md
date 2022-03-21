# Usage

## Installation

## Usage

`vslc` is called similarly to GCC compilers. Flags and arguments precede the file to compile. Only a single VSL file
may be compiled per call.

```bash
vslc [FLAG [ARGUMENT] ...] file
```

See the section [Flags](#flags) for flags and flag arguments. 

## Flags

Below is a table of compiler flags, descriptions and possibly default vaues and 
accepted ranges.

|Flag|Description|Range|Default value|
|---|---|---|---|
|-h, -help, --h, --help|Prints help message and exits the application.|||
|-o|Path to and file name of output file. If no output path is provided the compiler will write the resulting assembler to `stdout` or `app.out` for binaries.| |`stdout` or `app.out`|
|-ll|Use the LLVM backend to optimise and generate code.|||
|-t|Number of threads to run in parallel.|[1, 64]|1|
|-target|Set output architecture type.|{aarch64, riscv}|aarch64|
|-ts|Output the tokens of the source code and exit.|||
|-v, -version, --v, --version|Prints application version and exits the application.|||
|-vb|Verbose mode. Include flag to log verbose compiler status messages to stdout.|||
