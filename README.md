# Prysmatic Labs Ethereum 2.0 Implementation

![Travis Build](https://travis-ci.org/prysmaticlabs/prysm.svg?branch=master)

This is the main repository for the beacon chain and sharding implementation for Ethereum 2.0 [Prysmatic Labs](https://prysmaticlabs.com).

Before you begin, check out our [Contribution Guidelines](#contributing) and join our active chat room on Discord or Gitter below:

[![Discord](https://user-images.githubusercontent.com/7288322/34471967-1df7808a-efbb-11e7-9088-ed0b04151291.png)](https://discord.gg/KSA7rPr)
[![Gitter](https://badges.gitter.im/Join%20Chat.svg)](https://gitter.im/prysmaticlabs/geth-sharding?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge)

Also, read our [Sharding Reference Implementation Doc](https://github.com/prysmaticlabs/prysm/blob/master/docs/SHARDING.md). This doc provides a background on the sharding implementation we follow at Prysmatic Labs.


# Table of Contents

-   [Installation](#installation)
-   [Instructions](#instructions)
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

# Instructions

## Running Ethereum 2.0, v0.0.0 Release

To run our current release, v0.0.0, as a local demo, first build both parts of our system: a beacon chain node implementation, and a validator client.

```
bazel build //beacon-chain:beacon-chain
bazel build //validator:validator
```

As part of our current release v0.0.0, we allow users to start a beacon chain from genesis, connect as a validator client through a public key, and propose/vote on beacon blocks during each cycle. For more information on the full scope of the public demo, see the demo information [here](https://github.com/prysmaticlabs/prysm/blob/master/docs/DEMO_INFORMATION.md).

## Running the Beacon Node

To start the system, we need to seed the beacon chain state with an initial validator set for local development. We created a reference [genesis.json](https://github.com/prysmaticlabs/prysm/blob/master/genesis.json) you can you for this! You'll also need a special data directory where all the beacon chain data will be persisted to. Then, you can run the node as follows:

```
bazel run //beacon-chain --\
  --datadir /path/to/beacondatadir \
  --rpc-port 4000 \
  --genesis-json /path/to/genesis.json \
  --simulator \
  --demo-config

```

We added a `--simulator` flag that simulates other nodes connected to you sending your node blocks for processing. Given this is a local development version, this gives us a good idea of what the system will need to handle in the wild.

We also have a `--demo-config` flag that configures some internal parameters for you to run a local demo version of the system.

If you want to see what's happening in the system underneath the hood, add a `--verbosity debug` flag to show every single thing the beacon chain node does during its run time.

## Running a Single, ETH2.0 Validator Client

Once your beacon node is up, you'll need to attach a validator client as a separate process. This validator is in charge of running Casper+Sharding responsibilities(shards to be designed in phase 2). This validator will listen for incoming beacon blocks and shard assignments and determine when its time to perform attester/proposer responsibilities accordingly.

To get started, you'll need to use a public key from the initial validator set of the beacon node. Here are a few you can try out:

```
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
BBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB
CCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCC
```

Run as follows:

```
bazel run //validator --\
  --beacon-rpc-provider http://localhost:4000 \
  --pubkey AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
```

This will connect you to your running beacon node and listen for shard/slot assignments! The beacon node will update you at every cycle transition and shuffle your validator into different shards and slots in order to vote on or propose beacon blocks.

### Running Via Docker

To run the beacon node or validator client within a docker container, use the `//beacon-chain:image` and  `//validator:image` targets, respectively.

Example:

```text
bazel run //validator:image --\
  --beacon-rpc-provider http://localhost:4000 \
  --verbosity debug

INFO: Build options have changed, discarding analysis cache.
INFO: Analysed target //validator:image (306 packages loaded).
INFO: Found 1 target...
Target //validator:image up-to-date:
  bazel-bin/validator/image-layer.tar
INFO: Elapsed time: 8.568s, Critical Path: 0.22s
INFO: 0 processes.
INFO: Build completed successfully, 1 total action
INFO: Build completed successfully, 1 total action
37fd88e7190b: Loading layer  22.42MB/22.42MB
Loaded image ID: sha256:89b233de1a026eddeeff010fa1ef596ce791cb3f26488150aac72a91b80734c1
Tagging 89b233de1a026eddeeff010fa1ef596ce791cb3f26488150aac72a91b80734c1 as bazel/validator:image
...
```

TODO: Add [container_push](https://github.com/bazelbuild/rules_docker/#container_push-1) 
targets for the container images such that they can be pulled from GCR or 
dockerhub. 


# Testing

To run the unit tests of our system do:

```
bazel test //...
```

To run our linter, make sure you have [gometalinter](https://github.com/alecthomas/gometalinter) installed and then run

```
gometalinter ./...
```

# Contributing

We have put all of our contribution guidelines into [CONTRIBUTING.md](https://github.com/prysmaticlabs/prysm/blob/master/CONTRIBUTING.md)! Check it out to get started.

![nyancat](https://encrypted-tbn0.gstatic.com/images?q=tbn:ANd9GcRBSus2ozk_HuGdHMHKWjb1W5CmwwoxmYIjIBmERE1u-WeONpJJXg)

# License

[GNU General Public License v3.0](https://www.gnu.org/licenses/gpl-3.0.en.html)
