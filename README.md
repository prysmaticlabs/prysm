# Prysmatic Labs Ethereum Serenity Implementation

[![Build status](https://badge.buildkite.com/b555891daf3614bae4284dcf365b2340cefc0089839526f096.svg)](https://buildkite.com/prysmatic-labs/prysm)

This is the main repository for the Go implementation of the Ethereum 2.0 Serenity [Prysmatic Labs](https://prysmaticlabs.com).

Before you begin, check out our [Contribution Guidelines](#contributing) and join our active chat room on Discord or Gitter below:

[![Discord](https://user-images.githubusercontent.com/7288322/34471967-1df7808a-efbb-11e7-9088-ed0b04151291.png)](https://discord.gg/KSA7rPr)
[![Gitter](https://badges.gitter.im/Join%20Chat.svg)](https://gitter.im/prysmaticlabs/geth-sharding?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge)

Also, read our [Roadmap Reference Implementation Doc](https://github.com/prysmaticlabs/prysm/blob/master/docs/ROADMAP.md). This doc provides a background on the milestones we aim for the project to achieve.


# Table of Contents

 - [Installation](#installation)
    - [Run Via the Go Tool](#run-via-the-go-tool)
    - [Run Via Bazel (Recommended)](#run-via-bazel-recommended)
    - [Running The Beacon Chain](#running-the-beacon-chain)
    - [Running an ETH2.0 Validator Client](#running-an-eth20-validator-client)
    - [Running Via Docker](#running-via-docker)
    - [Running Under Windows](#running-under-windows)
-   [Testing](#testing)
-   [Contributing](#contributing)
-   [License](#license)

## Installation

You can either choose to run our system via:

- Use Our Build Tool, Bazel **(Recommended)**
- The Go tool to manage builds and tests
- Use Docker

## Run Via Bazel (Recommended)

First, clone our repository:

```
git clone https://github.com/prysmaticlabs/prysm
```

Download the Bazel build tool by Google [here](https://docs.bazel.build/versions/master/install.html) and ensure it works by typing:

```
bazel version
```

Bazel manages all of the dependencies for you (including go and necessary compilers) so you are all set to build prysm.


### Building

Then, build both parts of our system: a beacon chain node, and a validator client which attaches to it:

```
bazel build //beacon-chain:beacon-chain
bazel build //validator:validator
```

## Running The Beacon Chain

To start your beacon node with bazel:

```
bazel run //beacon-chain \ 
  --web3provider wss://goerli.prylabs.net/websocket \
  --deposit-contract DEPOSIT_CONTRACT_ADDRESS
```

If you want to see what's happening in the system underneath the hood, add a `--verbosity debug` flag to show every single thing the beacon chain node does during its run time.

## Running an ETH2.0 Validator Client

Once your beacon node is up, you'll need to attach a validator client as a separate process. Each validator represents 32ETH being staked in the system, so you can spin up as many as you want to have more at stake in the network.

To get started, you'll need to create a new validator account with a unique password and data directory to store your encrypted private keys:

```
bazel run //validator --\
  accounts create --password YOUR_PASSWORD \
  --keystore-path /path/to/validator/keystore/dir
```

This will connect you to your running beacon node and listen for validator assignments! The beacon node will update you at every cycle transition and shuffle your validator into different shards and slots in order to vote on or propose beacon blocks. To then run the validator, use the command: 

```
bazel run //validator --\
  --password YOUR_PASSWORD \
  --keystore-path /path/to/validator/keystore/dir
```

## Running Via Docker

```
docker run -p 4000:4000 -v gcr.io/prysmaticlabs/prysm/beacon-chain:latest \
  --rpc-port 4000 \
  --web3provider wss://goerli.prylabs.net/websocket \
  --deposit-contract DEPOSIT_CONTRACT_ADDRESS
```

Then, to run a validator client, use:

```
docker run gcr.io/prysmaticlabs/prysm/validator:latest \
  --beacon-rpc-provider http://{YOUR_LOCAL_IP}:4000 \
  --password YOUR_PASSWORD \
  --keystore-path /path/to/your/validator/keystore/dir
```

This will connect you to your running beacon node and listen for shard/slot assignments! The beacon node will update you at every cycle transition and shuffle your validator into different shards and slots in order to vote on or propose beacon blocks.

## Running Under Windows

The best way to run under Windows is to clone the repository and then run the node with go run from the Windows command line. Go 1.10 fails due to documented permission errors so be sure you are running Go 1.11 or later. Go through the source code and resolve any dependencies. Create two empty files for use as data directories by the beacon chain and validator respectively. The contents of these files should be deleted each time you run the software. After cloning the Prsym repository, run the node as follows:

```
go run ./beacon-chain main.go \
  --web3provider wss://goerli.prylabs.net/websocket \
  --deposit-contract DEPOSIT_CONTRACT_ADDRESS
```

After the beacon chain is up and running, run the validator client as a separate process as follows:

```
go run ./validator/main.go \
  --password YOUR_PASSWORD \
  --keystore-path /path/to/your/validator/keystore/dir
```

# Testing

To run the unit tests of our system do:

```
bazel test //...
```

To run our linter, make sure you have [gometalinter](https://github.com/alecthomas/gometalinter) installed and then run:

```
gometalinter ./...
```

# Contributing

We have put all of our contribution guidelines into [CONTRIBUTING.md](https://github.com/prysmaticlabs/prysm/blob/master/CONTRIBUTING.md)! Check it out to get started.

![nyancat](https://encrypted-tbn0.gstatic.com/images?q=tbn:ANd9GcRBSus2ozk_HuGdHMHKWjb1W5CmwwoxmYIjIBmERE1u-WeONpJJXg)

# License

[GNU General Public License v3.0](https://www.gnu.org/licenses/gpl-3.0.en.html)
