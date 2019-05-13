## Utility to Drain All Deposit Contracts

This is a utility to help users drain the contract addresses they have deployed in order to get their testnet ether back. To run the utility, it defaults to an infura link but you can use your own provider through the flags. The utility will print out each address it sends a transaction to.

### Usage

_Name:_
**drainContracts** - this is a util to drain all deposit contracts

_Usage:_
drainContracts [global options]

_Flags:_

- --keystoreUTCPath value keystore JSON for account
- --httpPath value HTTP-RPC server listening interface (default: "http://localhost:8545/")
- --passwordFile value Password file for unlock account (default: "./password.txt")
- --privKey value Private key to unlock account
- --help, -h show help
- --version, -v print the version

### Example

To use private key with default RPC:

```
bazel run //contracts/deposit-contract/drainContracts -- --httpPath=https://goerli.prylabs.net --privKey=$(echo /path/to/private/key/file)
```

### Output

```
current address is 0xdbA543721462680431eC4eeB26163079B3645660
nonce is 7060
0xd1faa3f9bca1d698df559716fe6d1c9999155b38d3158fffbc98d76d568091fc
1190 chain start logs found
1190 contracts ready to drain found
Contract address 0x4cb8976E4Bf0b6A462AF8704F0f724775B67b4Ce drained in TX hash: 0x3f963c30c4fd4ff875c641be1e7b873bfe02ae2cd2d73554cc6087c2d3acaa9e
```
