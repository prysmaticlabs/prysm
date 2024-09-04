# Prysm Client Interoperability Guide

This README details how to setup Prysm for interop testing for usage with other Ethereum consensus clients.

## Installation & Setup

1. Install [Bazel](https://docs.bazel.build/versions/master/install.html) **(Recommended)**
2. `git clone https://github.com/prysmaticlabs/prysm && cd prysm`
3. `bazel build //cmd/...`

## Starting from Genesis

Prysm can be started from a built-in mainnet genesis state, or started with a provided genesis state by
using the `--genesis-state` flag and providing a path to the genesis.ssz file.

## Generating a Genesis State

To setup the necessary files for these quick starts, Prysm provides a tool to generate a `genesis.ssz` from
a deterministically generated set of validator private keys following the official interop YAML format 
[here](https://github.com/ethereum/eth2.0-pm/blob/master/interop/mocked_start).

You can use `bazel run //cmd/prysmctl -- testnet generate-genesis` to create a deterministic genesis state for interop.

### Usage

- **--genesis-time-delay** uint: The number of seconds in the future to define genesis. Example: a value of 60 will set the genesis time to 1 minute in the future. This should be sufficiently large enough to allow for you to start the beacon node before the genesis time. 
- **--num-validators** int: Number of validators to deterministically include in the generated genesis state
- **--output-ssz** string: Output filename of the SSZ marshaling of the generated genesis state
- **--chain-config-file** string: Filepath to a chain config yaml file.

The example below creates 64 validator keys, instantiates a genesis state with those 64 validators and with genesis unix timestamp 1567542540,
and finally writes a ssz encoded output to ~/Desktop/genesis.ssz. A chain config file could be any valid config yaml such as the [minimal.yaml](https://github.com/ethereum/consensus-specs/blob/dev/configs/minimal.yaml) file.

Note: Use the bazel flag `--config=minimal` to run Prysm software with minimal state build constraints. This is required to run Prysm in the "minimal" configuration.

```
bazel run //cmd/prysmctl --config=minimal -- \
  testnet generate-genesis \
  --chain-config-file=~/Downloads/minimal.yaml \
  --output-ssz=~/Desktop/genesis.ssz \
  --num-validators=64 \
  --genesis-time-delay=60
```

## Launching a Beacon Node + Validator Client

### Launching from Pure CLI Flags

Open up two terminal windows, run:

```
bazel run //beacon-chain --config=minimal -- \
  --bootstrap-node= \
  --deposit-contract 0x8A04d14125D0FDCDc742F4A05C051De07232EDa4 \
  --datadir=/tmp/beacon-chain-minimal-devnet \
  --force-clear-db \
  --min-sync-peers=0 \
  --genesis-ssz=~/Desktop/genesis.ssz \
  --chain-config-file=~/Downloads/minimal.yaml
```

This will start the system with 64 validators. The flags used can be explained as such:

- `bazel run //cmd/beacon-chain --config=minimal` builds and runs the beacon node in minimal build configuration.
- `--` is a flag divider to distingish between bazel flags and flags that should be passed to the application. All flags and arguments after this divider are passed to the beacon chain.
- `--bootstrap-node=` disables the default bootstrap nodes. This prevents the client from attempting to peer with mainnet nodes.
- `--datadir=/tmp/beacon-chain-minimal-devnet` sets the data directory in a temporary location. Change this to your preferred destination.
- `--force-clear-db` will delete the beaconchain.db file without confirming with the user. This is helpful for iteratively running local devnets without changing the datadir, but less helpful for one off runs where there was no database in the data directory.
- `--min-sync-peers=0` allows the beacon node to skip initial sync without peers. This is essential because Prysm expects at least a few peers to start start the blockchain.
- `--genesis-ssz=~/Desktop/genesis.ssz` defines the path to the generated genesis ssz file. The beacon node will use this as the initial genesis state.
- `--chain-config-file=~/Downloads/minimal.yaml` defines the path to the yaml file with the chain configuration.

As soon as the beacon node has started, start the validator in the other terminal window. 

```
bazel run //validator --config=minimal -- --keymanager=interop --keymanageropts='{"keys":64}'
```

This will launch and kickstart the system with your 64 validators performing their duties accordingly.
