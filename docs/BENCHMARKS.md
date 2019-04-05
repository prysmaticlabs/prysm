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
| ProcessBlockRandao      | 650.38 μs | 14.367 ms | 25.047 ms | dddddd | dddsad |
| ProcessEth1Data          | 1.523 μs | 2.216 μs | 4.422 μs | dddddd | dddsad |
| ProcessProposerSlashings | 9.950 ms | 4.145 μs | 288.01 ms | dddddd | dddsad |
| ProcessAttesterSlashings | 5.015 ms | 1.479 μs | 140.65 ms | dddddd | dddsad |
| ProcessBlockAttestations | 127.27 μs | 19.20 μs | 116.3 μs | dddddd | dddsad |
| ProcessValidatorDeposits | 2.992 ms | 81.012 ms | 82.948 ms | dddddd | dddsad |
| ProcessValidatorExits    | 487 ns   | 510 ns | 3.240 μs  | dddddd | dddsad |
| ProcessBlock            | 18.357 ms | 91.70 ms | 468.05 ms | dddddd | dddsad |


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