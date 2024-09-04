# Prysm Client Interoperability Guide

This README details how to setup Prysm for interop testing for usage with other Ethereum consensus clients.

## Installation & Setup

1. Install [Bazel](https://docs.bazel.build/versions/master/install.html) **(Recommended)**
2. `git clone https://github.com/prysmaticlabs/prysm && cd prysm`
3. `bazel build //...`

## Starting from Genesis

Prysm can be started from a built-in mainnet genesis state, or started with a provided genesis state by
using the `--genesis-state` flag and providing a path to the genesis.ssz file.

## Generating a Genesis State

To setup the necessary files for these quick starts, Prysm provides a tool to generate a `genesis.ssz` from
a deterministically generated set of validator private keys following the official interop YAML format 
[here](https://github.com/ethereum/eth2.0-pm/blob/master/interop/mocked_start).

You can use `bazel run //cmd/prysmctl -- testnet generate-genesis` to create a deterministic genesis state for interop.

### Usage

- **--genesis-time** uint: Unix timestamp used as the genesis time in the generated genesis state (defaults to now)
- **--num-validators** int: Number of validators to deterministically include in the generated genesis state
- **--output-ssz** string: Output filename of the SSZ marshaling of the generated genesis state
- **--chain-config-file** string: Filepath to a chain config yaml file.

The example below creates 64 validator keys, instantiates a genesis state with those 64 validators and with genesis unix timestamp 1567542540,
and finally writes a ssz encoded output to ~/Desktop/genesis.ssz.

```
bazel run //cmd/prysmctl -- testnet generate-genesis --chain-config-file ~/Desktop/interop.yaml --output-ssz ~/Desktop/genesis.ssz --num-validators 64 --genesis-time 1567542540
```

## Launching a Beacon Node + Validator Client

### Launching from Pure CLI Flags

Open up two terminal windows, run:

```
bazel run //beacon-chain -- \
--bootstrap-node= \
--deposit-contract 0x8A04d14125D0FDCDc742F4A05C051De07232EDa4 \
--datadir=/tmp/beacon-chain-interop \
--force-clear-db \
--min-sync-peers=0 \
--genesis-ssz=~/Desktop/genesis.ssz \
--chain-config-file=~/Desktop/interop.yaml
```

This will deterministically generate a beacon genesis state and start
the system with 64 validators and the genesis time set to the current unix timestamp.
Wait a bit until your beacon chain starts, and in the other window:

```
bazel run //validator -- --keymanager=interop --keymanageropts='{"keys":64}'
```

This will launch and kickstart the system with your 64 validators performing their duties accordingly.
