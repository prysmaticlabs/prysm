# Prysm Client Interoperability Guide

This README details how to setup Prysm for interop testing for usage with other Ethereum 2.0 clients.

**WARNING**: The tool can only generate up to 190 private keys at the moment, as there is a BLS bug that
prevents deterministically generating more keys than those. 

## Installation & Setup

1. Install [Bazel](https://docs.bazel.build/versions/master/install.html) **(Recommended)**
2. `git clone https://github.com/prysmaticlabs/prysm && cd prysm`
3. `bazel build //...`

## Starting from Genesis

Prysm supports a few ways to quickly launch a beacon node from basic configurations:

- `NumValidators + GenesisTime`: Launches a beacon node by deterministically generating a state from a num-validators flag along with a genesis time **(Recommended)**
- `SSZ Genesis`: Launches a beacon node  from a .ssz file containing a SSZ-encoded, genesis beacon state

## Generating a Genesis State

To setup the necessary files for these quick starts, Prysm provides a tool to generate a `genesis.ssz` from
a deterministically generated set of validator private keys following the official interop YAML format 
[here](https://github.com/ethereum/eth2.0-pm/blob/master/interop/mocked_start).

You can use `bazel run //tools/genesis-state-gen` to create a deterministic genesis state for interop.

### Usage

- **--genesis-time** uint: Unix timestamp used as the genesis time in the generated genesis state
- **--mainnet-config** bool: Select whether genesis state should be generated with mainnet or minimal (default) params
- **--num-validators** int: Number of validators to deterministically include in the generated genesis state
- **--output-ssz** string: Output filename of the SSZ marshaling of the generated genesis state

The example below creates 64 validator keys, instantiates a genesis state with those 64 validators and with genesis unix timestamp 1567542540,
and finally writes a YAML encoded output to ~/Desktop/genesis.yaml. This file can be used to kickstart the beacon chain in the next section.

```
bazel run //tools/genesis-state-gen -- --output-yaml ~/Desktop/genesis.yaml --num-validators 64 --genesis-time 1567542540
```

## Launching a Beacon Node + Validator Client

### Launching from Pure CLI Flags

Open up two terminal windows, run:

```
bazel run //beacon-chain -- \
--no-genesis-delay \
--bootstrap-node= \
--deposit-contract 0xD775140349E6A5D12524C6ccc3d6A1d4519D4029 \
--clear-db \
--interop-num-validators 64 \
--interop-genesis-time=$(date +%s) \
--interop-eth1data-votes
```

This will deterministically generate a beacon genesis state and start
the system with 64 validators and the genesis time set to the current unix timestamp.
Wait a bit until your beacon chain starts, and in the other window:

```
bazel run //validator -- --interop-num-validators 64
```

This will launch and kickstart the system with your 64 validators performing their duties accordingly.
specify which keys 

### Launching from `genesis.ssz`

Assuming you generated a `genesis.ssz` file with 64 validators, open up two terminal windows, run:

```
 bazel run //beacon-chain -- \
--no-genesis-delay \
--bootstrap-node= \
--deposit-contract 0xD775140349E6A5D12524C6ccc3d6A1d4519D4029 \
--clear-db \
--interop-genesis-state /path/to/genesis.ssz \
--interop-eth1data-votes
```

Wait a bit until your beacon chain starts, and in the other window:

```
bazel run //validator -- --interop-num-validators 64
```

This will launch and kickstart the system with your 64 validators performing their duties accordingly.






