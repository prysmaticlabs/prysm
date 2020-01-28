# Prysm: An Ethereum 2.0 Client Written in Go

[![Build status](https://badge.buildkite.com/b555891daf3614bae4284dcf365b2340cefc0089839526f096.svg?branch=master)](https://buildkite.com/prysmatic-labs/prysm)
[![ETH2.0_Spec_Version 0.9.3](https://img.shields.io/badge/ETH2.0%20Spec%20Version-v0.9.3-blue.svg)](https://github.com/ethereum/eth2.0-specs/tree/v0.9.3)
[![Discord](https://user-images.githubusercontent.com/7288322/34471967-1df7808a-efbb-11e7-9088-ed0b04151291.png)](https://discord.gg/KSA7rPr)
[![Gitter](https://badges.gitter.im/Join%20Chat.svg)](https://gitter.im/prysmaticlabs/geth-sharding?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge)

This is the core repository for Prysm, a [Golang](https://golang.org/) implementation of the Ethereum 2.0 client specifications developed by [Prysmatic Labs](https://prysmaticlabs.com).

### Need assistance?
A more detailed set of installation and usage instructions as well as breakdowns of each individual component are available in the [official documentation portal](https://docs.prylabs.network). If you still have questions, feel free to stop by either our [Discord](https://discord.gg/KSA7rPr) or [Gitter](https://gitter.im/prysmaticlabs/geth-sharding?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge) and a member of the team or our community will be happy to assist you.

### Come join the testnet!
Participation is now open to the public for our Ethereum 2.0 phase 0 testnet release. Visit [prylabs.net](https://prylabs.net) for more information on the project or to sign up as a validator on the network.

# Table of Contents

- [Dependencies](#dependencies)
- [Installation](#installing-prysm)
    - [Build via Docker](#build-via-docker)
    - [Build via Bazel](#build-via-bazel)
- [Connecting to the public testnet: running a beacon node](#connecting-to-the-testnet-running-a-beacon-node)
    - [Running via Docker](#running-via-docker)
    - [Running via Bazel](#running-via-bazel)
- [Staking ETH: running a validator client](#staking-eth-running-a-validator-client)
    - [Activating your validator: depositing 3.2 Goerli ETH](#activating-your-validator-depositing-32-gerli-eth)
    - [Starting the validator with Bazel](#starting-the-validator-with-bazel)
- [Setting up a local ETH2 development chain](#setting-up-a-local-eth2-development-chain)
    - [Installation and dependencies](#installation-and-dependencies) 
    - [Running a local beacon node and validator client](#running-a-local-beacon-node-and-validator-client)   
-   [Testing Prysm](#testing-prysm)
-   [Contributing](#contributing)
-   [License](#license)

## Dependencies

Prysm can be installed either with Docker **\(recommended\)** or using our build tool, Bazel. The below instructions include sections for performing both.

#### **For Docker installations:**

* The latest release of [Docker](https://docs.docker.com/install/)

#### **For Bazel installations:**

* The latest release of [Bazel](https://docs.bazel.build/versions/master/install.html)
* The latest release of `cmake`
* The latest release of `git`
* A modern UNIX operating system \(macOS included\)

## Installing Prysm

### Build via Docker

1. Ensure you are running the most recent version of Docker by issuing the command:

```text
docker -v
```

2. To pull the Prysm images, issue the following commands:

```text
docker pull gcr.io/prysmaticlabs/prysm/validator:latest
docker pull gcr.io/prysmaticlabs/prysm/beacon-chain:latest
```

This process will also install any related dependencies.

### Build via Bazel

1. Open a terminal window. Ensure you are running the most recent version of Bazel by issuing the command:

```text
bazel version
```

2. Clone Prysm's [main repository](https://github.com/prysmaticlabs/prysm) and enter the directory:

```text
git clone https://github.com/prysmaticlabs/prysm
cd prysm
```

3. Build both the beacon chain node and the validator client:

```text
bazel build //beacon-chain:beacon-chain
bazel build //validator:validator
```

Bazel will automatically pull and install any dependencies as well, including Go and necessary compilers.

## Connecting to the testnet: running a beacon node

Below are instructions for initialising a beacon node and connecting to the public testnet. To further understand the role that the beacon node plays in Prysm, see [this section of the documentation.](https://docs.prylabs.network/docs/how-prysm-works/architecture-overview/)


**NOTE:** It is recommended to open up port 13000 on your local router to improve connectivity and receive more peers from the network. To do so, navigate to `192.168.0.1` in your browser and login if required. Follow along with the interface to modify your routers firewall settings. When this task is completed, append the parameter`--p2p-host-ip=$(curl -s ident.me)` to your selected beacon startup command presented in this section to use the newly opened port.

### Running via Docker

#### **Docker on Linux/macOS:**

To start your beacon node, issue the following command:

```text
docker run -it -v $HOME/prysm:/data -p 4000:4000 -p 13000:13000 --name beacon-node \
  gcr.io/prysmaticlabs/prysm/beacon-chain:latest \
  --datadir=/data
```

The beacon node can be halted by either using `Ctrl+c` or with the command:

```text
docker stop beacon-node
```

To restart the beacon node, issue the following command:

```text
docker start -ai beacon-node
```

To delete a corrupted container, issue the following command:

```text
docker rm beacon-node
```

To recreate a deleted container and refresh the chain database, issue the start command with an additional `--clear-db` parameter:

```text
docker run -it -v $HOME/prysm:/data -p 4000:4000 -p 13000:13000 --name beacon-node \
  gcr.io/prysmaticlabs/prysm/beacon-chain:latest \
  --datadir=/data \
  --clear-db
```

#### **Docker on Windows:**

1. You will need to 'share' the local drive you wish to mount to \(e.g. C:\).
   1. Enter Docker settings \(right click the tray icon\)
   2. Click 'Shared Drives'
   3. Select a drive to share
   4. Click 'Apply'
   
2. You will next need to create a directory named `/prysm/` within your selected shared Drive. This folder will be used as a local data directory for Beacon Node chain data as well as account and keystore information required by the validator. Docker will **not** create this directory if it does not exist already. For the purposes of these instructions, it is assumed that `C:` is your prior-selected shared Drive.
3. To run the beacon node, issue the following command:

```text
docker run -it -v c:/prysm/:/data -p 4000:4000 -p 13000:13000 --name beacon-node gcr.io/prysmaticlabs/prysm/beacon-chain:latest --datadir=/data --clear-db
```

### Running via Bazel

To start your Beacon Node with Bazel, issue the following command:

```text
bazel run //beacon-chain -- --clear-db --datadir=$HOME/prysm
```

This will sync up the beacon node with the latest head block in the network.


**NOTE:** The beacon node must be **completely synced** before attempting to initialise a validator client, otherwise the validator will not be able to complete the deposit and **funds will lost**.


## Staking ETH: Running a validator client

Once your beacon node is up, the chain will be waiting for you to deposit 3.2 Goerli ETH into a [validator deposit contract](https://docs.prylabs.network/docs/how-prysm-works/validator-deposit-contract) in order to activate your validator \(discussed in the section below\). First though, you will need to create this validator and connect to this node to participate in consensus.

Each validator represents 3.2 Goerli ETH being staked in the system, and it is possible to spin up as many as you desire in order to have more stake in the network.

### Activating your validator: depositing 3.2 Göerli ETH

To begin setting up a validator, follow the instructions found on [prylabs.net](https://prylabs.net) to use the Göerli ETH faucet and make a deposit. For step-by-step assistance with the deposit page, see the [Activating a Validator ](https://docs.prylabs.network/docs/activating-a-validator)section of this documentation.

It will take a while for the nodes in the network to process a deposit. Once the node is active, the validator will immediately begin performing its responsibilities.

In your validator client, you will be able to frequently see your validator balance as it goes up over time. Note that, should your node ever go offline for a long period, a validator will start gradually losing its deposit until it is removed from the network entirely.

**Congratulations, you are now running Ethereum 2.0 Phase 0!**

## Setting up a local ETH2 development chain

This section outlines the process of setting up Prysm for local testing with other Ethereum 2.0 client implementations. See the [INTEROP.md](https://github.com/prysmaticlabs/prysm/blob/master/INTEROP.md) file for advanced configuration options. For more background information on interoperability development, see [this blog post](https://blog.ethereum.org/2019/09/19/eth2-interop-in-review/).

### Installation and dependencies

To begin setting up a local ETH2 development chain, follow the **Bazel** instructions found in the [dependencies](https://github.com/prysmaticlabs/prysm#dependencies) and [installation](https://github.com/prysmaticlabs/prysm#installation) sections respectively.

### Running a local beacon node and validator client

The example below will generate a beacon genesis state and initiate Prysm with 64 validators with the genesis time set to your machines UNIX time.

Open up two terminal windows. In the first, issue the command:

```text
bazel run //beacon-chain -- \
--custom-genesis-delay=0 \
--bootstrap-node= \
--deposit-contract $(curl https://prylabs.net/contract) \
--clear-db \
--interop-num-validators 64 \
--interop-eth1data-votes
```

Wait a moment for the beacon chain to start. In the other terminal, issue the command:

```text
bazel run //validator -- --keymanager=interop --keymanageropts='{"keys":64}'
```

This command will kickstart the system with your 64 validators performing their duties accordingly.

## Testing Prysm

To run the unit tests of our system, issue the command:

```text
bazel test //...
```

To run our linter, make sure you have [golangci-lint](https://github.com/golangci/golangci-lint) installed and then issue the command:

```text
golangci-lint run
```


## Contributing
Want to get involved? Check out our [Contribution Guide](https://docs.prylabs.network/docs/contribute/contribution-guidelines/) to learn more!

## License
[GNU General Public License v3.0](https://www.gnu.org/licenses/gpl-3.0.en.html)
