# Benchmarks for Prysm State Transition
This package contains the functionality needed for benchmarking Prysms state transitions, this includes its block processing (with and without caching) and epoch processing functions. There is also a benchmark for HashTreeRoot on a large beacon state.

## Benchmark Configuration
The following configs are in `config.go`:
* `ValidatorCount`: Sets the amount of active validators to perform the benchmarks with. Default is 65536.
* `AttestationsPerEpoch`: Sets the amount of attestations per epoch for the benchmarks to perform with, this affects the amount of attestations in a full block and the amount of attestations per epoch in the state for the `ProcessEpoch` and `HashTreeRoot` benchmark. Default is 128.

## Generating new SSZ files
Due to the sheer size of the benchmarking configurations (65536 validators), the files used for benchmarking are pregenerated so there's no wasted computations on generating a genesis state with 65536 validators. This should only be needed if there is a breaking spec change and the tests fail from SSZ issues.

To generate new files to use for benchmarking, run the below command in the root of Prysm.
```
go run beacon-chain/core/state/benchmarks/benchmark_files/generate_bench_files.go
```

Bazel does not allow writing to the project directory, so running with `go run` is needed.

## Current Results as of November 2019
```
BenchmarkExecuteStateTransition-4   	         25	35901941409 ns/op	7465058127 B/op	111046635 allocs/op
BenchmarkExecuteStateTransition_WithCache-4    25	24352836461 ns/op	716078697 B/op	22348530 allocs/op
BenchmarkProcessEpoch_2FullEpochs-4   	       5	177559078808 ns/op	12754314974 B/op	176571470 allocs/op
BenchmarkHashTreeRoot_FullState-4   	         50	1321382095 ns/op	253959577 B/op	 9645648 allocs/op
```