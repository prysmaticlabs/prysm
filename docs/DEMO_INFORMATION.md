# Ethereum Serenity Prysm Demo

**IMPORTANT: This document describes our v0.0.0 demo release from October 2018, which is no longer relevant to our current work and is only here for historical purposes**

## Overview & Research Background

At Prysmatic Labs, we started working on Ethereum Serenity all the way back since Vitalik first had a Sharding FAQ as the only reference for the system at the start of 2018. A lot has happened, with the specification evolving from a series of ETHResearch blog posts into a minimal viable blueprint for how to design a scalable, secure Ethereum blockchain using Casper Proof of Stake and Sharding at its core.

Now, the plan is to deploy ETH2.0 as a sidechain of Ethereum known as a beacon chain, where validators can stake their Ether and participate in consensus to vote on occurrences on shards known as cross-links.

## Version 0.0.0: Beacon Chain+Validator Demo

We call our Ethereum Serenity project Prysm, which will serve as the scaffold for a production-oriented release fully written in Go. We have been working hard to ensure we have a robust implementation of the Ethereum Serenity specification as created by the Ethereum Research Team along with industry standard approaches to building large scale applications.

We are proud to announce our very first release of Prysm, v0.0.0, which will serve as the building block for all future releases as we get to production. We want to show the community we have a project we have put a lot of work into through careful thought and design decisions that we hope will set a standard for future ETH2.0 developments.

## What This Release Encompasses

Version 0.0.0 includes a basic beacon-chain+validator demo which can do the following:

- Spin up a beacon chain node from a genesis configuration
- Connect a validator client via RPC
- The validator client gets shuffled into a specific shard at a given slot
- Validators propose/attest to canonical beacon blocks during their assigned slot
- Validators get reshuffled into shards every new cycle transition
- Casper FFG rewards/penalties are included in this release even though they are a constant area of research
- Basic, locally networked p2p via libp2p and the mDNS discovery protocol
- Beacon chain block sync through p2p (listening for incoming blocks + syncing to a latest head from scratch)
- A useful simulator of beacon blocks (this allows us to simulate other beacon nodes relaying info to our node locally)
- Storing blocks/attestations/states to a write-optimized key-value backend known as BoltDB
- gRPC public API client/server for querying a beacon node for canonical blocks, states, and latest validator assignments
- A robust, scalable build system known as Bazel used in production at Google, Pinterest, Dropbox, and other industry giants
- A Web3 Subscription service to listen to latest mainchain blocks and validator registrations

## Not Included in the Release

- Although a Validator Registration Contract is included, validator rotation, withdrawals, and dynasty transitions are not yet included
- Shards, their associated design, and cross-shard transactions are not included in this release
- Fork-choice rule for ETH2.0 is not yet included, we use a naive fork choice rule for this release
- Signature aggregation and verification are not included in this release
- Randomness via RANDAO and VDF are not included in this release as they are an active area of research - we use basic block hashes as a stub for now
- Serialization format for ETH2.0 is still an active area of research and is not included here
- Shardp2p and beacon node p2p peer discovery have not yet been designed beyond mDNS
- State execution is not included as it depends on shard functionality

## How to Run the Demo

Curious to see the system working and running a validator client locally? We have put together comprehensive instructions on running our demo in our Github repository! Try it out and see for yourself :). 

[Running Instructions](https://github.com/prysmaticlabs/prysm/blob/master/README.md#instructions)

You’ll be able to spin up a beacon node, connect a validator client, and start getting assigned to shards where you will then create beacon blocks or vote on beacon blocks through structures called “attestations”. If you are not quite as familiar with the Ethereum Serenity Roadmap and Spec, check out the following links:

- [Ethereum Serenity Devs Handbook](https://notes.ethereum.org/s/BkSZAJNwX#)
- [Ethereum Serenity Casper+Sharding Specification](https://github.com/ethereum/eth2.0-specs/tree/master/specs)

Even though canonical blocks are created in the demo due to your activity as a validator, you’ll quickly see not much can be done with these blocks until real, state execution comes into play. However, this beacon chain is a critical piece of consensus and coordination for all actors participating in Ethereum Serenity and will as the foundation for a full-fledged, sharding system. 

