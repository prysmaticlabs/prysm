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

- `SSZ Genesis`: Launches a beacon node  from a .ssz file containing a SSZ-encoded, genesis beacon state  **(Recommended)**
- `Yaml Genesis`: Launches a beacon node from a .yaml file specifying a genesis beacon state - binary data is _base64 encoded_
- `JSON Genesis`: Launches a beacon node  from a .json file specifying a genesis beacon state - binary data is _base64 encoded_

## Generating a Genesis State

To setup the necessary files for these quick starts, Prysm provides a tool to generate `genesis.yaml`, `genesis.ssz`, `genesis.json` from an
a deterministically generated set of validator private keys following the official interop YAML format 
[here](https://github.com/ethereum/eth2.0-pm/blob/master/interop/mocked_start). If you already have a genesis state in this format, you can skip this section.

You can use `bazel run //tools/genesis-state-gen` to create a deterministic genesis state for interop.

### Usage

- **--genesis-time** uint: Unix timestamp used as the genesis time in the generated genesis state
- **--mainnet-config** bool: Select whether genesis state should be generated with mainnet or minimal (default) params
- **--num-validators** int: Number of validators to deterministically include in the generated genesis state
- **--output-json** string: Output filename of the JSON marshaling of the generated genesis state
- **--output-ssz** string: Output filename of the SSZ marshaling of the generated genesis state
- **--output-yaml** string: Output filename of the YAML marshaling of the generated genesis state

The example below creates 10 validator keys, instantiates a genesis state with those 10 validators and with genesis time 1567542540,
and finally writes a YAML encoded output to ~/Desktop/genesis.yaml. This file can be used to kickstart the beacon chain in the next section.

```
bazel run //tools/genesis-state-gen -- --output-yaml ~/Desktop/genesis.yaml --num-validators 10 --genesis-time 1567542540
```

## Launching a Beacon Node + Validator Client

TODO: Add section after incorporating the mock start functionality into the beacon chain and validator bazel binaries.