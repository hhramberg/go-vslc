# Usage

## Installation

## Usage

## Flags

Below is a table of compiler flags, descriptions and possibly default vaues and 
accepted ranges.

|Flag|Description|Range|Default value|
|---|---|---|---|
|-h, -help, --h, --help|Prints help message and exits the application.|||
|-o|Path to and file name of output assembler file. If no output path is provided the compiler will write the resulting assembler to `stdout`.|||
|-s|Path to source VSL file. If no source path is provided the compiler attempts to read VSL source code from `stdin`.|||
|-t|Number of threads to run in parallel.|[1, 64]|1|
|-target|Set output architecture type.|{aarch64, riscv}|aarch64|
|-ts|Output the tokens of the source code and exit.|||
|-v, -version, --v, --version|Prints application version and exits the application.|||
|-vb|Verbose mode. Include flag to log verbose compiler status messages to stdout.|||
