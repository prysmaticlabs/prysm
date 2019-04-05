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
| ProcessBlockRandao       | 1.324 μs | 1.331 μs | 1.420 μs | dddddd | dddsad |
| ProcessEth1Data          | 1.645 μs  | 1.479 μs | 1.472 μs | dddddd | dddsad |
| ProcessProposerSlashings | 9.717 ms  |  390 ns  | 210.27 ms | dddddd | dddsad |
| ProcessAttesterSlashings | 4.753 ms  |  268 ns  | 105.17 ms | dddddd | dddsad |
| ProcessBlockAttestations | 127.69 μs | 13.62 μs | 116.98 μs | dddddd | dddsad |
| ProcessValidatorDeposits | 3.042 ms  | 61.147 ms | 60.855 ms | dddddd | dddsad |
| ProcessValidatorExits    |  486 ns   |  271 ns  | 450 ns  | dddddd | dddsad |
| ProcessBlock             | 17.708 ms | 61.821 ms | 375.97 ms | dddddd | dddsad |


### Epoch Processing

The epoch-processing benches are marked with the following:

* BIG - indicates max possible conditions.
* SML - indicates conditions more similar to real time.

#### Laptop

| Benchmark         | 16K BIG | 300K SML | 300K BIG | 4M SML | 4M BIG |
| ----------------- | ------- | -------- | -------- | ------ | ------ |
| process_eth1_data | 4324234 | dsadasdd | dsadaddd | dasddd | dadddd |
| process_eth1_data | 4324234 | dsadasdd | dsadaddd | dddddd | dddsad |
| process_eth1_data | 4324234 | dsadasdd | dsadaddd | dddddd | dddsad |
| process_eth1_data | 4324234 | dsadasdd | dsaddadd | dddddd | dddsad |
| process_eth1_data | 4324234 | dsadasdd | dsaddadd | dddddd | dddsad |
| process_eth1_data | 4324234 | dsadasdd | dsadaddd | dddddd | dddsad |