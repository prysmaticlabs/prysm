# Prysmatic Labs Main Sharding Reference

This document serves as a main reference for Prysmatic Labs' sharding implementation for the go-ethereum client, along with our roadmap and compilation of active research and approaches to various sharding schemes.

# Table of Contents

-   [Sharding Introduction](#sharding-introduction)
    -   [Basic Sharding Idea and Design](#basic-sharding-idea-and-design)
-   [Roadmap Phases](#roadmap-phases)
    -   [The Ruby Release: Local Network](#the-ruby-release-local-network)
    -   [The Sapphire Release: Ropsten Testnet](#the-sapphire-release-ropsten-testnet)
    -   [The Diamond Release: Ethereum Mainnet](#the-diamond-release-ethereum-mainnet)
-   [Go-Ethereum Sharding Alpha Implementation](#go-ethereum-sharding-alpha-implementation)
    -   [System Architecture](#system-architecture)
    -   [System Start and User Entrypoint](#system-start-and-user-entrypoint)
    -   [The Sharding Manager Contract](#the-sharding-manager-contract)
        -   [Notary Sampling](#notary-sampling)
    -   [The Notary Client](#the-notary-client)
        -   [Local Shard Storage](#local-shard-storage)
    -   [The Proposer Client](#the-proposer-client)
        -   [Collation Headers](#collation-headers)
    -   [Protocol Modifications](#protocol-modifications)
        -   [Protocol Primitives: Collations, Blocks, Transactions, Accounts](#protocol-primitives-collations-blocks-transactions-accounts)
        -   [The EVM: What You Need to Know](#the-evm-what-you-need-to-know)
    -   [Sharding In-Practice](#sharding-in-practice)
        -   [Use-Case Stories: Proposers](#use-case-stories-proposers)
        -   [Use-Case Stories: Notaries](#use-case-stories-notaries)
    -   [Current Status](#current-status)
-   [Security Considerations](#security-considerations)
    -   [Not Included in Ruby Release](#not-included-in-ruby-release)
    -   [Bribing, Coordinated Attack Models](#bribing-coordinated-attack-models)
    -   [Enforced Windback](#enforced-windback)
    -   [The Data Availability Problem](#the-data-availability-problem)
        -   [Introduction and Background](#introduction-and-background)
        -   [On Uniquely Attributable Faults](#on-uniquely-attributable-faults)
        -   [Erasure Codes](#erasure-codes)
-   [Beyond Phase 1](#beyond-phase-1)
    -   [Cross-Shard Communication](#cross-shard-communication)
        -   [Receipts Method](#receipts-method)
        -   [Merge Blocks](#merge-blocks)
        -   [Synchronous State Execution](#synchronous-state-execution)
    -   [Transparent Sharding](#transparent-sharding)
    -   [Tightly-Coupled Sharding (Fork-Free Sharding)](#tightly-coupled-sharding-fork-free-sharding)
-   [Active Questions and Research](#active-questions-and-research)
-   [Community Updates and Contributions](#community-updates-and-contributions)
-   [Acknowledgements](#acknowledgements)
-   [References](#references)

# Sharding Introduction

Currently, every single node running the Ethereum network has to process every single transaction that goes through the network. This gives the blockchain a high amount of security because of how much validation goes into each block, but at the same time it means that an entire blockchain is only as fast as its individual nodes and not the sum of its parts. Currently, transactions on the EVM are non-parallelizable, and every transaction is executed in sequence globally. The scalability problem then has to do with the idea that a blockchain can have at most 2 of these 3 properties: decentralization, security, and scalability.

If we have scalability and security, it would mean that our blockchain is centralized and that would allow it to have a faster throughput. Right now, Ethereum is decentralized and secure, but not scalable.

An approach to solving the scalability trilemma is the idea of blockchain sharding, where we split the entire state of the network into partitions called shards that contain their own independent piece of state and transaction history. In this system, certain nodes would process transactions only for certain shards, allowing the throughput of transactions processed in total across all shards to be much higher than having a single shard do all the work as the main chain does now.

## Basic Sharding Idea and Design

A sharded blockchain system is made possible by having nodes store “signed metadata” in the main chain of latest changes within each shard chain. Through this, we manage to create a layer of abstraction that tells us enough information about the global, synced state of parallel shard chains. These messages are called **cross-links**, which are specific structures that encompass important information about the shard blocks (known as **collations**) of a shard in question. Collations are created by actors known as **proposers** that are tasked with packaging transactions into collation bodies. These collations are then voted on by a party of actors known as **notaries**. These notaries are randomly selected for particular periods of time in certain shards and are then tasked into reaching consensus on these chains via a **proof of stake** system.

Cross-links are stored in blocks on a full proof of stake chain known as a **beacon chain**, which will be implemented as a sidechain to the Ethereum main chain initially.

Cross-links are holistic descriptions of the state and transactions on a certain shard. Transactions in a shard are stored in **collations** which contain both a collation header and collation body  A collation header at its most basic, high level summary contains information about who created it, when it was added to a shard, and its internal data stored as serialized blobs.

For detailed information on protocol primitives including collations, see: [Protocol Primitives](#protocol-primitives). We will have a few types of nodes that do the heavy lifting of our sharding logic: **proposers, notaries, and attesters**. The basic role of proposers is to fetch pending transactions from the txpool, wrap them into collations, grow the shard chains, and submit cross-links to the beacon chain.

<!--[Proposer{bg:wheat}]fetch txs-.->[TXPool], [TXPool]-.->[Proposer{bg:wheat}], [Proposer{bg:wheat}]-package txs>[Collation|header|body], [Collation|header|body]-submit header>[Sharding Manager Contract], [Notary{bg:wheat}]downloads collation availability and votes-.->[Sharding Manager Contract]-->
![proposers](https://yuml.me/69cbd7da.png)

We still keep the Ethereum main chain and deploy a smart contract into it known as the **Validator Registration Contract**, where users can deposit and burn 32 ETH. Beacon chain nodes would listen to deposits in this contract and consequently queue up a user with the associated address as a validator in the beacon chain PoS system. Validators then become part of a registered validator set in the beacon chain, and are committees of validators are selected to become notaries on shard chains in certain periods of blocks until they are ventually reshuffled into different shards.

Notaries are in charge of checking for data availability of such collations and reach consensus on canonical shard chains. So then, are proposers in charge of state execution? The short answer is that phase 1 will contain **no state execution**. Instead, proposers will simply package all types of transactions into collations and later down the line, agents known as executors will download, run, and validate state as they need to through possibly different types of execution engines (potentially TrueBit-style, interactive execution).

This separation of concerns between notaries and proposers allows for more computational efficiency within the system, as notaries will not have to do the heavy lifting of state execution and focus solely on consensus through fork-choice rules. In this scheme, it makes sense that eventually **proposers** will become **executors** in later phases of a sharding spec.

Given that we are splitting up the global state of the Ethereum blockchain into shards, new types of attacks arise because fewer resources are required to completely dominate a shard. This is why a **source of randomness** and periods are critical components to ensuring the integrity of the system.

The Ethereum Wiki’s [Sharding FAQ](https://github.com/ethereum/wiki/wiki/Sharding-FAQ) suggests pseudorandom sampling of notaries on each shard. The goal is so that these notaries will not know which shard they will get in advance. Otherwise, malicious actors could concentrate resources into a single shard and try to overtake it (See: [1% Attack](https://medium.com/@icebearhww/ethereum-sharding-and-finality-65248951f649)).

Sharding revolves around being able to store shard metadata in a full proof of stake chain known as a beacon chain. For pseudorandomness generation, a RANDAO mechanism can be used in the beacon chain to shuffle validators securely.

# Roadmap Phases

Prysmatic Labs will implement the beacon chain spec posted on [ETHResearch]() by the Foundation's research team and roll out a sharding client that communicates with this beacon.

To concretize these phases, we will be releasing our implementation of sharding and the beacon chain as follows:

## The Ruby Release: Local Network

Our current work is focused on creating a localized version of a beacon chain with a sharding system that would include the following:

-   A minimal, **beacon chain node** that will interact with a main chain geth node via JSON-RPC
-   A **Validator Registration Contract** deployed on the main chain where a beacon node can read logs to check for registered validators
-   A minimal, gossipsub shardp2p network
-   Ability for proposers/notaries/attesters to be selected by the beacon chain's randomness into committees that work on specific shards
-   Ability to serialize blobs into collations on shard chains and advance the growth of the shard chains
-   An observer node that can join a network on shardp2p, sync to the latest head, and send tx's to nodes in the network


We will forego several security considerations that will be critical for testnet and mainnet release for the purposes of demonstration and local network testing as part of the Ruby Release (See: [Security Considerations Not Included in Ruby](#not-included-in-ruby-release)).

ETA: To be determined

## The Sapphire Release: Ropsten Testnet

Part 1 of the **Sapphire Release** will focus around getting the **Ruby Release** polished enough to be live on an Ethereum testnet and manage a a beacon chain + sharding system. This will require a lot more elaborate simulations around the safety of the randomness behind the notary assignments in the SMC. Futhermore we need to pass stress testing against DoS and other sorts of byzantine attacks. Additionally, it will be the first release to have real users proposing collations concurrently with notaries reaching consensus on these collations, alongside beacon node validators producing blocks via PoS.

Part 2 of the **Sapphire Release** will focus on implementing state execution and defining the State Transition Function for sharding on a local testnet (as outlined in [Beyond Phase 1](#beyond-phase-1)) as an extenstion to the Ruby Release.

ETA: To be determined

## The Diamond Release: Ethereum Mainnet

The **Diamond Release** will reconcile the best parts of the previous releases and deploy a full-featured, cross-shard transaction system through a Sharding Manager Contract on the Ethereum mainnet. As expected, this is the most difficult and time consuming release on the horizon for Prysmatic Labs. We plan on growing our community effort significantly over the first few releases to get all hands-on deck preparing for this.

The Diamond Release should be considered the production release candidate for sharding Ethereum on the mainnet.

ETA: To Be determined

# Beacon Chain + Sharding Alpha Implementation

Prysmatic Labs will begin by focusing its implementation entirely on the **Ruby Release** from our roadmap. We plan on being as pragmatic as possible to create something that can be locally run by any developer as soon as possible. Our initial deliverable will center around a command line tool that will serve as an entrypoint into a beacon chain node that allows for users to become a notary, proposer, and to manage the growth of shard chains.

Here is a reference spec explaining how our initial system will function:

## System Architecture

Our implementation revolves around 5 core components:

-   A **locally-running geth node** that spins up an instance of the Ethereum blockchain and mines on the Proof of Work chain
-   A **Sharding Manager Contract (SMC)** that is deployed onto this blockchain instance
-   A **sharding node** that connects to the running geth node through JSON-RPC, provides bindings to the SMC
-   A **notary service** that allows users to stake ETH into the SMC and be selected as a notary in a certain period on a shard
-   A **proposer service** that is tasked with processing pending tx's into collations that are then submitted to the SMC. In phase 1, proposers _do not_ execute state, but rather just serialize pending tx data into possibly valid/invalid data blobs.

Our initial implementation will function through simple command line arguments that will allow a user running the local geth node to deposit ETH into the SMC and join as a notary that is randomly assigned to a shard in a certain period.

A basic, end-to-end example of the system is as follows:

1.  _**User starts a sharding node and deposits 1000ETH into the SMC:**_ the sharding node connects to a locally running geth node and asks the user to confirm a deposit from his/her personal account.

2.  _**Client connects & listens to incoming headers from the geth node and assigns user as notary on a shard per period:**_ The notary is selected in CURRENT_PERIOD + LOOKEAD_PERIOD (which is around a 5 minutes notice) and must download data for collation headers submitted in that time period.

3.  _**Concurrently, a proposer protocol processes pending transaction data into blobs:**_ the proposer client will create collation bodies and submit their headers to the SMC. In Phase 1, it is important to note that we will _not_ have any state execution. Proposers will just serialize pending tx into fixed collation body sizes without executing them for state transition validity.

5.  _**The set of notaries vote on collation headers as canonical unitl the period ends:**_ the headers that received >= 2/3 votes are accepted as canonical.

6.  _**User is selected as notary again on the SMC in a different period or can withdraw his/her stake:**_ the user can keep staking and voting on incoming collation headers and restart the process, or withdraw his/her stake and be deregistered from the SMC.

Now, we’ll explore our architecture and implementation in detail as part of the go-ethereum repository.

## System Start and User Entrypoint

Our Ruby Release requires users to start a local geth node running a localized, private blockchain to deploy the **SMC** into. Users can spin up a notary client as a command line entrypoint into geth while the node is running as follows:

    geth sharding --actor "notary" --datadir /path/to/your/datadir --password /path/to/your/password.txt --networkid 12345 --deposit

This will extract 1000ETH from the user's account balance and insert him/her into the SMC's notaries. Then, the program will listen for incoming block headers and notify the user when he/she has been selected as to vote on collations for a certain shard in a given period. Once you are selected, the sharding node will download collation information to check for data availability on vote on proposals that have been submitted via the `addHeader` function on the SMC.

Users can also run a proposer client that is tasked with processing transactions into collations and submitting them to the SMC via the `addHeader` function. 

    geth sharding --actor "proposer" --datadir /path/to/your/datadir --password /path/to/your/password.txt --networkid 12345

This client is tasked with processing pending transactions into blobs within collations by serializing data into collation bodies. It is responsible for submitting proposals (collation headers) to the SMC via the `addHeader` function.

The sharding node begins to work by its main loop, which involves the following steps:

1.  _**Subscribe to incoming block headers:**_ the client will begin by issuing a subscription over JSON-RPC for block headers from the running geth node.

2.  _**Check shards for notary selection within LOOKEAD_PERIOD:**_ on incoming headers, the client will interact with the SMC to check if the current user is an eligible notary for an upcoming period (only a few minutes notice)

3.  _**If the notary is selected, check data availability for submitted collation headers:**_ once a notary is selected, he/she has to download subimtted collation headers for the shard in a certain period and check for their data availability

5.  _**The notary issues a vote:**_ the notary votes on the available collation header that came first in the submissions. 

6.  _**Other notaries vote, period ends, and header is selected as canonical shard chain header:**_ Once notaries vote, headers that received >=2/3 votes are selected as canonical

### Notary Sampling

The probability of being selected as a notary on a particular shard is being heavily researched in the latest ETHResearch discussions. As specified in the [Sharding FAQ](https://github.com/ethereum/wiki/wiki/Sharding-FAQ) by Vitalik, “if validators [collators] could choose, then attackers with small total stake could concentrate their stake onto one shard and attack it, thereby eliminating the system’s security.”

The idea is that notaries should not be able to figure out which shard they will become a notary of and during which period they will be assigned with anything more than a few minutes notice.

Ideally, we want notaries to shuffle across shards very rapidly and through a source of pseudorandomness built in-protocol.

Despite its benefits, random sampling does not help in a bribing, coordinated attack model. In Vitalik’s own words:

_"Either the attacker can bribe the great majority of the sample to do as the attacker pleases, or the attacker controls a majority of the sample directly and can direct the sample to perform arbitrary actions at low cost (O(c) cost, to be precise).
At that point, the attacker has the ability to conduct 51% attacks against that sample. The threat is further magnified because there is a risk of cross-shard contagion: if the attacker corrupts the state of a shard, the attacker can then start to send unlimited quantities of funds out to other shards and perform other cross-shard mischief. All in all, security in the bribing attacker or coordinated choice model is not much better than that of simply creating O(c) altcoins.”_

However, this problem transcends the sharding scheme itself and goes into the broader problem of fraud detection, which we have yet to comprehensively address.

## The Notary Client

One of the main running threads of our implementation is the notary client, which serves as a bridge between users staking their ETH to become notaries and the **Sharding Manager Contract** that verifies collation headers on the canonical chain.

When we launch the client, The instance connects to a running geth node via JSON-RPC and calls the deposit function on a deployed, Sharding Manager Contract to insert the user into a notary pool. Then, we subscribe for updates on incoming block headers and determine if the user is a notary on receiving each header. Once we are selected within a LOOKAHEAD_PERIOD, our client fetches data associated with submitted collation headers to that shard. The notary votes on the SMC, and if other notaries reach consensus, the collation is accepted as canonical.

### Local Shard Storage

Local shard information is done through a key-value store used to store the mainchain information in the local data directory specified by the running geth node. Adding a collation to a shard will effectively modify this key-value store.

Work in progress.

## The Proposer Client

In addition to launching a notary client, our system requires a user to concurrently launch a proposer client that is tasked with fetching pending tx’s from the network and creating collations that can be sent to the SMC.

This client connects via JSON-RPC to give the client the ability to call required functions on the SMC. The proposer is tasked with packaging pending transaction data into _blobs_ and **not** executing these transactions. This is very important, we will not consider state execution until later phases of a sharding roadmap.

Then, the proposer node calls the `addHeader` function on the SMC by submitting this collation header. We’ll explore the structure of collation headers in this next section.

## Peer Discovery and Shard Wire Protocol

Work in progress.

## Protocol Modifications

### Protocol Primitives: Collations, Blocks, Transactions, Accounts

(Outline the interfaces for each of these constructs, mention crucial changes in types or receiver methods in Go for each, mention transaction access lists)

Work in progress.

### The EVM: What You Need to Know

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

## Bribing, Coordinated Attack Models

Work in progress.

## Enforced Windback

When notaries are extending shard chains, it is critical that they are able to verify some of the collation headers in the immediate past for security purposes. There have already been instances where mining blindly has led to invalid transactions that forced Bitcoin to undergo a fork (See: [BIP66 Incident](https://bitcoin.stackexchange.com/questions/38437/what-is-spv-mining-and-how-did-it-inadvertently-cause-the-fork-after-bip66-wa)).

This process of checking previous blocks is known as **“windback”**. In a [post](https://ethresear.ch/t/enforcing-windback-validity-and-availability-and-a-proof-of-custody/949) by Justin Drake on ETHResearch, he outlines that this is necessary for security, but is counterintuitive to the end-goal of scalability as this obviously imposes more computational and network constraints on nodes.

One way to enforce **validity** during the windback process is for nodes to produce zero-knowedge proofs of validity that can then be stored in collation headers for quick verification.

On the other hand, to enforce **availability** for the windback process, a possible approach is for nodes to produce “proofs of custody” in collation headers that prove the notary was in possession of the full data of a collation when produced. Drake proposes a constant time, non-interactive zkSNARK method for notaries to check these proofs of custody. In his construction, he mentions splitting up a collation body into “chunks” that are then mixed with the node's private key through a hashing scheme. The security in this relies in the idea that a node would not leak his/her private key without compromising him or herself, so it provides a succinct way of checking if the full data was available when a node processed the collation body and proof was created.

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

## Fixed ETH Deposit Size for Notaries

A notary must submit a deposit to the Sharding Manager Contract in order to get randomly selected to vote on a block. A fixed size deposit is good for making the random selection convenient and work well with slashing, as it can always destroy at least a minimum amount of ether. However, a fixed-size deposit does not do well with rewards and penalties. An alternative solution is to design incentive system where rewards and penalties are tracked in a separate variable, and when the final balance when the withdrawal penalties minus rewards reach a threshold, the notary can be voted out. Such a design might ignore an important function which is to reduce the influence of notaries that are offline. In Casper FFG, if more than 1/3 of validators to offline around same time, the deposits will begin to leak quickly. This is called quadratic leak.

<https://ethresear.ch/t/fixed-size-deposits-and-rewards-penalties-quad-leak/2073/7>

# Community Updates and Contributions

Excited by our work and want to get involved in building out our sharding releases? We created this document as a single source of reference for all things related to sharding Ethereum, and we need as much help as we can get!

You can explore our [Current Projects](https://github.com/prysmaticlabs/geth-sharding/projects) in-the works for the Ruby release. Each of the project boards contain a full collection of open and closed issues relevant to the different parts of our first implementation that we use to track our open source progress. Feel free to fork our repo and start creating PR’s after assigning yourself to an issue of interest. We are always chatting on [Gitter](https://gitter.im/prysmaticlabs/geth-sharding), so drop us a line there if you want to get more involved or have any questions on our implementation!

**Contribution Steps**

-   Create a folder in your `$GOPATH` and navigate to it `mkdir -p $GOPATH/src/github.com/ethereum && cd $GOPATH/src/github.com/ethereum`
-   Clone our repository as `go-ethereum`, `git clone https://github.com/prysmaticlabs/geth-sharding ./go-ethereum`
-   Fork the `go-ethereum` repository on Github: <https://github.com/ethereum/go-ethereum>
-   Add a remote to your fork
    \`git remote add YOURNAME <https://github.com/YOURNAME/go-ethereum>

Now you should have a remote pointing to the `origin` repo (geth-sharding) and to your forked, go-ethereum repo on Github. To commit changes and start a Pull Request, our workflow is as follows:

-   Create a new branch with a clear feature name such as `git checkout -b collations-pool`
-   Issue changes with clear commit messages
-   Push to your remote `git push YOURNAME collations-pool`
-   Go to the [geth-sharding](https://github.com/prysmaticlabs/geth-sharding) repository on Github and start a PR comparing `geth-sharding:master` with `go-ethereum:collations-pool` (your fork on your profile).
-   Add a clear PR title along with a description of what this PR encompasses, when it can be closed, and what you are currently working on. Github markdown checklists work great for this.

# Acknowledgements

A special thanks for entire [Prysmatic Labs](https://gitter.im/prysmaticlabs/geth-sharding) team for helping put this together and to Ethereum Research (Hsiao-Wei Wang) for the help and guidance in our approach.

# References

[Sharding FAQ](https://github.com/ethereum/wiki/wiki/Sharding-FAQ)

[Sharding Reference Spec](https://github.com/ethereum/sharding/blob/develop/docs/doc.md)

[Ethereum Sharding and Finality - Hsiao-Wei Wang](https://medium.com/@icebearhww/ethereum-sharding-and-finality-65248951f649)

[Data Availability and Erasure Coding](https://github.com/ethereum/research/wiki/A-note-on-data-availability-and-erasure-coding)

[Proof of Visibility for Data Availability](https://ethresear.ch/t/proof-of-visibility-for-data-availability/1073)

[Enforcing Windback and Proof of Custody](https://ethresear.ch/t/enforcing-windback-validity-and-availability-and-a-proof-of-custody/949)

[Fork-Free Sharding](https://ethresear.ch/t/fork-free-sharding/1058)

[Delayed State Execution](https://ethresear.ch/t/delayed-state-execution-finality-and-cross-chain-operations/987)

[State Execution Scalability and Cost Under DDoS Attacks](https://ethresear.ch/t/state-execution-scalability-and-cost-under-dos-attacks/1048)

[Guaranteed Collation Subsidies](https://ethresear.ch/t/guaranteed-collation-subsidies/1016)

[Fork Choice Rule for Collation Proposals](https://ethresear.ch/t/fork-choice-rule-for-collation-proposal-mechanisms/922)

[Model for Phase 4 Tightly-Coupled Sharding](https://ethresear.ch/t/a-model-for-stage-4-tightly-coupled-sharding-plus-full-casper/1065)

[History, State, and Asynchronous Accumulators in the Stateless Model](https://ethresear.ch/t/history-state-and-asynchronous-accumulators-in-the-stateless-model/287)

[Torus Shaped Sharding Network](https://ethresear.ch/t/torus-shaped-sharding-network/1720)

[Data Availability Proof-friendly State Tree Transitions](https://ethresear.ch/t/data-availability-proof-friendly-state-tree-transitions/1453)

[General Framework of Overhead and Finality Time in Sharding](https://ethresear.ch/t/a-general-framework-of-overhead-and-finality-time-in-sharding-and-a-proposal/1638)

[Safety Notary Pool Size](https://ethresear.ch/t/safe-notary-pool-size/1728)

[Fixed Size Deposits and Rewards Penalties Quadleak](https://ethresear.ch/t/fixed-size-deposits-and-rewards-penalties-quad-leak/2073/7)

[Two Ways To Do Cross Links](https://ethresear.ch/t/two-ways-to-do-cross-links/2074/2)

[Extending Minimal Sharding with Cross Links](https://ethresear.ch/t/two-ways-to-do-cross-links/2074/2)

[Leaderless K of N Random Beacon](https://ethresear.ch/t/leaderless-k-of-n-random-beacon/2046/3)
