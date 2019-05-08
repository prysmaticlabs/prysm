# Prysmatic Labs Ethereum Serenity Implementation

[![Build status](https://badge.buildkite.com/b555891daf3614bae4284dcf365b2340cefc0089839526f096.svg)](https://buildkite.com/prysmatic-labs/prysm)

This is the main repository for the Go implementation of the Ethereum 2.0 Serenity [Prysmatic Labs](https://prysmaticlabs.com).

Before you begin, check out our [official documentation portal](https://prysmaticlabs.gitbook.io/prysm/) and join our active chat room on Discord or Gitter below:

[![Discord](https://user-images.githubusercontent.com/7288322/34471967-1df7808a-efbb-11e7-9088-ed0b04151291.png)](https://discord.gg/KSA7rPr)
[![Gitter](https://badges.gitter.im/Join%20Chat.svg)](https://gitter.im/prysmaticlabs/geth-sharding?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge)

Also, read our [Roadmap Reference Implementation Doc](https://github.com/prysmaticlabs/prysm/blob/master/docs/ROADMAP.md). This doc provides a background on the milestones we aim for the project to achieve.


# Table of Contents

- [Join Our Testnet](#join-our-testnet)
- [Installation](#installation)
    - [Run Via Docker](#run-via-docker-recommended)
    - [Run Via Bazel](#run-via-bazel)
  - [Prysm Main Components](#prysm-main-components)
    - [Running an Ethereum 2.0 Beacon Node](#running-an-ethereum-20-beacon-node)
    - [Staking ETH: Running a Validator Client](#staking-eth-running-a-validator-client)
-   [Testing](#testing)
-   [Contributing](#contributing)
-   [License](#license)

# Join Our Testnet

You can now participate in our public testnet release for Ethereum 2.0 phase 0. Visit [prylabs.net](https://prylabs.net) 💎 to participate!

# Installing Prysm

### Installation Options
You can either choose to run our system via:
- Our latest [release](https://github.com/prysmaticlabs/prysm/releases) **(Easiest)**
- Using Docker **(Recommended)**
- Using Our Build Tool, Bazel

### Fetching via Docker (Recommended)
Docker is a convenient way to run Prysm, as all you need to do is fetch the latest images:

```
docker pull gcr.io/prysmaticlabs/prysm/validator:latest
docker pull gcr.io/prysmaticlabs/prysm/beacon-chain:latest
```

### Build Via Bazel
First, clone our repository:

```
git clone https://github.com/prysmaticlabs/prysm
```

Download the Bazel build tool by Google here and ensure it works by typing:

```
bazel version
```

Bazel manages all of the dependencies for you (including go and necessary compilers) so you are all set to build prysm. Then, build both parts of our system: a beacon chain node implementation, and a validator client:

```
bazel build //beacon-chain:beacon-chain
bazel build //validator:validator
```

# Prysm Main Components
Prysm ships with two important components: a beacon node and a validator client. The beacon node is the server that performs the heavy lifting of Ethereum 2.0., A validator client is another piece of software that securely connects to the beacon node and allows you to stake 3.2 Goerli ETH in order to secure the network. You'll be mostly interacting with the validator client to manage your stake.
Another critical component of Ethereum 2.0 is the Validator Deposit Contract, which is a smart contract deployed on the Ethereum 1.0 chain which can be used for current holders of ETH to do a one-way transfer into Ethereum 2.0.

### Running an Ethereum 2.0 Beacon Node
With docker:

```
docker run -v /tmp/prysm-data:/data -p 4000:4000 \
  gcr.io/prysmaticlabs/prysm/beacon-chain:latest \
  --datadir=/data
  --clear-db
```

To start your beacon node with bazel:

```
bazel run //beacon-chain -- --clear-db --datadir=/tmp/prysm-data
```

This will sync you up with the latest head block in the network, and then you'll have a ready beacon node.
The chain will then be waiting for you to deposit 3.2 Goerli ETH into the Validator Deposit Contract before your validator can become active! Now, you'll need to create a validator client to connect to this node and stake 3.2 Goerli ETH to participate as a validator in Ethereum 2.0's Proof of Stake system.

### Staking ETH: Running a Validator Client
Once your beacon node is up, you'll need to attach a validator client as a separate process. Each validator represents 3.2 Goerli ETH being staked in the system, so you can spin up as many as you want to have more at stake in the network

**Activating Your Validator: Depositing 3.2 Goerli ETH**

Using your validator deposit data from the previous step, use the instructions in https://alpha.prylabs.net/participate to deposit.

It'll take a while for the nodes in the network to process your deposit, but once you're active, your validator will begin doing its responsibility! In your validator client, you'll be able to frequently see your validator balance as it goes up. If you ever go offline for a while, you'll start gradually losing your deposit until you get kicked out of the system. Congratulations, you are now running Ethereum 2.0 Phase 0 :).

# Testing

To run the unit tests of our system do:

```
bazel test //...
```

To run our linter, make sure you have [golangci-lint](https://https://github.com/golangci/golangci-lint) installed and then run:

```
golangci-lint run
```

# Contributing

We have put all of our contribution guidelines into [CONTRIBUTING.md](https://github.com/prysmaticlabs/prysm/blob/master/CONTRIBUTING.md)! Check it out to get started.

![nyancat](https://encrypted-tbn0.gstatic.com/images?q=tbn:ANd9GcRBSus2ozk_HuGdHMHKWjb1W5CmwwoxmYIjIBmERE1u-WeONpJJXg)

# License

[GNU General Public License v3.0](https://www.gnu.org/licenses/gpl-3.0.en.html)
