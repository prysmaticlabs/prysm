## Utility to Update Beacon State Genesis Time

This is a utility to help users update genesis time of an input beacon state

### Usage

_Name:_
**update-genesis-time** - this is a utility to update genesis time of a beacon state

_Usage:_
update-genesis-time [global options]

_Flags:_

- --input-ssz-state: Input filename of the SSZ marshaling of the genesis state
- --genesis-time: Unix timestamp used as the genesis time in the generated genesis state (defaults to now)

### Example

To use private key with default RPC:

```
bazel run //tools/update-genesis-time -- --input-ssz-state=/tmp/genesis.ssz
```

### Output

```
INFO: Elapsed time: 5.887s, Critical Path: 4.99s
INFO: 41 processes: 41 darwin-sandbox.
INFO: Build completed successfully, 44 total actions
INFO: Build completed successfully, 44 total actions
2020/04/28 11:55:21 No --genesis-time specified, defaulting to now
2020/04/28 11:55:21 Done writing to /tmp/genesis.ssz
```
