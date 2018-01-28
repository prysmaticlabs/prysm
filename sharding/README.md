## Go-Ethereum Sharding

[![Join the chat at https://gitter.im/prysmaticlabs/geth-sharding](https://badges.gitter.im/prysmaticlabs/geth-sharding.svg)](https://gitter.im/prysmaticlabs/geth-sharding?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge&utm_content=badge)

This repository contains the sharding implementation for the go-ethereum client. The system consists of an entry point that serves as a bridge between a **Validator Manager Contract** and a Geth node running on the same network.

To get started with running the project, follow the instructions to initialize your own private Ethereum blockchain and geth node:

```
$ make geth
$ ./build/bin/geth init ./sharding/genesis.json -datadir /path/to/your/datadir
$ ./build/bin/geth --nodiscover console --datadir /path/to/your/datadir --networkid 12345
```

Then, the geth console can start up and you can start a miner as follows:

```
> personal.newAccount()
> miner.setEtherbase(eth.accounts[0])
> miner.start()
```

Then, once you are satisfied with mining for a few seconds, stop the miner with

```
> miner.stop()
```

Now, save the passphrase you used in the geth node into a text file called `password.txt`. Then, once you have this private geth node running on your local network, the sharding client can be started as a standalone geth command as follows in a separate terminal window:

```
$ ./build/bin/geth shard --datadir=/path/to/your/datadir --password=password.txt
```

The project consists of the following parts, with each of them requiring comprehensive tests:

### Validator Manager Contract

The VMC is built in Solidity and deployed to the geth node upon launch of the client if it does not exist in the network at a specified address. If the contract already exists, the client simply sets up an interface to programmatically call the internal contract functions and listens to transactions broadcasted to the geth node to begin the sharding system.

### VMC Wrapper & Sharding Client

As we will be interacting with a geth node, we will create a Golang interface that wraps over the VMC and a client that connects to the local geth node upon launch via JSON-RPC.

It will be the client's responsibility to listen to any new broadcasted transactions to the node and interact package validators up to be sent to the VMC.

### Sharding VM

As sharding will require a different set of protocol primitives, we will have to specify new primitives for Blocks, Transactions, and even the low-level functioning of the EVM to accommodate this new structure.

We can implement a new ShardingEVM by overriding some of the regular EVM interface's methods in how it executes transactions. Our sharding client will process transactions using this modified vm.

An example of the current approach followed in the python implementation can be found [here](https://github.com/ethereum/py-evm/blob/sharding/evm/vm/forks/sharding/__init__.py)

In this case, the VM of the sharding client is set to be the subclassed ByzantiumVM that has its methods overwritten and sharding transactions primitives specified.

### Contributing

We will be tracking progress on these major milestones as through WIP pull requests. If you have any thoughts on major parts of the system that will still need to be implemented, message the Prysmatic Labs team gitter channel.
