# Prysmatic Labs Main Sharding Reference

Table of Contents
=================

- [Sharding Introduction](#sharding-introduction)
  - [Basic Sharding Idea and Design](#basic-sharding-idea-and-design)
- [Roadmap Phases](#roadmap-phases)
  - [The Ruby Release: Local Network](#the-ruby-release-local-network)
  - [The Sapphire Release: Ropsten Testnet](#the-sapphire-release-ropsten-testnet)
  - [The Diamond Release: Ethereum Mainnet](#the-diamond-release-ethereum-mainnet)
- [Go-Ethereum Sharding Alpha Implementation](#go-ethereum-sharding-alpha-implementation)
  - System Architecture
  - System Start & User Entrypoint
  - The Validator Manager Contract
    - Necessary Functionality
      - Depositing ETH and Becoming a Validator
      - Determining an Eligible Proposer for a Period on a Shard
      - Withdrawing From the Validator Set
      - Processing and Verifying a Collation Header
    - Validator Sampling
    - Collation Header Approval
    - Event Logs
  - The Validator Client
    - Local Shard Storage
  - The Proposer Client
    - Collation Headers and State Execution
  - Peer Discovery and Shard Wire Protocol
  - Protocol Modifications
    - Protocol Primitives: Collations, Blocks, Transactions, Accounts
    - The EVM: What You Need to Know
  - Sharding In-Practice
    - Fork Choice Rule
    - Use-Case Stores: Proposers
    - Use-Case Stories: Validators
    - Use-Case Stories: Supernodes
  - Current Status
- Security Considerations
  - Not Included in Ruby Release
  - Bribing, Coordinated Attack Models
  - Enforced Windback
    - Explicity Finality for Stateless Clients
  - The Data Availability Problem
    - Introduction & Background
    - On Uniquely Attributable Faults
    - Erasure Codes
- Beyond Phase 1
  - Cross-Shard Communication
    - Receipts Method
    - Merge Blocks
    - Synchronous State Execution
  - Transparent Sharding
  - Tightly-Coupled Sharding (Fork-Free Sharding)
- Active Questions & Research
  - Separation of Proposals & Consensus
  - Selecting Eligible Validators Off-Chain
- Community Updates & Contributions
- References

Sharding Introduction
=====================

Currently, every single node running the Ethereum network has to process every single transaction that goes through the network. This gives the blockchain a high amount of security because of how much validation goes into each block, but at the same time it means that an entire blockchain is only as fast as its individual nodes and not the sum of their parts. Currently, transactions on the EVM are not parallelizable, and every transaction is executed in sequence globally. The scalability problem then has to do with the idea that a blockchain can have at most 2 of these 3 properties: decentralization, security, and scalability.

If we have scalability and security, it would mean that our blockchain is centralized and that would allow it to have a faster throughput. Right now, Ethereum is decentralized and secure, but not scalable.

An approach to solving the scalability trilemma is the idea of blockchain sharding, where we split the entire state of the network into partitions called shards that contain their own independent piece of state and transaction history. In this system, certain nodes would process transactions only for certain shards, allowing the throughput of transactions processed in total across all shards to be much higher than having a single shard do all the work as the main chain does now.

Basic Sharding Idea and Design
------------------------------

A sharded blockchain system is made possible by having nodes store “signed metadata” in the main chain of latest changes within each shard chain. Through this, we manage to create a layer of abstraction that tells us enough information about the global, synced state of parallel shard chains. These messages are called **collation headers**, which are specific structures that encompass important information about the chainstate of a shard in question. Collations are created by actors known as **proposer nodes** or _collators_ that are randomly tasked into packaging transactions and “selling” them to validator nodes that are then tasked into adding these collations into particular shards through a **proof of stake** system in a designated period of time.

These collations are holistic descriptions of the state and transactions on a certain shard.  A collation header contains the following information:

- Information about what shard the collation corresponds to (let’s say shard 10)
- Information about the current state of the shard before all transactions are applied
- Information about what the state of the shard will be after all transactions are applied

For detailed information on protocol primitives including collations, see: [Protocol Primitives](#protocol-primitives). We will have two types of nodes that do the heavy lifting of our sharding logic: **proposers and validators**. The basic role of proposers is to fetch pending transactions from the txpool, execute any state logic or computation, wrap them into collations, and submit them along with an ETH deposit to a **proposals pool**.

![proposers](https://yuml.me/8a367c37.png)

Validators then subscribe to updates in this proposals pool and accept collations that offer the highest payouts. Once validators are selected to add collations to a shard chain by adding their headers to a smart contract, and do so successfully, they get paid by the deposit the proposer offered.

To recap, the role of a validator is reach consensus through Proof of Stake on collations they receive in the period they are assigned to. This consensus will involve validation and data availability proofs of collations proposed to them by proposer nodes, along with validating collations from the immediate past (See: [Windback](#enforced-windback)).

When processing collations, proposer nodes download the merkle branches of the state that transactions within their collations need. In the case of cross-shard transactions, an access list of the state along with transaction receipts are required as part of the transaction primitive (See: [Protocol Primitives](#protocol-primitives)). Additionally, these proposers need to provide proofs of availability and validity when submitting collations for “sale” to validators. This submission process is akin to the current transaction fee open bidding market where miners accept the transactions that offer the most competitive (highest) transaction fees first. This abstract separation of concerns between validators and proposers allows for more computational efficiency within the system, as validators will not have to do the heavy lifting of state execution and focus solely on consensus through fork-choice rules.

When deciding and signing a proposed, valid collation, collators have the responsibility of finding the **longest valid shard chain within the longest valid main chain**.

In this new protocol, a block is valid when

- Transactions in all collations are valid
- The state of collations after the transactions is the same as what the collation headers specified

Given that we are splitting up the global state of the Ethereum blockchain into shards, new types of attacks arise because fewer hash power is required to completely dominate a shard. This is why the **source of randomness** that assigns validators and the fixed period period of time each validator has on a particular shard is critical to ensuring the integrity of the system.

The Ethereum Wiki’s [Sharding FAQ](https://github.com/ethereum/wiki/wiki/Sharding-FAQ) suggests random sampling of validators on each shard. The goal is so these validators will not know which shard they will get in advance. Every shard will get assigned a bunch of collators and the ones that will actually be validating transactions will be randomly sampled from that set. Otherwise, malicious actors could concentrate hash power into a single shard and try to overtake it (See: [1% Attack](https://medium.com/@icebearhww/ethereum-sharding-and-finality-65248951f649)).

Casper Proof of Stake (Casper [FFG](https://arxiv.org/abs/1710.09437) and [CBC](https://arxiv.org/abs/1710.09437)) makes this quite trivial because there is already a set of global validators that we can select validator nodes from. The source of randomness needs to be common to ensure that this sampling is entirely compulsory and can’t be gamed by the validators in question.

In practice, the first phase of sharding will not be a complete overhaul of the network, but rather an implementation through a smart contract on the main chain known as the **Validator Manager Contract**. Its responsibility is to manage shards and the sampling of proposed validators from a global validator set and will take responsibility for the global reconciliation of all shard states.

Among its basic responsibilities, the **VMC** will be responsible for reconciling validators across all shards, and will be in charge of pseudorandomly samping validators from a validator set of people that have staked ETH into the contract. The VMC will also be responsible for providing immediate collation header verification that records a valid collation header hash on-chain. In essence, sharding revolves around being able to store proofs of shard states on-chain through this smart contract.

The idea is that validators will be assigned to propose collations for only a certain timeframe, known as a **period** which we will define as a fixed number of blocks on the main chain. In each period, there can only be at most one valid collation per shard.

Roadmap Phases
==============
Prysmatic Labs’ implementation will follow parts of the roadmap outlined by Vitalik in his [Sharding FAQ](https://github.com/ethereum/wiki/wiki/Sharding-FAQ) to roll out a working version of quadratic sharding, with a few modifications on our releases.

1. **Phase 1:** Basic VMC shard system with no cross-shard communication along with a proposer + validator node architecture
2. **Phase 2:** Receipt-based, cross-shard communication
3. **Phase 3:** Require collation headers to be added in as uncles instead of as transactions
4. **Phase 4:** Tightly-coupled sharding with data availability proofs and robust security

To concretize these phases, we will be releasing our implementation of sharding for the geth client as follows:

The Ruby Release: Local Network
-------------------------------
Our current work is focused on creating a localized version of phase 1, quadratic sharding that would include the following:

- A minimal, **validator client** system that will deploy a **Validator Manager Contract** to a locally running geth node
- Ability to deposit ETH into the validator manager contract through the command line and to be selected as a validator by the local **VMC** in addition to the ability to withdraw the ETH staked
- A **proposer node client** and Cryptoeconomic incentive system for proposer nodes to listen for pending tx’s, create collations, and submit them along with a deposit to validator nodes in the network
- A simple command line util to **simulate pending transactions** of different types posted to the local geth node’s txpool for the local collation proposer to begin proposing collation headers
- Ability to inspect the shard states and visualize the working system locally through the command line

We will forego many of the security considerations that will be critical for testnet and mainnet release for the purposes of demonstration and local network execution as part of the Ruby Release (See: [Security Considerations Not Included in Ruby](#not-included-in-ruby-release)).

ETA: To be determined

The Sapphire Release: Ropsten Testnet
-------------------------------------
Part 1 of the **Sapphire Release** will focus around getting the **Ruby Release** polished enough to be live on an Ethereum testnet and manage a set of validators effectively processing collations through the **on-chain VMC**. This will require a lot more elaborate simulations around the safety of the pseudorandomness behind the validator assignments in the VMC and stress testing against DDoS attacks. Additionally, it will be the first release to have real users proposing collations concurrently along with validators that can accept these proposals and add their headers to the VMC.

Part 2 of the **Sapphire Release** will focus on implementing a cross-shard transaction mechanism via two-way pegging and the receipts system (as outlined in [Beyond Phase 1](#beyond-phase-1)) and getting that functionality ready to run on a **local, private network** as an extension to the Ruby Release.

ETA: To be determined

The Diamond Release: Ethereum Mainnet
-------------------------------------
The **Diamond Release** will reconcile the best parts of the previous releases and deploy a full-featured, cross-shard transaction system through a Validator Manager Contract on the Ethereum mainnet. As expected, this is the most difficult and time consuming release on the horizon for Prysmatic Labs. We plan on growing our community effort significantly over the first few releases to get all hands-on deck preparing for real ether to be staked in the VMC.

The Diamond Release should be considered the production release candidate for sharding Ethereum on the mainnet.

ETA: To Be

Go-Ethereum Sharding Alpha Implementation
=========================================
Prysmatic Labs will begin by focusing its implementation entirely on the **Ruby Release** from our roadmap. We plan on being as pragmatic as possible to create something that can be locally run by any developer as soon as possible. Our initial deliverable will center around a command line tool that will serve as an entrypoint into a validator sharding client that allows for staking, a proposer client that allows for simple state execution and collation proposals, and processing of transactions into shards through on-chain verification via the Validator Manager Smart Contract.

Here is a full reference spec explaining how our initial system will function:

System Architecture
-------------------
Our implementation revolves around 5 core components:

- A **locally-running geth node** that spins up an instance of the Ethereum blockchain
- A **validator manager contract (VMC)** that is deployed onto this blockchain instance
- A **validator client** that connects to the running geth node through JSON-RPC, provides bindings to the VMC, and listens for incoming collation proposals
- A **proposer client** that is tasked with state execution, processing pending tx’s from the Geth node, and creates collations that are then broadcast to validators via a wire protocol
- A **user** that will interact with the sharding client to become a validator and automatically process transactions into shards through the sharding client’s VMC bindings.

Our initial implementation will function through simple command line arguments that will allow a user running the local geth node to deposit ETH into the VMC and join as a validator that is automatically assigned to a certain period. We will also launch a proposer client that will create collations from the geth node’s tx pool and submit them to validators for them to add their headers to the VMC.

A basic, end-to-end example of the system is as follows:

1. _**A User Starts a Validator Client and Deposits 100ETH into the VMC:**_ the sharding client connects to a locally running geth node and asks the user to confirm a deposit from his/her personal account.

2. _**Validator Client Connects & Listens to Incoming Headers from the Geth Node and Assigns User as Validator on a Shard per Period:**_ the validator is selected for the current period and must accept collations from proposer nodes that offer the best prices.

3. _**Concurrently, the Proposer Client Processes and Executes Pending tx’s from the Geth Node:**_ the proposer client will create valid collations and submit them to validators on an open bidding system.

4. _**Validators Will Select Collation Proposals that Offer Highest Payout:**_ Validators listen to collation headers on a certain shard with high deposit sizes and sign them.

5. _**The Validator Adds Collation Headers Through the VMC:**_ the validator client calls the add_header function in the VMC and to do on-chain verification of the countersigned and accepted collation header immediately.

6. _**The User Will be Selected as Validator Again on the VMC in a Different Period or Can Withdraw His/Her Stake from the Validator Pool:**_ the user can keep staking and adding incoming collation headers and restart the process, or withdraw his/her stake and be removed from the VMC validator set.

Now, we’ll explore our architecture and implementation in detail as part of the go-ethereum repository.

System Start and User Entrypoint
--------------------------------
Our Ruby Release will require users to start a local geth node running a localized, private blockchain to deploy the **VMC** into. Then, users can spin up a validator client as a command line entrypoint into geth while the node is running as follows:

```
$ geth validator --deposit 100eth --password ~/Desktop/password.txt
```

If it is the first time the client runs, it will deploy a new **VMC** into the local chain and establish a JSON-RPC connection to interact with the node directly. The `--deposit` flag tells the sharding client to automatically unlock the user’s keystore and begin depositing ETH into the VMC to become a validator.

If the initial deposit is successful, the sharding client will launch a **local, transaction simulation generator**, which will queue transactions into the txpool for the geth node to process that can then be added into collations on a shard.

Concurrently, a user will need to launch a **proposer client** that will start processing transactions into collations that can then be “sold” to validators by including a cryptographic proof of an eth deposit in their unsign collation headers. This proof is a guarantee of a state change in the validator’s account balance for accepting to sign the incoming collation header. The proposer client can also be initialized as follows in a separate process:

```
geth proposer --password ~/Desktop/password.txt
```

Back to the validators, the validator client will begin its main loop, which involves the following steps:

1. _**Subscribe to Incoming Block Headers:**_ the client will begin by issuing a subscription over JSON-RPC for block headers from the running geth node.

2. _**Check Shards for Eligible Proposer:**_ On incoming headers, the client will interact with the VMC to check if the current validator is an eligible validator for an upcoming periods (only a few minutes notice)

3. _**If Validator is Selected, Fetch Proposals from Proposal Nodes and Add Collation Headers to VMC:**_ Once a validator is selected, he/she only has a small timeframe to add collation headers through the VMC, so he/she looks for proposals from proposer nodes and accepts those that offer the highest payouts. The validator then countersigns the collation header, receives the full collation with its body after signing, and adds it to the VMC through PoS consensus.

4. _**Supernode Reconciles and Adds to Main Chain:**_ Supernodes that will be responsible for reconciling global state across shards into the main Ethereum blockchain. They are tasked with observing the state across the whole galaxy of shards and adding blocks to the canonical PoW main chain. Proposers get rewarded to their coinbase address for inclusion of a block (also known as a collation subsidy).

5. _**If User Withdraws, Remove from Validator Set:**_ A user can choose to stop being a validator and then his/her ETH is withdrawn from the validator set.

6. _**Otherwise, Validating Client Keeps Subscribing to Block Headers:**_ If the user chooses to keep going,

It will be the proposer client’s responsibility to listen to any new broadcasted transactions to the node and interact with validators that have staked their ETH into the VMC through an open bidding system for collation proposals. Proposer clients are the ones responsible for **state execution** of transactions in the tx pool.

![system functioning](https://i.imgur.com/C54LL1R.png)

## The Validator Manager Contract

Our solidity implementation of the Validator Manager Contract follows the reference spec outlined by Vitalik [here](https://github.com/ethereum/sharding/blob/develop/docs/doc.md).

### Necessary functionality

In our Solidity implementation, we begin with the following sensible defaults:

```javascript
// Constant values
uint constant periodLength = 5;
int constant public shardCount = 100;
// The exact deposit size which you have to deposit to become a validator
uint constant depositSize = 100 ether;
// Number of periods ahead of current period, which the contract
// is able to return the collator of that period
uint constant lookAheadPeriods = 4;
```

Then, the 4 minimal functions required by the VMC are as follows:
