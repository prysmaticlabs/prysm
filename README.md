Prysmatic Labs Sharding Implementation
==========================
This is the main repository for the sharding implementation of the go-ethereum client by [Prysmatic Labs](https://prysmaticlabs.com). For the original, go-ethereum project, refer to the following [link](https://github.com/ethereum/go-ethereum).

Before you begin, check out our [Sharding Reference Implementation Doc](https://github.com/prysmaticlabs/geth-sharding/blob/master/sharding/README.md). This doc serves as the single source of truth for our team, our milestones, and details on the different components of our architecture.

Interested in contributing? Check out our [Contribution Guidelines](#contribution-guidelines) and join our active chat room on Gitter below:

[![Gitter](https://badges.gitter.im/Join%20Chat.svg)](https://gitter.im/prysmaticlabs/geth-sharding?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge)

Table of Contents
=================

- [Installation](#installation)
- [Sharding Instructions](#sharding)
  - [Running a Local Geth Node](#running-a-local-geth-node)
  - [Transaction Generator](#transaction-generator)
  - [Becoming a Validator](#becoming-a-validator)
  - [Becoming a Proposer](#becoming-a-proposer)
- [Testing](#testing)
- [Contribution Guidelines](#contribution-guidelines)
- [License](#license)

Installation
============

Create a folder in your `$GOPATH` and navigate to it

```
mkdir -p $GOPATH/src/github.com/ethereum && cd $GOPATH/src/github.com/ethereum
```

Clone our repository as `go-ethereum`

```
git clone https://github.com/prysmaticlabs/geth-sharding ./go-ethereum
```

For prerequisites and detailed build instructions please read the
[Installation Instructions](https://github.com/ethereum/go-ethereum/wiki/Building-Ethereum)
on the wiki.

Building geth requires both a Go (version 1.7 or later) and a C compiler.
You can install them using your favourite package manager.
Once the dependencies are installed, run

```
make geth
```

or, to build the full suite of utilities:

```
make all
```

Sharding Instructions
==============

To get started with running the project, follow the instructions to initialize your own private Ethereum blockchain and geth node, as they will be required to run before you can become a validator or a proposer.

Running a Local Geth Node
------------------------------

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
    "gasLimit": "210000000000"
}
```

Then, you can build `geth` and init a new instance of a local, Ethereum blockchain as follows:

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

Now, save the passphrase you used in the geth node into a text file called password.txt. Then, once you have this private geth node running on your local network, we will need to generate fake, pending transactions that can then be processed into shards by validators and proposers. For this, we have created an in-house transaction generator CLI tool.

Transaction Generator
---------------------

Work in Progress. To track our current draft of the tx generator cli spec, visit this [link](https://docs.google.com/document/d/1YohsW4R9dIRo0u5RqfNOYjCkYKVCmzjgoBDBYDdu5m0/edit?usp=drive_web&ouid=105756662967435769870).

Once we have fake transactions broadcast to our local node, we can start a validator and proposer client in separate terminal windows to begin the sharding process.

Becoming a Validator
-----------------------
To deposit ETH and join as a validator in the Validator Manager Contract, run the following command:

```
geth sharding-validator --deposit 100eth --password /path/to/your/password.txt
```

This will extract 100ETH from your account balance and insert you into the VMC's validator set. Then, the program will listen for incoming block headers and notify you when you have been selected as an eligible proposer for a certain shard in a given period. Once you are selected, the validator will request collations from a "collation proposals pool" that is created by a proposer node. We will need to run a proposal node concurrently in a separate terminal window as follows:

Becoming a Proposer
-----------------------
The proposer node can be started with the following command:

```
geth sharding-proposer --password /path/to/your/password.txt
```

Proposers are tasked with state execution, so they will process and validate pending transactions in the Geth node and create collations with headers that are then broadcast to a proposals pool along with an ETH deposit.

Validators then subscribe to changes in the proposals pool and fetch the collation headers that offer the highest ETH deposit. Once a validator signs this collation, the proposer needs to provide the full collation body and the validator can then append the collation header to the Validator Manager Contract.

Once this is done, the full, end-to-end sharding example is complete and another iteration can occur.


Making Changes
==============

Rebuilding the Validator Manager Contract Bindings
---------------------------------------------------------
The Validator Manager Contract is built in Solidity and deployed to the geth node upon launch of the client if it does not exist in the network at a specified address. If there are any changes to the VMC's code, the Golang bindigs must be rebuilt with the following command.

```
go generate abigen --sol contracts/validator_manager.sol --pkg contracts --out contracts/validator_manager.go
```

Testing
=======

To run the unit tests of our system do:

```
go test ./sharding
```

We will require more complex testing scenarios (fuzz tests) to measure the full integrity of the system as it evolves.

Contribution Guidelines
===============

Excited by our work and want to get involved in building out our sharding releases? Our  [Sharding Reference Implementation Doc](https://github.com/prysmaticlabs/geth-sharding/blob/master/sharding/README.md) has all you need to know in order to begin helping us make this happen. We created this document as a single source of reference for all things related to sharding Ethereum, and we need as much help as we can get!

You can explore our [Current Projects](https://github.com/prysmaticlabs/geth-sharding/projects) in-the works for the Ruby release. Each of the project boards contain a full collection of open and closed issues relevant to the different parts of our first implementation that we use to track our open source progress. Feel free to fork our repo and start creating PR’s after assigning yourself to an issue of interest. We are always chatting on [Gitter](https://gitter.im/prysmaticlabs/geth-sharding), so drop us a line there if you want to get more involved or have any questions on our implementation!

**Contribution Steps**

- Create a folder in your `$GOPATH` and navigate to it `mkdir -p $GOPATH/src/github.com/ethereum && cd $GOPATH/src/github.com/ethereum`
- Clone our repository as `go-ethereum`, `git clone https://github.com/prysmaticlabs/geth-sharding ./go-ethereum`
- Fork the `go-ethereum` repository on Github: https://github.com/ethereum/go-ethereum
- Add a remote to your fork
`git remote add YOURNAME https://github.com/YOURNAME/go-ethereum

Now you should have a remote pointing to the `origin` repo (geth-sharding) and to your forked, go-ethereum repo on Github. To commit changes and start a Pull Request, our workflow is as follows:

- Create a new branch with a clear feature name such as `git checkout -b collations-pool`
- Issue changes with clear commit messages
- Push to your remote `git push YOURNAME collations-pool`
- Go to the [geth-sharding](https://github.com/prysmaticlabs/geth-sharding) repository on Github and start a PR comparing `geth-sharding:master` with `go-ethereum:collations-pool` (your fork on your profile).
- Add a clear PR title along with a description of what this PR encompasses, when it can be closed, and what you are currently working on. Github markdown checklists work great for this.

License
=====
The go-ethereum library (i.e. all code outside of the `cmd` directory) is licensed under the
[GNU Lesser General Public License v3.0](https://www.gnu.org/licenses/lgpl-3.0.en.html), also
included in our repository in the `COPYING.LESSER` file.

The go-ethereum binaries (i.e. all code inside of the `cmd` directory) is licensed under the
[GNU General Public License v3.0](https://www.gnu.org/licenses/gpl-3.0.en.html), also included
in our repository in the `COPYING` file.
