# Scripts

The repository contains three bundled scripts.
- [test.sh](test.sh)
- [test_run.sh](test_run.sh)
- [bench.sh](bench.sh)

The former two verifies that the compiler successfully compiles VSL source code into executable assembler.
The latter benchmarks the compilation process into assembler with respect to parallel compiler execution.

[test_run.sh](test_run.sh) requires that QEMU and GCC variants for `aarch64` and `riscv64` be installed.
All test scripts must be run with the repository root as working directory.