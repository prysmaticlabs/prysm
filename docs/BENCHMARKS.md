# Prysm ETH2.0 Client Benchmarks

This document details the results of benchmarking the Prysm ETH2.0 Client.

## Table of Contents:

- Results

## Info / Disclaimers

- All benchmarks were performed using the Prysm ETH2.0 Client (golang).
- The tests are purposely stressful to demonstrate worst case conditions.
- All benchmarks are purely functional, no DB interations or network calls.

## Results

These benchmarks were performed with the following conditions:

- 65K Validators
- 1 Attester Slashing per slot
- 1 Proposer Slashing per slot
- 16 Attestations per slot
- 2 Deposits per slot
- 2 Exits per slot

### Block Processing

| Benchmark                |            65K |
| :----------------------- | -------------: |
| ProcessBlockHeader       |  1297706 ns/op |
| ProcessBlockRandao       |      728 ns/op |
| ProcessEth1Data          |      806 ns/op |
| ProcessValidatorExits    |  1585956 ns/op |
| ProcessProposerSlashings |     3399 ns/op |
| ProcessAttesterSlashings |     3796 ns/op |
| ProcessBlockAttestations |   505450 ns/op |
| ProcessValidatorDeposits | 11020831 ns/op |
| ProcessBlock             | 26115600 ns/op |

### Epoch Processing

| Benchmark                           |             16K |
| :---------------------------------- | --------------: |
| ProcessJustificationAndFinalization |       226 ns/op |
| ProcessCrosslinks                   |   7567184 ns/op |
| ProcessRewardsAndPenalties          | 142134735 ns/op |
| ProcessRegistryUpdates              |    937718 ns/op |
| ProcessSlashings                    |    609449 ns/op |
| ProcessFinalUpdates                 |  11787574 ns/op |
| ProcessEpoch                        | 150747803 ns/op |
| ActiveValidatorIndices              |       201 ns/op |
| ValidatorIndexMap                   |  11838948 ns/op |
