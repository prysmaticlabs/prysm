# Prysm: An Ethereum 2.0 Client Written in Go

[![Build status](https://badge.buildkite.com/b555891daf3614bae4284dcf365b2340cefc0089839526f096.svg?branch=master)](https://buildkite.com/prysmatic-labs/prysm)
[![ETH2.0_Spec_Version 0.9.0](https://img.shields.io/badge/ETH2.0%20Spec%20Version-v0.9.0-blue.svg)](https://github.com/ethereum/eth2.0-specs/tree/v0.9.0)
[![Discord](https://user-images.githubusercontent.com/7288322/34471967-1df7808a-efbb-11e7-9088-ed0b04151291.png)](https://discord.gg/KSA7rPr)
[![Gitter](https://badges.gitter.im/Join%20Chat.svg)](https://gitter.im/prysmaticlabs/geth-sharding?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge)

This is the core repository for Prysm, a [Golang](https://golang.org/) implementation of the Ethereum 2.0 client specifications developed by [Prysmatic Labs](https://prysmaticlabs.com).

### Need assistance?
A more detailed set of installation and usage instructions as well as breakdowns of each individual component are available in the [official documentation portal](https://prysmaticlabs.gitbook.io/prysm/). If you still have questions, feel free to stop by either our [Discord](https://discord.gg/KSA7rPr) or [Gitter](https://gitter.im/prysmaticlabs/geth-sharding?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge) and a member of the team or our community will be happy to assist you.

### Come join the testnet!
Participation is now open to the public for our Ethereum 2.0 phase 0 testnet release. Visit [prylabs.net](https://prylabs.net) for more information on the project or to sign up as a validator on the network.

# Table of Contents

- [Dependencies](#dependencies)
- [Installation](#installation)
    - [Build via Docker](#build-via-docker)
    - [Build via Bazel](#build-via-bazel)
- [Connecting to the public testnet: running a beacon node](#connecting-to-the-testnet-running-a-beacon-node)
    - [Running via Docker](#build-via-docker)
    - [Running via Bazel](#build-via-bazel)
- [Staking ETH: running a validator client](#staking-eth-running-a-validator-client)
    - [Activating your validator: depositing 3.2 Goerli ETH](#activating-your-validator-depositing-32-goerli-eth)
    - [Starting the validator with Bazel](#starting-the-validator-with-bazel)
- [Setting up a local ETH2 development chain](#setting-up-a-local-eth2-development-chain)
    - [Installation and dependencies](#installation-and-dependencies) 
    - [Running a local beacon node and validator client](#running-a-local-beacon-node-and-validator-client)   
-   [Testing Prysm](#testing-prysm)
-   [Contributing](#contributing)
-   [License](#license)

## Dependencies
Prysm can be installed either with Docker **(recommended method)** or using our build tool, Bazel. The below instructions include sections for performing both.

**For Docker installations:**
  - The latest release of [Docker](https://docs.docker.com/install/)

**For Bazel installations:**
  - The latest release of [Bazel](https://docs.bazel.build/versions/master/install.html)
  - A modern UNIX operating system (MacOS included)

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

4. Build the configuration for the Prysm testnet by issuing the commands:

```
bazel build --define ssz=minimal //beacon-chain:beacon-chain
bazel build --define ssz=minimal //validator:validator
```

The binaries will be built in an architecture-dependent subdirectory of `bazel-bin`, and are supplied as part of Bazel's build process.  To fetch the location, issue the command:

```
$ bazel build --define ssz=minimal //beacon-chain:beacon-chain
...
Target //beacon-chain:beacon-chain up-to-date:
  bazel-bin/beacon-chain/linux_amd64_stripped/beacon-chain
...
```

In the example above, the beacon chain binary has been created in `bazel-bin/beacon-chain/linux_amd64_stripped/beacon-chain`.

## Connecting to the testnet: running a beacon node

This section contains instructions for initialising a beacon node and connecting to the public testnet. To further understand the role that both the beacon node and validator play in Prysm, see [this section of our documentation](https://prysmaticlabs.gitbook.io/prysm/how-prysm-works/overview-technical).

### Running via Docker

#### Docker on Linux/Mac

To start your beacon node, issue the following command:

```
docker run -v $HOME/prysm-data:/data -p 4000:4000 \
  --name beacon-node \
  gcr.io/prysmaticlabs/prysm/beacon-chain:latest \
  --no-genesis-delay \
  --datadir=/data
```

(Optional) If you want to enable gRPC, then run this command instead of the one above:

```
docker run -v $HOME/prysm-data:/data -p 4000:4000 -p 7000:7000 \
  --name beacon-node \
  gcr.io/prysmaticlabs/prysm/beacon-chain:latest \
  --datadir=/data \
  --no-genesis-delay \
  --grpc-gateway-port=7000
```

You can halt the beacon node using `Ctrl+c` or with the following command:

```
docker stop beacon-node
```

To restart the beacon node, issue the command:

```
docker start -ai beacon-node
```

To delete a corrupted container, issue the command:

```
docker rm beacon-node
```

To recreate a deleted container and refresh the chain database, issue the start command with an additional `--force-clear-db` parameter:

```
docker run -it -v $HOME/prysm-data:/data -p 4000:4000 --name beacon-node \
  gcr.io/prysmaticlabs/prysm/beacon-chain:latest \
  --datadir=/data \
  --force-clear-db
```

#### Docker on Windows

1) You will need to share the local drive you wish to mount to to container (e.g. C:).
    1. Enter Docker settings (right click the tray icon)
    2. Click 'Shared Drives'
    3. Select a drive to share
    4. Click 'Apply'

2) You will next need to create a directory named ```/tmp/prysm-data/``` within your selected shared Drive. This folder will be used as a local data directory for Beacon Node chain data as well as account and keystore information required by the validator. Docker will **not** create this directory if it does not exist already. For the purposes of these instructions, it is assumed that ```C:``` is your prior-selected shared Drive.

4) To run the beacon node, issue the command:
```
docker run -it -v c:/tmp/prysm-data:/data -p 4000:4000 gcr.io/prysmaticlabs/prysm/beacon-chain:latest --datadir=/data
```

### Running via Bazel

1) To start your Beacon Node with Bazel, issue the command:
```
bazel run //beacon-chain -- --datadir=/tmp/prysm-data
```
This will sync up the Beacon Node with the latest head block in the network. Note that the beacon node must be **completely synced** before attempting to initialise a validator client, otherwise the validator will not be able to complete the deposit and funds will be lost.


## Staking ETH: running a validator client

Once your beacon node is up, the chain will be waiting for you to deposit 3.2 Goerli ETH into the Validator Deposit Contract to activate your validator (discussed in the section below). First though, you will need to create a validator client to connect to this node in order to stake and participate. Each validator represents 3.2 Goerli ETH being staked in the system, and it is possible to spin up as many as you desire in order to have more stake in the network.

For more information on the functionality of validator clients, see [this section](https://prysmaticlabs.gitbook.io/prysm/how-prysm-works/validator-clients) of our official documentation.

### Activating your validator: depositing 3.2 Goerli ETH

Using your validator deposit data from the previous step, follow the instructions found on https://prylabs.net/participate to make a deposit.

It will take a while for the nodes in the network to process your deposit, but once your node is active, the validator will begin doing its responsibility. In your validator client, you will be able to frequently see your validator balance as it goes up over time. Note that, should your node ever go offline for a long period, you'll start gradually losing your deposit until you are removed from the system.

### Starting the validator with Bazel

1. Open another terminal window. Enter your Prysm directory and run the validator by issuing the following command:
```
cd prysm
bazel run //validator
```
**Congratulations, you are now running Ethereum 2.0 Phase 0!**

## Setting up a local ETH2 development chain

This section outlines the process of setting up Prysm for local interop testing with other Ethereum 2.0 client implementations. See the [INTEROP.md](https://github.com/prysmaticlabs/prysm/blob/master/INTEROP.md) file for advanced configuration options. For more background information on interoperability development, see [this blog post](https://blog.ethereum.org/2019/09/19/eth2-interop-in-review/).

### Installation and dependencies

To begin setting up a local ETH2 development chain, follow the **Bazel** instructions found in the [dependencies](#dependencies) and [installation](#installation) sections respectively. 

### Running a local beacon node and validator client

The example below will deterministically generate a beacon genesis state, initiate Prysm with 64 validators and set the genesis time to your local machines current UNIX time.

1. Open up two terminal windows. In the first, issue the command:

```
bazel run //beacon-chain -- \
--no-genesis-delay \
--bootstrap-node= \
--deposit-contract 0xD775140349E6A5D12524C6ccc3d6A1d4519D4029 \
--clear-db \
--interop-num-validators 64 \
--interop-eth1data-votes
```

2. Wait a moment for the beacon chain to start. In the other terminal, issue the command:

```
bazel run //validator -- --interop-num-validators 64
```

This command will kickstart the system with your 64 validators performing their duties accordingly.

## Testing Prysm

To run the unit tests of our system, issue the command:
```
bazel test //...
```

To run the linter, ensure you have [golangci-lint](https://github.com/golangci/golangci-lint) installed, then issue the command:
```
golangci-lint run
```

## Contributing
Want to get involved? Check out our [Contribution Guide](https://prysmaticlabs.gitbook.io/prysm/getting-involved/contribution-guidelines) to learn more!

## License
[GNU General Public License v3.0](https://www.gnu.org/licenses/gpl-3.0.en.html)
