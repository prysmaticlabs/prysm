# Prysmatic Labs Sharding Implementation

![Travis Build](https://travis-ci.org/prysmaticlabs/geth-sharding.svg?branch=master)

This is the main repository for the sharding implementation for the go-ethereum project by [Prysmatic Labs](https://prysmaticlabs.com). For the original, go-ethereum project, refer to the following [link](https://github.com/ethereum/go-ethereum).

Before you begin, check out our [Contribution Guidelines](#contribution-guidelines) and join our active chat room on Gitter below:

[![Gitter](https://badges.gitter.im/Join%20Chat.svg)](https://gitter.im/prysmaticlabs/geth-sharding?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge)


Also, read our [Sharding Reference Implementation Doc](https://github.com/prysmaticlabs/geth-sharding/blob/master/sharding/README.md). This doc serves as a source of truth for the sharding implementation we follow at Prysmatic Labs.


# Table of Contents

-   [Installation](#installation)
-   [Sharding Instructions](#sharding)
    -   [Running a Local Geth Node](#running-a-local-geth-node)
    -   [Transaction Generator](#transaction-generator)
    -   [Becoming an Attester](#becoming-an-attester)
    -   [Running a Collation Proposal Node](#running-a-collation-proposal-node)
-   [Testing](#testing)
-   [Contributing](#contributing)
-   [License](#license)

# Installation

Create a folder in your `$GOPATH` and navigate to it

```
$ mkdir -p $GOPATH/src/github.com/ethereum && cd $GOPATH/src/github.com/ethereum
```

Clone our repository as `go-ethereum`

```
$ git clone https://github.com/prysmaticlabs/geth-sharding ./go-ethereum
```

For prerequisites and detailed build instructions please read the
[Installation Instructions](https://github.com/ethereum/go-ethereum/wiki/Building-Ethereum)
on the wiki.

Building geth requires both a Go (version 1.7 or later) and a C compiler.
You can install them using your favourite package manager.
Once the dependencies are installed, run

```
$ make geth
```

or, to build the full suite of utilities:

```
$ make all
```

# Sharding Instructions

To get started with running the project, follow the instructions to initialize your own private Ethereum blockchain and geth node, as they will be required to run before you can begin proposing collations into shard chains.

## Running a Local Geth Node

To start a local Geth node, you can create your own `genesis.json` file similar to:

```json
{
    "config": {
        "chainId": 12345,
        "homesteadBlock": 0,
        "eip155Block": 0,
        "eip158Block": 0
    },
    "difficulty": "200",
    "gasLimit": "210000000000",
    "alloc": {
        "826f3F66dB0416ea82033aE917A611bfBF4D98b6": { "balance": "300000" },
    }
}
```

The `alloc` portion specifies account addresses with prefunded ETH when the Ethereum blockchain is created. You can modify this section of the genesis to include your own test address and prefund it with 100ETH.

Then, you can build `geth` and init a new instance of a local, Ethereum blockchain as follows:

    $ make geth
    $ ./build/bin/geth init /path/to/genesis.json -datadir /path/to/your/datadir
    $ ./build/bin/geth --nodiscover console --datadir /path/to/your/datadir --networkid 12345

It is **important** to note that the `--networkid` flag must match the `chainId` property in the genesis file.

Then, the geth console can start up and you can start a miner as follows:

    > personal.newAccount()
    > miner.setEtherbase(eth.accounts[0])
    > miner.start(1)

Now, save the passphrase you used in the geth node into a text file called password.txt. Then, once you have this private geth node running on your local network, we will need to generate test, pending transactions that can then be processed into collations by proposers. For this, we have created an in-house transaction generator CLI tool.

## Transaction Generator

Work in Progress. To track our current draft of the tx generator cli spec, visit this [link](https://docs.google.com/document/d/1YohsW4R9dIRo0u5RqfNOYjCkYKVCmzjgoBDBYDdu5m0/edit?usp=drive_web&ouid=105756662967435769870). Generating test transactions on a local network will allow for benchmarking of tx throughput within our system.

## Becoming an Attester

Our system outlined below follows the [Minimal Sharding Protocol](https://ethresear.ch/t/a-minimal-sharding-protocol-that-may-be-worthwhile-as-a-development-target-now/1650) as outlined by Vitalik on ETHResearch where any actor can submit collation headers via the SMC, but only a selected committee of attesters is allowed to vote on collations in each period. Attesters are in charge of data availability checking and consensus is reached upon a collation header receiving >= 2/3 votes in a period.

To deposit ETH and join as an attester in the Sharding Manager Contract, run the following command:

```
geth sharding --actor "attester" --deposit --datadir /path/to/your/datadir --password /path/to/your/password.txt --networkid 12345
```

This will extract 1000ETH from your account balance and insert you into the SMC's attesters. Then, the program will listen for incoming block headers and notify you when you have been selected as to vote on proposals for a certain shard in a given period. Once you are selected, your sharding node will download collation information to check for data availability on vote on proposals that have been submitted via the `addHeader` function on the SMC.

Concurrently, you will need to run another service that is tasked with processing transactions into collations and submitting them to the SMC via the `addHeader` function.

## Running a Collation Proposal Node

```
geth sharding --actor "proposer" --datadir /path/to/your/datadir --password /path/to/your/password.txt --shardid 0 --networkid 12345
```

This node is tasked with processing pending transactions into blobs within collations by serializing data into collation bodies. It is responsible for submitting proposals on shard 0 (collation headers) to the SMC via the `addHeader` function.

## Running an Observer Node

    geth sharding --datadir /path/to/your/datadir --password /path/to/your/password.txt --shardid 0 --networkid 12345

Omitting the `--actor` flag will launch a simple observer service attached to the sharding client that is able to listen to changes happening throughout the sharded Ethereum network on shard 0.

# Making Changes

## Rebuilding the Sharding Manager Contract Bindings

The Sharding Manager Contract is built in Solidity and deployed to a running geth node upon launch of the sharding node if it does not exist in the network at a specified address. If there are any changes to the SMC's code, the Golang bindigs must be rebuilt with the following command.

    go generate github.com/prysmaticlabs/geth-sharding/sharding
    # OR
    cd sharding && go generate

# Testing

To run the unit tests of our system do:

```
go test github.com/prysmaticlabs/geth-sharding/sharding
```

We will require more complex testing scenarios (fuzz tests) to measure the full integrity of the system as it evolves.

# Contributing

We have put all of our contribution guidelines into [CONTRIBUTING.md](https://github.com/prysmaticlabs/geth-sharding/blob/master/sharding/CONTRIBUTING.md)! Check it out to get started.

![nyancat](https://encrypted-tbn0.gstatic.com/images?q=tbn:ANd9GcRBSus2ozk_HuGdHMHKWjb1W5CmwwoxmYIjIBmERE1u-WeONpJJXg)

# License

The go-ethereum library (i.e. all code outside of the `cmd` directory) is licensed under the
[GNU Lesser General Public License v3.0](https://www.gnu.org/licenses/lgpl-3.0.en.html), also
included in our repository in the `COPYING.LESSER` file.

The go-ethereum binaries (i.e. all code inside of the `cmd` directory) is licensed under the
[GNU General Public License v3.0](https://www.gnu.org/licenses/gpl-3.0.en.html), also included
in our repository in the `COPYING` file.
