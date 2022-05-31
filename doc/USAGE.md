# Usage

## Prerequisites

1. llvm toolchain, preferable LLVM 13
2. llvm-devel toolchain, preferable LLVM 13
3. gcc compiler framework with C++ compiler support
4. go compiler and runtime (this software was developed on go 1.15.2)

## Installation

### Download the source files.

```bash
gi clone github.com/hhramberg/go-vslc
```

### Configure the C-bindings for Go

Follow the instructions on how to prepare the system for using the [C-bindings for Go](https://github.com/tinygo-org/go-llvm) 
by reading the Usage section.

#### I cannot configure the C-bindings

The update-script in the tinygo-library might point to a deprecated LLVM URL. If you're unable to configure the
C-bindings do the following.

1. Locate the path of your system's `GOPATH` by running the following command.
```bash
go env 
```
2. Change directory to the tinygo module, likely located in `%GOPATH%/pkg/mod/tinygo.org/x/go-llvm`.
3. Copy the ZIP-file `resourcs/tinygo_fix.zip` from the `go-vslc` project, unzip and replace the contents of the existing tinygo folder.
4. Now you have correctly configured C-bindings for LLVM 13 on linux-amd-64.

### Compile the vslc compiler

Move into the `src` folder of the `go-vslc` repository.
The below snippet will install the `vslc` compiler into the directory `/path/to/put/compiler/vslc`.

```bash
cd go-vslc/src
go build -o /path/to/put/compiler/vslc
```

## Usage

`vslc` is called similarly to GCC compilers. Flags and arguments precede the file to compile. Only a single VSL file
may be compiled per call. Mind you, the argument parser is simple, a filename must always be provided as the final 
argument, even when passing the --version or --help flags.

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
|-arch|Set output architecture type. Only one architecture is supported.|aarch64|aarch64|
|-ts|Output the tokens of the source code and exit.|||
|-v, -version, --v, --version|Prints application version and exits the application.|||
|-vb|Verbose mode. Include flag to log verbose compiler status messages to stdout, such as AST and SSA.|||
