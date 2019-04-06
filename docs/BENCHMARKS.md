# Prysm ETH2.0 Client Benchmarks

This document details the results of benchmarking the Prysm ETH2.0 Client.

## Table of Contents:

* Results
* Per-epoch processing details

## Info / Disclaimers

* All benchmarks were performed using the Prysm ETH2.0 Client (golang).
* The tests are purposely stressful to demonstrate worst case conditions.
* All benchmarks are purely functional, no DB interations or network calls.

## Results

The following conditions are benched with the following conditions and indicated as so:

* 16K: 16,000 validators
* 300K: 300,000 validators
* 4M: 4,000,000 validators

### Block Processing

The block-processing benches are marked with the following:

* BIG - indicates max possible conditions.
* SML - indicates conditions more similar to real time.

#### Laptop

| Benchmark             |   16K BIG  | 300K SML | 300K BIG | 4M SML | 4M BIG |
| --------------------- | ---------- | -------- | -------- | ------ | ------ |
| ProcessBlockRandao       | 1.324 μs | 1.331 μs | 1.420 μs | ----- | dddsad |
| ProcessEth1Data          | 1.645 μs  | 1.479 μs | 1.472 μs | ----- | dddsad |
| ProcessProposerSlashings | 9.717 ms  |  390 ns  | 210.27 ms | ----- | dddsad |
| ProcessAttesterSlashings | 4.753 ms  |  268 ns  | 105.17 ms | ------ | dddsad |
| ProcessBlockAttestations | 127.69 μs | 13.62 μs | 116.98 μs | ------- | dddsad |
| ProcessValidatorDeposits | 3.042 ms  | 61.147 ms | 60.855 ms | ------ | dddsad |
| ProcessValidatorExits    |  486 ns   |  271 ns  | 450 ns  | ------ | dddsad |
| ProcessBlock             | 17.708 ms | 61.821 ms | 375.97 ms | ------ | dddsad |


### Epoch Processing

The epoch-processing benches are done with the following:


#### Laptop

| Benchmark             | 16K     |    300K    |   4M    |
| --------------------- | ------- | ---------- | -------- |
| ProcessEth1Data       | 240 ns  |         -   | 559 ns |
| ProcessJustification  | 308 ns   |        -   | 478 ns  |
| ProcessCrosslinks     | 208.19 ms |  4.575 s  |  -       |
| ProcessRewards         | 1.088 ms | 23.714 ms | 431.21 ms |
| ProcessLeak             | 1.471 ms | 33.526 ms | 588.44 ms |
| ProcessPenaltiesAndExit  | 283.02 μs | 7.601 μs | 138.49 ms |
| ProcessEjections          | 47.482 μs | 22.410 μs | 40.965 ms |
| UpdateRegistry             | 130.35 μs | 65.433 μs | 110.18 ms |
| CleanupAttestations         |   398 ns  |  424 ns  |  838 ns  |
| UpdateLatestActiveIndexRoots | 51.419 μs | 23.504 μs | 42.01 ms |
| UpdateLatestSlashedBalances | 270 ns    |   271 ns  |  425 ns   |
| ActiveValidatorIndices      | 42.683 μs | 24.075 μs | 38.418 ms |
| ValidatorIndexMap           | 2.558 ms  |  60.816 ms  | 1.4297 s |
