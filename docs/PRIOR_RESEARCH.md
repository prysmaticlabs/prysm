# Beacon Chain Research Synopsis

This doc will summarize some historical discussions and roadmap updates around integrating Casper/Sharding through a beacon chain and what it means for Ethereum's end-game. 

### Research Notes Leading Up To This

-   [Offchain Collation Headers](https://ethresear.ch/t/offchain-collation-headers/1679)
-   [RANDAO Beacon Exploitability Part. 1](https://ethresear.ch/t/rng-exploitability-analysis-assuming-pure-randao-based-main-chain/1825/9)
-   [Leaderless k-of-n Random Beacon](https://ethresear.ch/t/leaderless-k-of-n-random-beacon/2046)
-   [Two Ways to do Cross Links](https://ethresear.ch/t/two-ways-to-do-cross-links/2074)
-   [Registrations, Shard Count, and Shuffling](https://ethresear.ch/t/registrations-shard-count-and-shuffling/2129)
-   [Committee Based Sharded Casper](https://ethresear.ch/t/committee-based-sharded-casper/2197)
-   [Attestation Committee Based Full PoS Chains](https://ethresear.ch/t/attestation-committee-based-full-pos-chains/2259)

The beacon chain idea emerged from the research around notarization of shard information by a committee of notaries, similar to a committee of validators in Casper. The key difference in sharding, however, is the pseudorandomness generation required for fast reshuffling of actors across shards within the system.

Piggybacking off the VRF (verifiable random function) research alongside the ideas of BLS signatures put forth by DFINITY, there are many elements that can be taken to create a sidechain that would potentially merge sharding/casper :heart:.

## The Beacon Chain

### A Sidechain Instead of a Sharding Manager Contract

When handling the sharded system via smart contract on the Ethereum mainchain, we were able to derive implicit finality via the transactions that submit a collation header into the contract that would then be mined onto a block in the mainchain. However, we were _bounded_ by gas and the current functioning of the EVM 1.0. That is, the number of shards realistically could only grow as much as the sharding manager contract could handle as a load of incoming transactions. 

This system is also more affected by hard-forks and changes occurring in the main Ethereum network. Moreover, the incoming integration of a hybrid Casper PoS system would create a complicated co-existence of two types of validators: namely Casper and sharding validators. Signature verification, in particular, is an extremely expensive operation if done entirely through a smart contract on the mainchain, creating inherent bottlenecks in a hybrid system.

Instead, we propose the creation of a _sidechain_ known as a **beacon chain** that has links to the mainchain by containing hashes of canonical mainchain blocks within its own block construction.

### Desiderata

There are a few important traits we will include in this sidechain construct that are particularly important for sharding.

-   Block references to the main chain
-   Full Proof of Stake via Casper FFG
-   RANDAO for random selection of committees
-   One-Way Deposits
-   Ability to store verifiable metadata of occurrences across shards (we refer to this information as a cross-link)

In Vitalik Buterin's [words](https://notes.ethereum.org/SCIg8AH5SA-O4C1G1LYZHQ?both):

> A cross-link is a special type of transaction that says “here is the hash of some recent block on shard X. Here are signatures from at least 2/3 of some randomly selected sample of M validators (eg. M = 1024) that attest to the validity of the cross-link”. Every shard (eg. there might be 4000 shards total) is itself a PoS chain, and the shard chains are where the transactions and accounts will be stored. The cross-links serve to “confirm” segments of the shard chains into the main chain, and are also the primary way through which the different shards will be able to talk to each other.

The main purpose of beacon chain is to handle these shard cross-links as well as the set of validators that are locked into the system. This initial set is seeded by users burning 32ETH into a contract on the mainchain and specifying their public key, which can then be verified by the beacon chain which interacts with this contract.

The exact specification of the beacon chain is a work-in-progress being written [here](https://notes.ethereum.org/SCIg8AH5SA-O4C1G1LYZHQ?both). Instead of paraphrasing its spec, we will elaborate on the research that led to this point and what it means moving forward.

## Research History

Recall that the **Sharding Manager Contract** was originally going to be used for management of the sharded system.

> Whole point of this is to give a background and better understanding of the system...I think I won't follow the structure below in the final writing but will help as an outline.

### Limitations of the Sharding Manager Contract

Using a Sharding Manager Contract, albeit attractive from a development standpoint, poses the following challenges:

-   miners on the main chain can censor transactions
-   the number of shards is limited by gas costs of writes to the storage of this contract
-   any upgrades to this contract/sharding system would require a hard fork on the main chain

#### Offchain Collation Headers

[ETHResearch Link](https://ethresear.ch/t/offchain-collation-headers/1679)

One idea that naturally arose was to store collation headers offchain, allowing for more shards to exist as there would be no processing bottleneck and faster finality guarantees. Reminiscent of DFINITY's chain, this process could be done through a construct known as a **random beacon side chain** that is pegged to the main chain via checkpoints on main chain blocks.

Reminiscent of plasma chains without the exit mechanism according to Justin Drake, this construct would provide a solid ground for experimentation of shard designs without modifications to the main chain. The beacon chain would provide the pseudorandomness required for committee selection of the sharding system, through BLS distributed key generation, as well as better finality given that shards only care about finalized deposits from the main chain.

Management and submission of collation headers can now be done within each respective shard, with the beacon chain only serving as a coordination device for summarizing what was voted on within each shard. That is, the beacon chain would be responsible for handling what we call a **cross-link**, which is a piece of metadata summarizing which collations were voted on as canonical within shards, who voted on these collations, and what are their blob merkle roots.

#### Randomness in Committee Selections

[ETHResearch Link](https://ethresear.ch/t/rng-exploitability-analysis-assuming-pure-randao-based-main-chain/1825/9)

While DFINITY's beacon chain uses the BLS Signature scheme integrated into something called Threshold Relay for distributed randomness, this is not needed for the random beacon chain construct we are building for sharding. That is, there are other satisfactory ways to achieve randomness that make more sense for sharding.

Ethereum's sharding beacon chain will use a system called **RANDAO** which is a Decentralized Autonomous Organization (a DAO) where randomness is generated by participants contributing a value to a "hash-onion" in the system 

$$H(H(H(.....S.....)))$$

where participants creating a block have to reveal the pre-image (the value before the hash) of their commitment value and update the current commitment value to this pre-image. In a recursive manner, the next participants will have to do the same and reveal their pre-images when creating a block. This system updates a random value $R$ during each iteration by taking the $XOR$ of it with the revealed pre-image. This value, $R$, is then used for randomization of committee selection. For sharding in particular, Justin mentioned the source of randomness can be this global $R$ value also $XOR$ with the proposer's pre-image commitment for that particular shard, restricting the visibility of $R\_{shard}$ to only proposers participating in that shard.

#### Leaderless Random Beacon - Alternative to RANDAO

[ETHResearch Link](https://ethresear.ch/t/leaderless-k-of-n-random-beacon/2046)

The downside of RANDAO is the need for a leader in each step of the hash pre-image reveal step for determining the next value of $R$. In an alternative approach, we have a "committee of size $n$ generate random numbers if $k$ participants generate correctly" (Justin Drake). In this approach, we have every participant commit a temporary secret key, public key pair and form a polynomial via point interpolation. At a reveal phase of the protocol, k-of-n polynomial encrypted shares. At a reveal phase, $k$ participants reveal their private keys and it is then easy to check which ones did not reveal correctly. Even then, we can reconstruct an appropriate candidate polynomial using the revealed keys. Then, the randomness becomes random output to be the "sum of the secret keys for which the corresponding participants committed correctly" as explained by Justin.

### Shard Metadata & Finality

[ETHResearch Link](https://ethresear.ch/t/two-ways-to-do-cross-links/2074)

The main idea behind sharding is to leverage this beacon chain construct for consensus on **cross-links** which are the heart of the sharding spec. In a way, cross-links are metadata that summarize the latest occurrences on shards. That is, they compress the results of PoS consensus from each shard into a simple message that uses proposer signatures to confirm this information. They are called **cross-links** because they will eventually be the way shards can communicate between each other, as they are linked to the finality of the main chain.

#### Registrations and Committee Reshuffling

[ETHResearch Link](https://ethresear.ch/t/registrations-shard-count-and-shuffling/2129)

Leading up to the beacon chain spec, a few proposals for structuring committees were being explored, including this one by Justin Drake. We refer to participants in the Ethereum protocol 2.0 (Casper + Sharding) as validators which can be in one of three states: either pending_registration, registered, or pending_deregistration (Justin Drake).

The main idea behind structuring committees of validators is that shards will be empty and uninitialized until the beacon chain reaches a certain minimum number of registered validators.

In the post, it is mentioned that

> Proposers and notaries are shuffled (via pseudo-random permutations) across shards in a staggered fashion and at a constant rate. Proposers are assigned to shards for 2^19 periods (~30 days) and the oldest proposer from each shard are shuffled every 2^(19 - 10) periods. Notaries are assigned to a shard for 2^7 periods (~10 minutes) and the oldest 2^(10 - 7) notaries from each shard are shuffled every period.
>
> However, the current spec for the beacon chain mentions a fixed number for the SHARD_COUNT set to 1024.

The entire reshuffling mechanism was revamped given this fixed shard_count number. In the current beacon chain spec, it is mentioned that

> For shard crosslinks, the process is somewhat more complicated. First, we choose the set of shards active during each epoch. We want to target some fixed number of notaries per crosslink, but this means that since there is a fixed number of shards and a variable number of validators, we won’t be able to go through every shard in every epoch. Hence, we first need to select which shards we will be crosslinking in some given epoch

Additionally, the current spec forces casper validators to also be sharding validators, which enforces greater security and takes advantage of the enshrined randomness + full PoS properties of the beacon chain.
