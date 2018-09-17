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
        "826f3F66dB0416ea82033aE917A611bfBF4D98b6": { "balance": "300000" }
    }
}
```

The `alloc` portion specifies account addresses with prefunded ETH when the Ethereum blockchain is created. You can modify this section of the genesis to include your own test address and prefund it with 100ETH.

Then, you can build and init a new instance of a local, Ethereum blockchain as follows:

```
geth init /path/to/genesis.json --datadir /path/to/your/datadir
geth --nodiscover console --datadir /path/to/your/datadir --networkid 12345 --ws --wsaddr=127.0.0.1 --wsport 8546 --wsorigins "*" --rpc
````

It is **important** to note that the `--networkid` flag must match the `chainId` property in the genesis file.

Then, the geth console can start up and you can start a miner as follows:

    > personal.newAccount()
    > miner.setEtherbase(eth.accounts[0])
    > miner.start(1)

Now, save the passphrase you used in the geth node into a text file called password.txt. Then, once you have this private geth node running on your local network, we will need to generate test, pending transactions that can then be processed into collations by proposers. For this, we have created an in-house transaction generator CLI tool.


# Running Ethereum 2.0

**NOTE**: This section is in flux, much of this will likely change as the beacon chain spec evolves.

Build our system first

```
bazel build //beacon-chain:beacon-chain
bazel build //validator:validator
```

## Step 1: Deploy a Validator Registation Contract

Deploy the Validator Registration Contract into the chain of the running geth node by following the instructions [here](https://github.com/prysmaticlabs/prysm/blob/master/contracts/validator-registration-contract/deployVRC/README.md).

## Step 2a: Running a Beacon Node as a Validator or Observer

Make sure a geth node is running as a separate process according to the instructions from the previous section. Then, you can run a full beacon node as follows:

```
bazel run //beacon-chain --\
  --web3provider  ws://127.0.0.1:8546 \
  --datadir /path/to/your/datadir \
  --rpc-port 4000 \
  --validator
```

This will spin up a full beacon node that connects to your running geth node, opens up an RPC connection for sharding validators to connect to it, and begins listening for p2p events.

To try out the beacon node in development by simulating incoming blocks, run the same command above but enable the `--simulator` and a debug level, log verbosity with `--verbosity debug` to see everything happening underneath the hood.

```
bazel run //beacon-chain --\
  --web3provider  ws://127.0.0.1:8546 \
  --datadir /path/to/your/datadir \
  --rpc-port 4000 \
  --validator \
  --simulator \
  --verbosity debug
```

Now, deposit ETH to become a validator in the contract using instructions [here](https://github.com/prysmaticlabs/prysm/blob/master/docs/VALIDATOR_REGISTRATION.md)

If you don't want to deposit ETH and become a validator, one option is to run a beacon node as an Observer. A beacon observer node has full privilege to listen in beacon chain and shard chains activities, but it will not participate
in validator duties such as proposing or attesting blocks. In addition, an observer node doesn't need to deposit 32ETH. To run an observer node, you discard the `--validator` flag.

```
bazel run //beacon-chain --\
  --datadir /path/to/your/datadir \
  --rpc-port 4000 \
```

### Running via Docker

To run the beacon node within a docker container, use the `//beacon-chain:image` target.

```text
bazel run //beacon-chain:image --\
  --web3provider  ws://127.0.0.1:8546 \
  --datadir /path/to/your/datadir \
  --rpc-port 4000 \
  --simulator \
  --verbosity debug
```

## Step 3: Running a Beacon/Sharding validator

Once your beacon node is up, you'll need to attach a validator as a separate process. This validator is in charge of running attester/proposer responsibilities and processing shard cross links (shards to be designed in phase 2). This validator will listen for incoming beacon blocks and crystallized states and determine when its time to perform attester/proposer responsibilities accordingly.

Run as follows:

```
bazel run //validator --\
  --beacon-rpc-provider http://localhost:4000 \
  --verbosity debug
```

Then, the beacon node will update this validator with new blocks + crystallized states in order for the validator to act as an attester or proposer.

### Running via Docker

To run the validator within a docker container, use the `//validator:image` target.

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
