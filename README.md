# Prysmatic Labs Ethereum 2.0 Implementation

![Travis Build](https://travis-ci.org/prysmaticlabs/prysm.svg?branch=master)

This is the main repository for the beacon chain and sharding implementation for Ethereum 2.0 [Prysmatic Labs](https://prysmaticlabs.com). 

Before you begin, check out our [Contribution Guidelines](#contribution-guidelines) and join our active chat room on Gitter below:

[![Gitter](https://badges.gitter.im/Join%20Chat.svg)](https://gitter.im/prysmaticlabs/prysm?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge)


Also, read our [Sharding Reference Implementation Doc](https://github.com/prysmaticlabs/prysm/blob/master/client/README.md). This doc provides a background on the sharding implementation we follow at Prysmatic Labs.


# Table of Contents

-   [Installation](#installation)
-   [Sharding Instructions](#sharding)
-   [Testing](#testing)
-   [Contributing](#contributing)
-   [License](#license)

# Installation

Create a folder in your `$GOPATH` and navigate to it

```
mkdir -p $GOPATH/src/github.com/prysmaticlabs && cd $GOPATH/src/github.com/prysmaticlabs
```

Note: it is not necessary to clone to the gopath if you're only building with Bazel. 

Clone our repository: 

```
git clone https://github.com/prysmaticlabs/prysm
```

Download the Bazel build tool by Google [here](https://docs.bazel.build/versions/master/install.html) and ensure it works by typing

```
bazel version
```

Bazel manages all of the dependencies for you (including go and necessary compilers) so you are all set to build prysm.

# Sharding Instructions

To get started with running the project, follow the instructions to initialize your own private Ethereum blockchain and geth node, as they will be required to run before you can begin running our system

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

Then, you can build and init a new instance of a local, Ethereum blockchain as follows:

    $ geth init /path/to/genesis.json -datadir /path/to/your/datadir
    $ geth --nodiscover console --datadir /path/to/your/datadir --networkid 12345

It is **important** to note that the `--networkid` flag must match the `chainId` property in the genesis file.

Then, the geth console can start up and you can start a miner as follows:

    > personal.newAccount()
    > miner.setEtherbase(eth.accounts[0])
    > miner.start(1)

Now, save the passphrase you used in the geth node into a text file called password.txt. Then, once you have this private geth node running on your local network, we will need to generate test, pending transactions that can then be processed into collations by proposers. For this, we have created an in-house transaction generator CLI tool.


# Sharding Minimal Protocol 

**NOTE**: This section is in flux: will be deprecated in favor of a beacon chain)

Build our system first

```
$ bazel build //client/...
```

## Becoming a Notary


Make sure a geth node is running as a separate process. Then, to deposit ETH and join as a notary in the Sharding Manager Contract, run the following command:

```
bazel run //client -- \
   --actor "notary" \
   --deposit \
   --datadir /path/to/your/datadir \
   --password /path/to/your/password.txt \
   --networkid 12345
```

This will extract 1000ETH from your account balance and insert you into the SMC's notaries. Then, the program will listen for incoming block headers and notify you when you have been selected as to vote on proposals for a certain shard in a given period. Once you are selected, your sharding node will download collation information to check for data availability on vote on proposals that have been submitted via the `addHeader` function on the SMC.

Concurrently, you will need to run another service that is tasked with processing transactions into collations and submitting them to the SMC via the `addHeader` function. 

## Running a Collation Proposal Node

```
bazel run //client -- \
   --actor "proposer" \
   --datadir /path/to/your/datadir \
   --password /path/to/your/password.txt \
   --shardid 0 \
   --networkid 12345
```

This node is tasked with processing pending transactions into blobs within collations by serializing data into collation bodies. It is responsible for submitting proposals on shard 0 (collation headers) to the SMC via the `addHeader` function.

## Running an Observer Node

```
bazel run //client -- \
   --datadir /path/to/your/datadir \
   --password /path/to/your/password.txt \
   --shardid 0 \
   --networkid 12345
```

Omitting the `--actor` flag will launch a simple observer service attached to the sharding client that is able to listen to changes happening throughout the sharded Ethereum network on shard 0.

# Making Changes

## Rebuilding the Sharding Manager Contract Bindings

The Sharding Manager Contract is built in Solidity and deployed to a running geth node upon launch of the sharding node if it does not exist in the network at a specified address. If there are any changes to the SMC's code, the Golang bindigs must be rebuilt with the following command.

    go generate github.com/prysmaticlabs/prysm/contracts
    # OR
    cd contracts && go generate

# Testing

To run the unit tests of our system do:

```
$ bazel test //...
```

To run our linter, make sure you have [gometalinter](https://github.com/alecthomas/gometalinter) installed and then run

```
$ gometalinter ./...
```

# Contributing

We have put all of our contribution guidelines into [CONTRIBUTING.md](https://github.com/prysmaticlabs/prysm/blob/master/client/CONTRIBUTING.md)! Check it out to get started.

![nyancat](https://encrypted-tbn0.gstatic.com/images?q=tbn:ANd9GcRBSus2ozk_HuGdHMHKWjb1W5CmwwoxmYIjIBmERE1u-WeONpJJXg)

# License

The go-ethereum library is licensed under the
[GNU Lesser General Public License v3.0](https://www.gnu.org/licenses/lgpl-3.0.en.html)

The go-ethereum binaries is licensed under the
[GNU General Public License v3.0](https://www.gnu.org/licenses/gpl-3.0.en.html)
