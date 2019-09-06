# Prysm: Ethereum 'Serenity' 2.0 Go Implementation

[![Build status](https://badge.buildkite.com/b555891daf3614bae4284dcf365b2340cefc0089839526f096.svg?branch=master)](https://buildkite.com/prysmatic-labs/prysm)
[![ETH2.0_Spec_Version 0.8.1](https://img.shields.io/badge/ETH2.0%20Spec%20Version-v0.8.1-blue.svg)](https://github.com/ethereum/eth2.0-specs/commit/452ecf8e27c7852c7854597f2b1bb4a62b80c7ec)
[![Discord](https://user-images.githubusercontent.com/7288322/34471967-1df7808a-efbb-11e7-9088-ed0b04151291.png)](https://discord.gg/KSA7rPr)
[![Gitter](https://badges.gitter.im/Join%20Chat.svg)](https://gitter.im/prysmaticlabs/geth-sharding?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge)

This is the Core repository for Prysm, [Prysmatic Labs](https://prysmaticlabs.com)' [Go](https://golang.org/) implementation of the Ethereum protocol 2.0 (Serenity).

### Need assistance?
A more detailed set of installation and usage instructions as well as explanations of each component are available on our [official documentation portal](https://prysmaticlabs.gitbook.io/prysm/). If you still have questions, feel free to stop by either our [Discord](https://discord.gg/KSA7rPr) or [Gitter](https://gitter.im/prysmaticlabs/geth-sharding?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge) and a member of the team or our community will be happy to assist you.

**Interested in what's next?** Be sure to read our [Roadmap Reference Implementation](https://github.com/prysmaticlabs/prysm/blob/master/docs/ROADMAP.md) document. This page outlines the basics of sharding as well as the various short-term milestones that we hope to achieve over the coming year.

### Come join the testnet!
Participation is now open to the public in our testnet release for Ethereum 2.0 phase 0. Visit [prylabs.net](https://prylabs.net) for more information on the project itself or to sign  up as a validator on the network.

# Table of Contents

- [Dependencies](#dependencies)
- [Installation](#installation)
    - [Build Via Docker](#build-via-docker)
    - [Build Via Bazel](#build-via-bazel)
- [Running an Ethereum 2.0 Beacon Node](#running-an-ethereum-20-beacon-node)
- [Staking ETH: Running a Validator Client](#staking-eth-running-a-validator-client)
-   [Testing Prysm](#testing-prysm)
-   [Contributing](#contributing)
-   [License](#license)

## Dependencies
Prysm can be installed either with Docker **(recommended method)** or using our build tool, Bazel. The below instructions include sections for performing both.

**For Docker installations:**
  - The latest release of [Docker](https://docs.docker.com/install/)

**For Bazel installations:**
  - The latest release of [Bazel](https://docs.bazel.build/versions/master/install.html)
  - A modern GNU/Linux operating system

## Installation

### Build via Docker
1. Ensure you are running the most recent version of Docker by issuing the command:
```
docker -v
```
2.  To pull the Prysm images from the server, issue the following commands:
```
docker pull gcr.io/prysmaticlabs/prysm/validator:latest
docker pull gcr.io/prysmaticlabs/prysm/beacon-chain:latest
```
This process will also install any related dependencies.

### Build via Bazel

1. Open a terminal window. Ensure you are running the most recent version of Bazel by issuing the command:
```
bazel version
```
2. Clone this repository and enter the directory:
```
git clone https://github.com/prysmaticlabs/prysm
cd prysm
```
3. Build both the beacon chain node implementation and the validator client:
```
bazel build //beacon-chain:beacon-chain
bazel build //validator:validator
```
Bazel will automatically pull and install any dependencies as well, including Go and necessary compilers.

## Running an Ethereum 2.0 Beacon Node
To understand the role that both the beacon node and validator play in Prysm, see [this section of our documentation](https://prysmaticlabs.gitbook.io/prysm/how-prysm-works/overview-technical).

### Running via Docker

**Docker on Linux/Mac:**
1. To start your beacon node, issue the following command:
```
docker run -v /tmp/prysm-data:/data -p 4000:4000 \
  gcr.io/prysmaticlabs/prysm/beacon-chain:latest \
  --datadir=/data
  --clear-db
```
**Docker on Windows:**

1) You will need to share the local drive you wish to mount to to container (e.g. C:).
    1. Enter Docker settings (right click the tray icon)
    2. Click 'Shared Drives'
    3. Select a drive to share
    4. Click 'Apply'

2) You will next need to create a directory named ```/tmp/prysm-data/``` within your selected shared Drive. This folder will be used as a local data directory for Beacon Node chain data as well as account and keystore information required by the validator. Docker will **not** create this directory if it does not exist already. For the purposes of these instructions, it is assumed that ```C:``` is your prior-selected shared Drive.

4) To run the beacon node, issue the following command:
```
docker run -it -v c:/tmp/prysm-data:/data -p 4000:4000 gcr.io/prysmaticlabs/prysm/beacon-chain:latest --datadir=/data --clear-db
```

### Running via Bazel

1) To start your Beacon Node with Bazel, issue the following command:
```
bazel run //beacon-chain -- --clear-db --datadir=/tmp/prysm-data
```

This will sync up the Beacon Node with the latest head block in the network.


## Staking ETH: Running a Validator Client

Once your beacon node is up, the chain will be waiting for you to deposit 3.2 Goerli ETH into the Validator Deposit Contract to activate your validator (discussed in the section below). First though, you will need to create a *validator client* to connect to this node in order to stake and participate. Each validator represents 3.2 Goerli ETH being staked in the system,  and it is possible to spin up as many as you desire in order to have more stake in the network.

### Activating Your Validator: Depositing 3.2 Goerli ETH 

Using your validator deposit data from the previous step, follow the instructions found on https://prylabs.net/participate to make a deposit.

It will take a while for the nodes in the network to process your deposit, but once your node is active, the validator will begin doing its responsibility. In your validator client, you will be able to frequently see your validator balance as it goes up over time. Note that, should your node ever go offline for a long period, you'll start gradually losing your deposit until you are removed from the system. 

### Starting the validator with Bazel

1. Open another terminal window. Enter your Prysm directory and run the validator by issuing the following command:
```
cd prysm
bazel run //validator
```
**Congratulations, you are now running Ethereum 2.0 Phase 0!**

## Testing Prysm

**To run the unit tests of our system**, issue the command:
```
bazel test //...
```

**To run our linter**, make sure you have [golangci-lint](https://github.com/golangci/golangci-lint) installed and then issue the command:
```
golangci-lint run
```

## Contributing
We have put all of our contribution guidelines into [CONTRIBUTING.md](https://github.com/prysmaticlabs/prysm/blob/master/CONTRIBUTING.md)! Check it out to get started.

## License
[GNU General Public License v3.0](https://www.gnu.org/licenses/gpl-3.0.en.html)
