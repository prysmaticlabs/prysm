## Utility to Deploy Deposit Contract

This is a utility to help users deploy deposit contract for running their own beacon chain node in a local containerized set up. To run the utility, it assumes there is a running geth node as a separate process attached to proof-of-work main chain. The utility will deploy the validator registration contract and print out the contract address. Users will pass the contract address to the beacon chain node to monitor when they have been conducted to become an active validator.

### Usage

*Name:*  
   **sendDepositTx** - this is a util to send deposit transactions

*Usage:*  
   sendDepositTx [global options] command [command options] [arguments...]

*Flags:*  
- --keystoreUTCPath value   Location of keystore
- --ipcPath value           Filename for IPC socket/pipe within the datadir
- --httpPath value          HTTP-RPC server listening interface (default: "http://localhost:8545/")
- --passwordFile value      Password file for unlock account (default: "./password.txt")
- --privKey value           Private key to unlock account
- --depositContract value   Address of the deposit contract
- --numberOfDeposits value  number of deposits to send to the contract (default: 8)
- --depositAmount value     Maximum deposit value allowed in contract(in gwei) (default: 3200)
- --depositDelay value      The time delay between sending the deposits to the contract(in seconds) (default: 5)
- --variableTx              This enables variable transaction latencies to simulate real-world transactions
- --txDeviation value       The standard deviation between transaction times (default: 2)
- --help, -h                show help
- --version, -v             print the version


### Example

To use private key with default RPC:

```
bazel run //contracts/deposit-contract/sendDepositTx -- --httpPath=https://goerli.prylabs.net --keystoreUTCPath /path/to/keystore --passwordFile /path/to/password --depositDelay 2  --depositContract 0x767E9ef9610Abb992099b0994D5e0c164C0813Ab

```


### Output

```
INFO main: Deposit 7 sent to contract for validator with a public key 0x333362343964316561623337336433313433356233626330393866653262613162333631333965326235613033303933643966396238356231363566653635646166383738396164356637343035313665353563666633346665343339653038656239306236313863303962326364653036646539333435643635366437333032643961623964336163323965636336663739613137656533663333323538656436383638623161393862363738383932636334306565336634333865373031 
Transaction Hash=[213 23 244 203 91 45 79 72 109 141 43 113 67 92 178 94 24 209 39 240 111 59 238 18 189 145 140 166 49 236 157 71]
```
