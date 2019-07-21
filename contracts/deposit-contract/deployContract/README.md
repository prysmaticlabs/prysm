## Utility to Deploy Deposit Contract

This is a utility to help users deploy deposit contract for running their own beacon chain node in a local containerized set up. To run the utility, it assumes there is a running geth node as a separate process attached to proof-of-work main chain. The utility will deploy the validator registration contract and print out the contract address. Users will pass the contract address to the beacon chain node to monitor when they have been conducted to become an active validator.

### Usage

*Name:*  
   **deployContract** - this is a util to deploy deposit contract

*Usage:*  
   deployContract [global options] command [command options] [arguments...]

*Flags:*  
- --skipChainstartDelay    Whether to skip ChainStart log being fired a day later
- --ipcPath value          Filename for IPC socket/pipe within the datadir
- --httpPath value         HTTP-RPC server listening interface (default: "http://localhost:8545/")
- --passwordFile value     Password file for unlock account (default: "./password.txt")
- --privKey value          Private key to unlock account
- --k8sConfig value        Name of kubernetes config map to update with the contract address
- --chainStart value       Number of validators required for chain start (default: 16384)
- --minDeposit value       Minimum deposit value allowed in contract (default: 1000000000)
- --maxDeposit value       Maximum deposit value allowed in contract (default: 32000000000)
- --help, -h               show help
- --version, -v            print the version

### Example

To use private key with default RPC:

```
bazel run //contracts/deposit-contract/deployContract -- --httpPath=https://goerli.prylabs.net --privKey=$(echo /path/to/private/key/file) --chainStart=8 --minDeposit=100 --maxDeposit=3200
```


### Output

```
INFO[0001] New contract deployed at 0x5275C2220C574330E230bFB7e4a0b96f60a18f02 
```
