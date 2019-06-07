# Prysmatic Labs Roadmap Reference

This document serves as a main reference for Prysmatic Labs' sharding and beacon chain implementation in Go, along with our roadmap and compilation of active research.

# Table of Contents

-   [Sharding Introduction](#sharding-introduction)
    -   [Basic Sharding Idea and Design](#basic-sharding-idea-and-design)
-   [Roadmap Phases](#roadmap-phases)
    -   [The Ruby Release: Local Network](#the-ruby-release-local-network)
    -   [The Sapphire Release: Goerli Testnet](#the-sapphire-release-goerli-testnet)
    -   [The Diamond Release: Ethereum Mainnet](#the-diamond-release-ethereum-mainnet)
    -   [System Architecture](#system-architecture)
-   [Acknowledgements](#acknowledgements)
-   [References](#references)

# Sharding Introduction

Currently, every single node running the Ethereum network has to process every single transaction that goes through the network. This gives the blockchain a high amount of security because of how much validation goes into each block, but at the same time it means that an entire blockchain is only as fast as its individual nodes and not the sum of its parts. Currently, transactions on the EVM are non-parallelizable, and every transaction is executed in sequence globally. The scalability problem then has to do with the idea that a blockchain can have at most 2 of these 3 properties: decentralization, security, and scalability.

If we have scalability and security, it would mean that our blockchain is centralized and that would allow it to have a faster throughput. Right now, Ethereum is decentralized and secure, but not scalable.

An approach to solving the scalability trilemma is the idea of blockchain sharding, where we split the entire state of the network into partitions called shards that contain their own independent piece of state and transaction history. In this system, certain nodes would process transactions only for certain shards, allowing the throughput of transactions processed in total across all shards to be much higher than having a single shard do all the work as the main chain does now.

## Basic Sharding Idea and Design

A sharded blockchain system is made possible by having nodes store “signed metadata” in the main chain of latest changes within each shard chain. Through this, we manage to create a layer of abstraction that tells us enough information about the global, synced state of parallel shard chains. These messages are called **cross-links**, which are specific structures that encompass important information about blocks in shards across the network. We coordinate these crosslinks on a chain known as a **beacon-chain**. In this chain, blocks are created by actors known as **proposers** that are tasked with information about cross-links together. These blocks are then voted on by a party of actors known as **attesters**. These attesters are randomly selected for particular periods of time in certain shards and are then tasked into reaching consensus on these chains and on beacon chain blocks via a **proof of stake** system. Attesters and proposers are the names of two different responsibilities a participant in Ethereum 2.0 consensus can have. These participants are known in general as **validators**.

Cross-links are stored in blocks on a full proof of stake chain known as a **beacon chain**, which will be implemented as a sidechain to the Ethereum main chain initially.

We still keep the Ethereum main chain and deploy a smart contract into it known as the **Validator Deposit Contract**, where users can deposit and burn 32 ETH. Beacon chain nodes would listen to deposits in this contract and consequently queue up a user with the associated address as a validator in the beacon chain PoS system. Validators then become part of a registered validator set in the beacon chain, and are committees of validators are selected to become attesters or proposers for a certain period of time on shards until they are ventually reshuffled into different shards.

Given that we are splitting up the global state of the Ethereum blockchain into shards, new types of attacks arise because fewer resources are required to completely dominate a shard. This is why a **source of randomness** and periods are critical components to ensuring the integrity of the system.

The Ethereum Wiki’s [Sharding FAQ](https://github.com/ethereum/wiki/wiki/Sharding-FAQ) suggests pseudorandom sampling of validators on each shard. The goal is so that these attesters will not know which shard they will get in advance. Otherwise, malicious actors could concentrate resources into a single shard and try to overtake it (See: [1% Attack](https://medium.com/@icebearhww/ethereum-sharding-and-finality-65248951f649)).

Sharding revolves around being able to store shard metadata in a full proof of stake chain known as a beacon chain. For pseudorandomness generation, a mechanism known as RANDAO can be used in the beacon chain to shuffle validators securely.

# Roadmap Phases

Prysmatic Labs will implement the official beacon chain specification and the full roadmap for Ethereum 2.0 as written in its official repository [here](https://github.com/ethereum/eth2.0-specs) by the community of core researchers and developers in Ethereum.

To roll out these phases, we will be releasing our implementation in a series of steps:

## The Ruby Release: Local Network

Our current work is focused on creating a localized version of a beacon chain that would include the following:

-  A minimal, **beacon chain node** that will interact with an Ethereum 1.0 node via JSON-RPC
-  A **Validator Deposit Contract** deployed on an Ethereum 1.0 chain where a beacon node can read logs to check for registered validators
-  A minimal, floodsub p2p network
-  Ability for proposers/attesters to be selected by the beacon chain's randomness into committees that work on specific shards
-  Ability to execute state transitions, apply a fork choice rule, and advance a full proof of stake blockchain through distributed consensus

We released our Ruby release on October 2018

## The Sapphire Release: Goerli Testnet

Part 1 of the **Sapphire Release** will focus around getting the **Ruby Release** polished enough to be live on an Ethereum testnet and manage a a beacon chain system. 

Part 2 of the **Sapphire Release** will focus on implementing state execution and defining the State Transition Function for sharding on a local testnet (as outlined in [Beyond Phase 1](#beyond-phase-1)) as an extenstion to the Ruby Release.

ETA: End of Q1 2019

## The Diamond Release: Ethereum Mainnet

The **Diamond Release** will reconcile the best parts of the previous releases and deploy a full-featured, cross-shard transaction system through a beacon chain, casper FFG-enabled, sharding release. As expected, this is the most difficult and time consuming release on the horizon for Prysmatic Labs. We plan on growing our community effort significantly over the first few releases to get all hands-on deck preparing for this.

The Diamond Release should be considered the production release candidate for sharding Ethereum on the mainnet.

ETA: To Be determined

# Beacon Chain Phase 0 Implementation

Prysmatic Labs will begin by focusing its implementation entirely on Phase 0 from the Ethereum 2.0 roadmap. We plan on being as pragmatic as possible to create something that can be locally run by any developer as soon as possible. Our initial deliverable will center around a command line tool that will serve as an entry point into a beacon chain node that allows for users to become validators and earn rewrads for participating in Proof of Stake consensus.

Here is a reference spec explaining how our initial system will function:

## System Architecture

Our implementation revolves around the following core components:

-   A **beacon chain** that connects to this main chain node via JSON-RPC
-   A **validator client** that connects to a beacon node and allows for users to earn rewards for staking 32ETH to secure the protocol

A basic, end-to-end example of the system is as follows:

1.  _**User deposits 32 ETH into a Validator Deposit Contract on the Eth 1.0 chain:**_ the beacon chain listens for the logs in the chain to queue that validator into the beacon chain chain's main event loop

2.  _**Registered validator begins PoS process to propose blocks:**_ the PoS validator has the resposibility to participate in the addition of new blocks to the beacon chain

3.  _**RANDAO mechanism selects committees of proposers/attesters for shards:**_ the beacon chain node will use its RANDAO mechanism to select committees of proposers, attesters that each have responsibilities within the system. 

4. _**Beacon Chain State Advances, Committees are Reshuffled:**_ upon completing responsibilities, the different actors of the system are them reshuffled into new committees on different shards.

## The EVM: What You Need to Know

As an important aside, we’ll take a brief detour into the EVM and what we need to understand before we modify it for a sharded blockchain. At its core, the functionality of the EVM optimizes for _security_ and not for computational power with the following restrictions:

-   Every single step must be paid for upfront with gas costs to prevent DDoS
-   Programs can't interact with each other without a single byte array
    -   This also means programs can't access other programs' state
-   Sandboxed Execution - the EVM can only modify its internal state and nothing else
-   Deterministic execution guarantees

So what exactly is the EVM? The EVM was purposely designed to be a stack based machine with memory-byte arrays and key-value stores that are kept on a trie

-   Every single keys and storage values are 32 bytes
-   There are 100 total opcodes in the EVM
-   The EVM comes with a temporary memory byte-array and storage trie to hold persistent memory.

Cryptographic operations are done using pre-compiled contracts. Aside from that, the EVM provides a bunch of blockchain access-level context that allows certain opcodes to fetch useful information from the external system. For example, LOG opcodes store useful information in the log bloom filter that can be synced with light clients. This can be used as a low-gas form of storage, since LOG does not modify the state.

Additionally, the EVM contains a call-depth limit such that recursive invocations or chains of calls will eventually halt, preventing a drastic use of resources.

It is important to note that the merkle root of an Ethereum account is updated any time an `SSTORE` opcode is executed successfully by a program on the EVM that results in a key or value changing in the state merklix (merkle radix) tree.

How is this relevant to sharding? It is important to note the importance of certain opcodes in our implementation and how we will need to introduce and modify several of them for both security and scalability considerations in a sharded chain.

# Acknowledgements

A special thanks for entire [Prysmatic Labs](https://discord.gg/che9auJ) team for helping put this together and to Ethereum Research (Hsiao-Wei Wang, Vitalik, Justin Drake) for the help and guidance in our approach.

# References

[Sharding FAQ](https://github.com/ethereum/wiki/wiki/Sharding-FAQ)

[Sharding Reference Spec](https://github.com/ethereum/sharding/blob/develop/docs/doc.md)

[Data Availability and Erasure Coding](https://github.com/ethereum/research/wiki/A-note-on-data-availability-and-erasure-coding)

[Proof of Visibility for Data Availability](https://ethresear.ch/t/proof-of-visibility-for-data-availability/1073)

[History, State, and Asynchronous Accumulators in the Stateless Model](https://ethresear.ch/t/history-state-and-asynchronous-accumulators-in-the-stateless-model/287)

[Torus Shaped Sharding Network](https://ethresear.ch/t/torus-shaped-sharding-network/1720)

[Data Availability Proof-friendly State Tree Transitions](https://ethresear.ch/t/data-availability-proof-friendly-state-tree-transitions/1453)

[Safety Attester Pool Size](https://ethresear.ch/t/safe-attester-pool-size/1728)

[Fixed Size Deposits and Rewards Penalties Quadleak](https://ethresear.ch/t/fixed-size-deposits-and-rewards-penalties-quad-leak/2073/7)

[Two Ways To Do Cross Links](https://ethresear.ch/t/two-ways-to-do-cross-links/2074/2)

[Extending Minimal Sharding with Cross Links](https://ethresear.ch/t/two-ways-to-do-cross-links/2074/2)

[Leaderless K of N Random Beacon](https://ethresear.ch/t/leaderless-k-of-n-random-beacon/2046/3)
