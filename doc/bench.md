# Benchmarking

This file presents the benchmarking procedure and select results of the `go-vslc` compiler.

> Results for the hash comparison can be viewed in the Git repository located at: https://github.com/hhramberg/hashComparison

## Experimental setup

The below table list two test system's hardware specifications and operating system configuration.

|Callsign|Hardware type|CPU|RAM|OS|Description|
|---|---|---|---|---|---|
|laptop|Lenovo laptop|Intel Core i5-3320M|8 GB|Manjaro Linux 250-4-2 (kernel: 5.4.188-1-MANJARO)|Development system|
|server|HPE ProLiant DL360|2x Intel Xeon E5630|16 GB|VMWare ESXi 6.5|Hypervisor|

For the server callsign system I have created the following virtual machine which will be used
for testing.

|Callsign|Hardware type|CPU|RAM|OS|Description|
|---|---|---|---|---|---|
|vm|ESXi 6.5 virtual machine|16 vCPUs|2 GB|Centos 8 Stream||

The callsign `vm` has been allocated all the CPU resources of the hypervisor, with the remaining virtual machines
shutdown. Although hypervisors grants the flexibility of testing differently equipped computers with
little configuration it doesn't benefit the parallel benchmark test by testing a 3-core or 6-core
vm when you can utilise all 8 cores. The expected result would be that highly parallel source code
with many functions will run slower on fewer cores than on many cores.

The benchmark is not intended to specifically find a core count that maximizes efficiency,
but rather attempt to prove that function level parallelism may improve compilation time.

## Procedure

The same benchmarks will be executed on both systems described above, the `laptop` and `vm` callsign systems.
Timing will be measured using the [Go testing package](https://pkg.go.dev/testing) which is a reliable and idiomatic
way of benchmarking Go language software. Additionally it allows benchmarking of the compilers individual parts.
The benchmarks are located in the following file, relative to the project's root.

```bash
/src/vslc_test.go
```

The benchmarks define the parallelism of the benchmark by adjusting the util.Options.Threads parameter in their internal
for-loops. A benchmark tests every VSL source file, with thread count ranging from 1 to p, where p is a constants defined
in the benchmark file (defaults to 16). Reading source files into memory is not benchmarked.

There are 5 defined benchmarks.

|Name|Description|
|---|---|
|BenchmarkAarch64|Benchmarks the entire compiler from scanning to assembler generation. Targets aarch64 assembler.|
|BenchmarkASTOptimisation|Benchmarks only the process of optimising the syntax tree after parsing.|
|BenchmarkLIRGeneration|Benchmarks only the process of turning the syntax tree into LIR SSA.|
|BenchmarkRegisterAllocation|Benchmarks only the process of allocating aarch64 hardware registers to LIR SSA virtual registers.|
|BenchmarkAssemblerGeneration|Benchmarks only the process of turning LIR SSA into aarch64 assembler, including writing to file.|

## Results

> Results for the hash comparison can be viewed in the Git repository located at: https://github.com/hhramberg/hashComparison


Results for the parallel compiler benchmark is located in [bench.csv](bench.csv).

Older results, from a more excessive benchmark suite with almost 100% source file coverage
can be found at [bench_old_all_missing_some.ods](bench_old_all_missing_some.ods) (LibreOffice Calc file). Do note
that the source file coverage is incomplete.