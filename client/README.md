# Prysmatic Labs Main Sharding Reference

This document serves as a main reference for Prysmatic Labs' sharding and beacon chain implementation in Go, along with our roadmap and compilation of active research.

# Table of Contents

-   [Sharding Introduction](#sharding-introduction)
    -   [Basic Sharding Idea and Design](#basic-sharding-idea-and-design)
-   [Roadmap Phases](#roadmap-phases)
    -   [The Ruby Release: Local Network](#the-ruby-release-local-network)
    -   [The Sapphire Release: Ropsten Testnet](#the-sapphire-release-ropsten-testnet)
    -   [The Diamond Release: Ethereum Mainnet](#the-diamond-release-ethereum-mainnet)
-   [Beacon Chain and Sharding Alpha Implementation](#beacon-chain-and-sharding-alpha-implementation)
    -   [System Architecture](#system-architecture)
    -   [System Start and User Entrypoint](#system-start-and-user-entrypoint)
    -   [Attester Sampling](#attester-sampling)
-   [Security Considerations](#security-considerations)
    -   [Not Included in Ruby Release](#not-included-in-ruby-release)
    -   [Enforced Windback](#enforced-windback)
-   [Active Questions and Research](#active-questions-and-research)
-   [Acknowledgements](#acknowledgements)
-   [References](#references)

# Sharding Introduction

Currently, every single node running the Ethereum network has to process every single transaction that goes through the network. This gives the blockchain a high amount of security because of how much validation goes into each block, but at the same time it means that an entire blockchain is only as fast as its individual nodes and not the sum of its parts. Currently, transactions on the EVM are non-parallelizable, and every transaction is executed in sequence globally. The scalability problem then has to do with the idea that a blockchain can have at most 2 of these 3 properties: decentralization, security, and scalability.

If we have scalability and security, it would mean that our blockchain is centralized and that would allow it to have a faster throughput. Right now, Ethereum is decentralized and secure, but not scalable.

An approach to solving the scalability trilemma is the idea of blockchain sharding, where we split the entire state of the network into partitions called shards that contain their own independent piece of state and transaction history. In this system, certain nodes would process transactions only for certain shards, allowing the throughput of transactions processed in total across all shards to be much higher than having a single shard do all the work as the main chain does now.

## Basic Sharding Idea and Design

A sharded blockchain system is made possible by having nodes store “signed metadata” in the main chain of latest changes within each shard chain. Through this, we manage to create a layer of abstraction that tells us enough information about the global, synced state of parallel shard chains. These messages are called **cross-links**, which are specific structures that encompass important information about the shard blocks (known as **collations**) of a shard in question. Collations are created by actors known as **proposers** that are tasked with packaging transactions into collation bodies. These collations are then voted on by a party of actors known as **attesters**. These attesters are randomly selected for particular periods of time in certain shards and are then tasked into reaching consensus on these chains via a **proof of stake** system.

Cross-links are stored in blocks on a full proof of stake chain known as a **beacon chain**, which will be implemented as a sidechain to the Ethereum main chain initially.

Cross-links are holistic descriptions of the state and transactions on a certain shard. Transactions in a shard are stored in **collations** which contain both a collation header and collation body  A collation header at its most basic, high level summary contains information about who created it, when it was added to a shard, and its internal data stored as serialized blobs.

For detailed information on protocol primitives including collations, see: [Protocol Primitives](#protocol-primitives). We will have a few types of nodes that do the heavy lifting of our sharding logic: **proposers, attesters, and attesters**. The basic role of proposers is to fetch pending transactions from the txpool, wrap them into collations, grow the shard chains, and submit cross-links to the beacon chain.

We still keep the Ethereum main chain and deploy a smart contract into it known as the **Validator Registration Contract**, where users can deposit and burn 32 ETH. Beacon chain nodes would listen to deposits in this contract and consequently queue up a user with the associated address as a validator in the beacon chain PoS system. Validators then become part of a registered validator set in the beacon chain, and are committees of validators are selected to become attesters on shard chains in certain periods of blocks until they are ventually reshuffled into different shards.

Attesters are in charge of checking for data availability of such collations and reach consensus on canonical shard chains. So then, are proposers in charge of state execution? The short answer is that phase 1 will contain **no state execution**. Instead, proposers will simply package all types of transactions into collations and later down the line, agents known as executors will download, run, and validate state as they need to through possibly different types of execution engines (potentially TrueBit-style, interactive execution).

This separation of concerns between attesters and proposers allows for more computational efficiency within the system, as attesters will not have to do the heavy lifting of state execution and focus solely on consensus through fork-choice rules. In this scheme, it makes sense that eventually **proposers** will become **executors** in later phases of a sharding spec.

Given that we are splitting up the global state of the Ethereum blockchain into shards, new types of attacks arise because fewer resources are required to completely dominate a shard. This is why a **source of randomness** and periods are critical components to ensuring the integrity of the system.

The Ethereum Wiki’s [Sharding FAQ](https://github.com/ethereum/wiki/wiki/Sharding-FAQ) suggests pseudorandom sampling of attesters on each shard. The goal is so that these attesters will not know which shard they will get in advance. Otherwise, malicious actors could concentrate resources into a single shard and try to overtake it (See: [1% Attack](https://medium.com/@icebearhww/ethereum-sharding-and-finality-65248951f649)).

Sharding revolves around being able to store shard metadata in a full proof of stake chain known as a beacon chain. For pseudorandomness generation, a RANDAO mechanism can be used in the beacon chain to shuffle validators securely.

# Roadmap Phases

Prysmatic Labs will implement the beacon chain spec posted on [ETHResearch](https://ethresear.ch/t/convenience-link-to-full-casper-chain-v2-spec/2332) by the Foundation's research team and roll out a sharding client that communicates with this beacon.

To concretize these phases, we will be releasing our implementation of sharding and the beacon chain as follows:

## The Ruby Release: Local Network

Our current work is focused on creating a localized version of a beacon chain with a sharding system that would include the following:

-   A minimal, **beacon chain node** that will interact with a main chain geth node via JSON-RPC
-   A **Validator Registration Contract** deployed on the main chain where a beacon node can read logs to check for registered validators
-   A minimal, gossipsub shardp2p network
-   Ability for proposers/attesters/attesters to be selected by the beacon chain's randomness into committees that work on specific shards
-   Ability to serialize blobs into collations on shard chains and advance the growth of the shard chains
-   An observer node that can join a network on shardp2p, sync to the latest head, and send tx's to nodes in the network


We will forego several security considerations that will be critical for testnet and mainnet release for the purposes of demonstration and local network testing as part of the Ruby Release (See: [Security Considerations Not Included in Ruby](#not-included-in-ruby-release)).

ETA: To be determined

## The Sapphire Release: Ropsten Testnet

Part 1 of the **Sapphire Release** will focus around getting the **Ruby Release** polished enough to be live on an Ethereum testnet and manage a a beacon chain + sharding system. This will require a lot more elaborate simulations around the safety of the randomness behind the attester assignments in the SMC. Futhermore we need to pass stress testing against DoS and other sorts of byzantine attacks. Additionally, it will be the first release to have real users proposing collations concurrently with attesters reaching consensus on these collations, alongside beacon node validators producing blocks via PoS.

Part 2 of the **Sapphire Release** will focus on implementing state execution and defining the State Transition Function for sharding on a local testnet (as outlined in [Beyond Phase 1](#beyond-phase-1)) as an extenstion to the Ruby Release.

ETA: To be determined

## The Diamond Release: Ethereum Mainnet

The **Diamond Release** will reconcile the best parts of the previous releases and deploy a full-featured, cross-shard transaction system through a beacon chain, casper FFG-enabled, sharding release. As expected, this is the most difficult and time consuming release on the horizon for Prysmatic Labs. We plan on growing our community effort significantly over the first few releases to get all hands-on deck preparing for this.

The Diamond Release should be considered the production release candidate for sharding Ethereum on the mainnet.

ETA: To Be determined

# Beacon Chain and Sharding Alpha Implementation

Prysmatic Labs will begin by focusing its implementation entirely on the **Ruby Release** from our roadmap. We plan on being as pragmatic as possible to create something that can be locally run by any developer as soon as possible. Our initial deliverable will center around a command line tool that will serve as an entrypoint into a beacon chain node that allows for users to become a attester, proposer, and to manage the growth of shard chains.

Here is a reference spec explaining how our initial system will function:

## System Architecture

Our implementation revolves around the following core components:

-   A **main chain node** that spins up an instance of the Ethereum blockchain and mines on the Proof of Work chain
-   A **beacon chain** that connects to this main chain node via JSON-RPC
-   A **shardp2p system** that allows nodes to reshuffle across shard networks

A basic, end-to-end example of the system is as follows:

1.  _**User deposits 32 ETH into a Validator Registration Contract on the main chain:**_ the beacon chain listens for the logs in the main chain to queue that validator into the beacon chain chain's main event loop

2.  _**Registered validator begins PoS process to propose blocks:**_ the PoS validator has the resposibility to participate in the addition of new blocks to the beacon chain

3.  _**RANDAO mechanism selects committees of proposers/attesters/attesters for shards:**_ the beacon chain node will use its RANDAO mechanism to select committees of proposers, attesters, and attesters that each have responsibilities within the sharding system. Refer to the [Full Casper Chain V2 Doc](https://ethresear.ch/t/convenience-link-to-full-casper-chain-v2-spec/2332) for extensive detail on the different fields in the state of the beacon chain related to sharding

4. _**Beacon Chain State Advances, Committees are Reshuffled:**_ upon completing responsibilities, the different actors of the sharding system are them reshuffled into new committees on different shards

## System Start and User Entrypoint

Our Ruby Release requires users to start a local geth node running a localized, private blockchain to deploy the **Validator Registration Contract**. This will kickstart the entire beacon chain sync process and listen for registrations of validators in the main chain VRC. The beacon node begins to work by its main loop, which involves the following steps:

1.  _**Sync to the latest block header on the beacon chain:**_ the node will begin a sync process for the beacon chain

2.  _**Assign the validator as a proposer/attester/attester based on RANDAO mechanism:**_ on incoming headers, the client will interact with the SMC to check if the current user is an eligible attester for an upcoming period (only a few minutes notice)

3.  _**Process shard cross-links:**_ once a attester is selected, he/she has to download subimtted collation headers for the shard in a certain period and check for their data availability

5.  _**Reshuffle committees**_ the attester votes on the available collation header that came first in the submissions. 

6.  _**Propose blocks and finalize incoming blocks via PoS:**_ Once attesters vote, headers that received >=2/3 votes are selected as canonical

## Attester Sampling

The probability of being selected as a attester on a particular shard is being heavily researched in the latest ETHResearch discussions. As specified in the [Sharding FAQ](https://github.com/ethereum/wiki/wiki/Sharding-FAQ) by Vitalik, “if validators [collators] could choose, then attackers with small total stake could concentrate their stake onto one shard and attack it, thereby eliminating the system’s security.”

The idea is that attesters should not be able to figure out which shard they will become a attester of and during which period they will be assigned with anything more than a few minutes notice.

Ideally, we want attesters to shuffle across shards very rapidly and through a source of pseudorandomness built in-protocol.

Despite its benefits, random sampling does not help in a bribing, coordinated attack model. In Vitalik’s own words:

_"Either the attacker can bribe the great majority of the sample to do as the attacker pleases, or the attacker controls a majority of the sample directly and can direct the sample to perform arbitrary actions at low cost (O(c) cost, to be precise).
At that point, the attacker has the ability to conduct 51% attacks against that sample. The threat is further magnified because there is a risk of cross-shard contagion: if the attacker corrupts the state of a shard, the attacker can then start to send unlimited quantities of funds out to other shards and perform other cross-shard mischief. All in all, security in the bribing attacker or coordinated choice model is not much better than that of simply creating O(c) altcoins.”_

However, this problem transcends the sharding scheme itself and goes into the broader problem of fraud detection, which we have yet to comprehensively address.

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

# Security Considerations

## Not Included in Ruby Release

We will not be considering data availability proofs (part of the stateless client model) as part of the ruby release we will not be implementing them as it just yet as they are an area of active research.

Additionally, we will be using simple blockhashes for randomness in committee selections instead of a full RANDAO mechanism.

## Enforced Windback

When attesters are extending shard chains, it is critical that they are able to verify some of the collation headers in the immediate past for security purposes. There have already been instances where mining blindly has led to invalid transactions that forced Bitcoin to undergo a fork (See: [BIP66 Incident](https://bitcoin.stackexchange.com/questions/38437/what-is-spv-mining-and-how-did-it-inadvertently-cause-the-fork-after-bip66-wa)).

This process of checking previous blocks is known as **“windback”**. In a [post](https://ethresear.ch/t/enforcing-windback-validity-and-availability-and-a-proof-of-custody/949) by Justin Drake on ETHResearch, he outlines that this is necessary for security, but is counterintuitive to the end-goal of scalability as this obviously imposes more computational and network constraints on nodes.

One way to enforce **validity** during the windback process is for nodes to produce zero-knowedge proofs of validity that can then be stored in collation headers for quick verification.

On the other hand, to enforce **availability** for the windback process, a possible approach is for nodes to produce “proofs of custody” in collation headers that prove the attester was in possession of the full data of a collation when produced. Drake proposes a constant time, non-interactive zkSNARK method for attesters to check these proofs of custody. In his construction, he mentions splitting up a collation body into “chunks” that are then mixed with the node's private key through a hashing scheme. The security in this relies in the idea that a node would not leak his/her private key without compromising him or herself, so it provides a succinct way of checking if the full data was available when a node processed the collation body and proof was created.

# Active Questions and Research

## Leaderless Random Beacons

In the prevous research on random beacons, committees are able to generate random numbers if a certain number of participants participate correctly. This is similar to the random beacon used in Dfinity without the use of BLS threshold signatures. The scheme is separated into two separate sections.

In the first section, each participant is committed to a secret key and shares the resulting public key.
In the second section, each participant will use their secret key to deterministically build a polynomial and that polynomial is used to to create n shares (where n is the size of the committee) which can then be encrypted with respect to the public keys and then shared publicly.

Then, in the resolution, all participants are then to reveal their private keys, once the key is revealed anyone can check if the participant committed correctly. We can define the random output as the sum of the private keys for which the participants committed correctly.

<https://ethresear.ch/t/leaderless-k-of-n-random-beacon/2046/3>

## Torus-shaped Sharded P2P Network

One recommendation is using a [Torus-shaped sharding network](https://commons.wikimedia.org/wiki/File:Toroidal_coord.png). In this paradigm, there would be a single network that all shards share rather than a network for each shard. Nodes would propagate messages to peers interested in neighboring shards. A node listening on shard 16 would relay messages for shards in range of 11 to 21 (i.e +/-5). Nodes that need to listen on multiple shards can quickly change shards to find peers that may relay necessary messages. A node could potentially have access to messages from all shards with only 10 distinct peers for a 100 shard network. At the same time, we're considering replacing [DEVp2p](https://github.com/ethereum/wiki/wiki/%C3%90%CE%9EVp2p-Wire-Protocol) with [libp2p](https://github.com/libp2p) framework, which is actively maintained, proven to work with IPFS, and comes with client libraries for Go and Javascript. 
Active research is on going for moving Ethereum fron DEVp2p to libp2p. We are looking into how to map shards to libp2p and how to balance flood/gossipsub progagation vs active connections. Here is the current work of [poc](https://github.com/mhchia/go-libp2p/tree/poc-testing/examples/minimal) on [gossiphub](https://github.com/libp2p/go-floodsub/pull/67/). It utilizies pubsub for propagating messages such as transactions, proposals and sharded collations.

<https://ethresear.ch/t/torus-shaped-sharding-network/1720>

 
## Sparse Merkle Tree for State Storage

With a sharded network comes sharded state storage. State sync today is difficult for clients today. While the blockchain data stored on disk might use~80gb for a fast sync, less than 5gb of that disk is state data while state sync accounts for the majority of time spent syncing. As the state grows, this issue will also grow. We imagine that it might be difficult to sync effectively when there are 100 shards and 100 different state tries. One recommendation from the Ethereum Research team outlines using [sparse merkle trees].(https://www.links.org/files/RevocationTransparency.pdf)

<https://ethresear.ch/t/data-availability-proof-friendly-state-tree-transitions/1453>

# Acknowledgements

A special thanks for entire [Prysmatic Labs](https://gitter.im/prysmaticlabs/prysm) team for helping put this together and to Ethereum Research (Hsiao-Wei Wang, Vitalik, Justin Drake) for the help and guidance in our approach.

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
