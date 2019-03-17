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

You can prepare our entire repo for a local build by simply running:

```
bazel build //...
```

## Deploying a Validator Deposit Contract

If you want to run our system locally, you'll need to have a **Validator Deposit Contract** deployed on the Goerli Ethereum 1.0 [testnet](https://github.com/goerli/testnet). You'll need to acquire some Goerli ETH first and then you can easily deploy our deposit contract with Bazel: 

```
bazel run //contracts/deposit-contract/deployContract -- --httpPath=https://goerli.prylabs.net 
 --privKey=${YOUR_GOERLI_PRIV_KEY} 
```

You'll see the deposit contract address printed out, which we'll use when running our Eth 2.0 beacon nodes below:

```bash
INFO: Build completed successfully, 1 total action
[2019-03-17 11:54:27]  INFO main: New contract deployed address=0x1be4cbd38AC5b68727dCD2B73fc0553c1832ca42
```

Copy the address, you'll need it in the next step:

## Running The Beacon Chain

To start your beacon node with bazel:

```
bazel run //beacon-chain \ 
  --deposit-contract DEPOSIT_CONTRACT_ADDRESS
```

The chain will then be waiting for the **Validator Deposit Contract** to reach a deposit threshold before it begins! Now, you'll need to spin up enough validator clients that can reach the threshold with the following next steps:

## Running an ETH2.0 Validator Client

Once your beacon node is up, you'll need to attach a validator client as a separate process. Each validator represents 32ETH being staked in the system, so you can spin up as many as you want to have more at stake in the network.

To get started, you'll need to create a new validator account with a unique password and data directory to store your encrypted private keys:

```
bazel run //validator --\
  accounts create --password YOUR_PASSWORD \
  --keystore-path /path/to/validator/keystore/dir
```

Once you create your validator account, you'll see a special piece of information printed out below:

```bash
[2019-03-17 11:57:55]  INFO accounts: Account creation complete! Copy and paste the deposit data shown below when issuing a transaction into the ETH1.0 deposit contract to activate your validator client

========================Deposit Data=======================

0xbc00000060000000814eb687f39e9a0be79552cd51711dfb1711274a892e4ccae1e61d0bb28ef82c85e81b68b4911f73ca06e6694133c9610a6677d512df7a6a0289ecd1a218a8b2de29fa298c24c9e17dac4d7fb268992e8d08d74fafa076757d28ffa29ea7a36b30000000996e3494661110bf9c72f1bacee84d1b64039092bf1e6856eaf8d87b1a992999d4bff9c3701ded7714e8421c8ec1fd4520000000001e1ba7155f64eda1d1a87f3ed4eaf0280b86bef90b2cd1d9b905e370650251

===========================================================
```

Keep this deposit data string, as you'll need it in the next section.

```
bazel run //validator --\
  --password YOUR_PASSWORD \
  --keystore-path /path/to/validator/keystore/dir
```

This will connect you to your running beacon node and listen for validator assignments! The beacon node will update you at every cycle transition and shuffle your validator into different shards and slots in order to vote on or propose beacon blocks. To then run the validator, use the command: 

Given this is a local network, you'll need to create and run enough validators to reach the deposit contract threshold so your chain can begin.

## Depositing 32ETH and Starting the Eth 2.0 Network

Once you launch your beacon chain and validators, your nodes won't do much and will simply listen for the deposit contract to reach a valid threshold. You'll need an Eth 1.0 wallet that can send transactions via the Goerli network such as Metamask loaded with enough Ether to make deposits into the contract. Using the deposit contract address from earlier and your validator deposit data from the previous step, send a transactions with the validator deposit data as the `data` parameter in your web3 provider such as Metamask to kick things off. Once you make enough deposits, your beacon node will start and your system will begin running the phase 0 beacon chain!

## Running Via Docker

```
docker run -p 4000:4000 -v gcr.io/prysmaticlabs/prysm/beacon-chain:latest \
  --rpc-port 4000 \
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
