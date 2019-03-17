# Required Readings

This document serves as an outline of the required readings to catch up with and contribute to the implementation for of the Ethereum 2.0 (Serenity) protocol. Whether you are an expert at Ethereum and want to contribute, or are a beginner and want to take the time to learn the concepts, this will serve as a comprehensive set of readings and links to help you up to speed.

## Required Blockchain & Ethereum Knowledge

**Blockchain Fundamentals**

-   [What is Blockchain Technology? A Step by Step Guide for Beginners](https://blockgeeks.com/guides/what-is-blockchain-technology/)
-   [What is Bitcoin? A Step by Step Guide for Beginners](https://blockgeeks.com/guides/what-is-bitcoin/)
-   [The Science Behind Cryptocurrencies' Cryptography](https://blockgeeks.com/guides/cryptocurrencies-cryptography/)
-   [The Ins and Outs of Cryptographic Hash Functions](https://blockgeeks.com/guides/cryptographic-hash-functions/)
-   [Blockchain Glossary from A-Z](https://blockgeeks.com/guides/blockchain-glossary-from-a-z/)
-   [Blockchain Addresses 101: What Are They?](https://blockgeeks.com/guides/blockchain-address-101/)

**Ethereum**

-   [What is Ethereum?](http://ethdocs.org/en/latest/introduction/what-is-ethereum.html)
-   [How Does Ethereum Work Anyway?](https://medium.com/@preethikasireddy/how-does-ethereum-work-anyway-22d1df506369)
-   [Ethereum Introduction](https://github.com/ethereum/wiki/wiki/Ethereum-introduction)
-   [Ethereum Frequently Asked Questions](https://github.com/ethereum/wiki/wiki/FAQs)
-   [What is Hashing?](https://blockgeeks.com/guides/what-is-hashing/)
-   [Hashing Algorithms and Security](https://www.youtube.com/watch?v=b4b8ktEV4Bg)
-   [Understanding Merkle Trees](https://www.codeproject.com/Articles/1176140/Understanding-Merkle-Trees-Why-use-them-who-uses-t)
-   [Ethereum White Paper](https://github.com/ethereum/wiki/wiki/White-Paper)
-   [Ethereum Block Architecture](https://ethereum.stackexchange.com/questions/268/ethereum-block-architecture/6413#6413)
-   [Ethereum Beige Paper](https://github.com/chronaeon/beigepaper/blob/master/beigepaper.pdf)
-   [What is an Ethereum Token?](https://blockgeeks.com/guides/ethereum-token/)
-   [What is Ethereum Gas?](https://blockgeeks.com/guides/ethereum-gas-step-by-step-guide/)

**Ethereum Client Implementations**
-   [Geth](https://github.com/ethereum/go-ethereum) (known also as go-ethereum) is the Golang implementation of the Ethereum protocol
-   [Parity](https://github.com/paritytech/parity) the fastest and most performant implementation - written in Rust
-   [Trinity](https://github.com/ethereum/py-evm/tree/master/trinity) new project implements Ethereum in Python
-   [Cpp-Ethereum](https://github.com/ethereum/cpp-ethereum) a C++ implementation of Ethereum

**Consensus**

-   [Bitcoin Original White Paper](https://bitcoin.org/bitcoin.pdf)
-   [Basic Primer: Blockchain Consensus](https://blockgeeks.com/guides/blockchain-consensus/)
-   [Understanding Blockchain Fundamentals: Byzantine Fault Tolerance](https://medium.com/loom-network/understanding-blockchain-fundamentals-part-1-byzantine-fault-tolerance-245f46fe8419)
-   [Understanding Blockchain Fundamentals: Proof of Work vs. Proof of Stake](https://medium.com/loom-network/understanding-blockchain-fundamentals-part-2-proof-of-work-proof-of-stake-b6ae907c7edb)
-   [Proof of Work vs. Proof of Stake](https://blockgeeks.com/guides/proof-of-work-vs-proof-of-stake/)
-   [Proof of Stake FAQ](https://github.com/ethereum/wiki/wiki/Proof-of-Stake-FAQ)
-   [How Does Ethereum Mining Work?](https://www.coindesk.com/information/ethereum-mining-works/)
-   [ETHash Algorithm](https://github.com/ethereum/wiki/wiki/Ethash)

**Ethereum Virtual Machine**

-   [What is the Ethereum Virtual Machine?](https://themerkle.com/what-is-the-ethereum-virtual-machine/)
-   [Ethereum VM](https://medium.com/@jeff.ethereum/go-ethereums-jit-evm-27ef88277520)
-   [Ethereum Protocol Subtleties](https://github.com/ethereum/wiki/wiki/Subtleties)
-   [Awesome Ethereum Virtual Machine](https://github.com/ethereum/wiki/wiki/Ethereum-Virtual-Machine-(EVM)-Awesome-List)

**Ethereum-flavored WebAssembly**

-   [Ewasm background, motivation, goals, and design](https://github.com/ewasm/design)
-   [The current Ewasm spec](https://github.com/ewasm/design/blob/master/eth_interface.md)
-   [Latest Ewasm community call including live demo of the testnet](https://www.youtube.com/watch?v=apIHpBSdBio)
-   [Why eWASM? by Alex Beregszaszi](https://www.youtube.com/watch?v=VF7f_s2P3U0)
-   [Panel: entire eWASM team discussion and Q&A](https://youtu.be/ThvForkdPyc?t=119)
-   [Ewasm community meetup at ETHBuenosAires](https://www.youtube.com/watch?v=qDzrbj7dtyU)

**Smart Contracts, dApps, and Cryptoeconomics**

-   [What are dApps? The New Decentralized Future](https://blockgeeks.com/guides/dapps/)
-   [How to Learn Solidity](https://blockgeeks.com/guides/solidity/)
-   [Ethereum Development Tutorial](https://github.com/ethereum/wiki/wiki/Ethereum-Development-Tutorial)
-   [What is Cryptocurrency Game Theory?](https://blockgeeks.com/guides/cryptocurrency-game-theory/)
-   [What is Cryptoeconomics?](https://blockgeeks.com/guides/what-is-cryptoeconomics/)
-   [Mechanism Design for Cryptoeconomic Applications](https://medium.com/blockchannel/a-crash-course-in-mechanism-design-for-cryptoeconomic-applications-a9f06ab6a976)
-   [Cryptoeconomics: An Introduction](https://cryptoeconomics.study/)

**Peer-to-Peer Networking**

-   [Ethereum Peer to Peer Networking](https://github.com/ethereum/go-ethereum/wiki/Peer-to-Peer)
-   [How Does the P2P on Ethereum Work?](https://www.reddit.com/r/ethereum/comments/3918u0/how_does_the_p2p_network_on_ethereum_work/)
-   [How Does Kademlia Work?](http://gleamly.com/article/introduction-kademlia-dht-how-it-works)
-   [Kademlia Protocol](http://www.divms.uiowa.edu/~ghosh/kademlia.pdf)

## Required Sharding Knowledge

With respect to knowing enough about sharding, we will cover the requirements for both part-time and core contributors to Prysmatic Labs.

### For Part Time Contributors

-   [Blockchain Scalability: Why?](https://blockgeeks.com/guides/blockchain-scalability/)
-   [What Are Ethereum Nodes and Sharding](https://blockgeeks.com/guides/what-are-ethereum-nodes-and-sharding/)
-   [How to Scale Ethereum: Sharding Explained](https://medium.com/prysmatic-labs/how-to-scale-ethereum-sharding-explained-ba2e283b7fce)
-   [Sharding FAQ](https://github.com/ethereum/wiki/wiki/Sharding-FAQ)
-   [Sharding Introduction: R&D Compendium](https://github.com/ethereum/wiki/wiki/Sharding-introduction-R&D-compendium)

### For Core Contributors

**Fundamentals**

-   [Blockchain Scalability: Why?](https://blockgeeks.com/guides/blockchain-scalability/)
-   [What Are Ethereum Nodes and Sharding](https://blockgeeks.com/guides/what-are-ethereum-nodes-and-sharding/)
-   [How to Scale Ethereum: Sharding Explained](https://medium.com/prysmatic-labs/how-to-scale-ethereum-sharding-explained-ba2e283b7fce)
-   [Sharding FAQ](https://github.com/ethereum/wiki/wiki/Sharding-FAQ)
-   [Sharding Introduction: R&D Compendium](https://github.com/ethereum/wiki/wiki/Sharding-introduction-R&D-compendium)

After reading the Sharding FAQ, it is important to understand the minimal implementation notes and the research that went into writing it up.

**Serenity Concepts and Notes**

-   [Sharding Concepts Mental Map](https://www.mindomo.com/zh/mindmap/sharding-d7cf8b6dee714d01a77388cb5d9d2a01)
-   [Taiwan Sharding Workshop Notes](https://hackmd.io/s/HJ_BbgCFz#%E2%9F%A0-General-Introduction)
-   [Sharding Research Compendium](http://notes.ethereum.org/s/BJc_eGVFM)
-   [Torus Shaped Sharding Network](https://ethresear.ch/t/torus-shaped-sharding-network/1720/8)
-   [General Theory of Sharding](https://ethresear.ch/t/a-general-theory-of-what-quadratically-sharded-validation-is/1730/10)
-   [Sharding Design Compendium](https://ethresear.ch/t/sharding-designs-compendium/1888/25)

**Serenity Research Posts in Order**

-   [Sharding v2.1 Spec](https://notes.ethereum.org/SCIg8AH5SA-O4C1G1LYZHQ)
-   [Casper/Sharding/Beacon Chain FAQs](https://notes.ethereum.org/9MMuzWeFTTSg-3Tz_YeiBA?view)
-   [RETIRED! Sharding Phase 1 Spec](https://ethresear.ch/t/sharding-phase-1-spec-retired/1407/92)
-   [Exploring the Proposer/Collator Spec and Why it Was Retired](https://ethresear.ch/t/exploring-the-proposer-collator-split/1632/24)
-   [The Stateless Client Concept](https://ethresear.ch/t/the-stateless-client-concept/172/4)
-   [Shard Chain Blocks vs. Collators](https://ethresear.ch/t/shard-chain-blocks-vs-collators/429)
-   [Ethereum Concurrency Actors and Per Contract Sharding](https://ethresear.ch/t/ethereum-concurrency-actors-and-per-contract-sharding/375)
-   [Future Compatibility for Sharding](https://ethresear.ch/t/future-compatibility-for-sharding/386)
-   [Fork Choice Rule for Collation Proposal Mechanisms](https://ethresear.ch/t/fork-choice-rule-for-collation-proposal-mechanisms/922/8)
-   [State Execution](https://ethresear.ch/t/state-execution-scalability-and-cost-under-dos-attacks/1048)
-   [Fast Shard Chains With Notarization](https://ethresear.ch/t/as-fast-as-possible-shard-chains-with-notarization/1806/2)
-   [RANDAO Notary Committees](https://ethresear.ch/t/fork-free-randao/1835/3)
-   [Safe Notary Pool Size](https://ethresear.ch/t/safe-notary-pool-size/1728/3)
-   [Cross Links Between Main and Shard Chains](https://ethresear.ch/t/cross-links-between-main-chain-and-shards/1860/2)

**Serenity Talk from Conferences & Meet Ups in Order**
-   [Sharding Presentation by Vitalik from IC3-ETH Bootcamp](https://vod.video.cornell.edu/media/Sharding+-+Vitalik+Buterin/1_1xezsfb4/97851101)
-   [Latest Research and Sharding by Justin Drake from Tech Crunch](https://www.youtube.com/watch?v=J6xO7DH20Js)
-   [Beacon Casper Chain by Vitalik and Justin Drake](https://www.youtube.com/watch?v=GAywmwGToUI)
-   [Proofs of Custody by Vitalik and Justin Drake](https://www.youtube.com/watch?v=jRcS9D_gw_o)
-   [So You Want To Be a Casper Validator by Vitalik](https://www.youtube.com/watch?v=rl63S6kCKbA)
-   [Ethereum Sharding from EDCon by Justin Drake](https://www.youtube.com/watch?v=J4rylD6w2S4)
-   [Casper CBC and Sharding by Vlad Zamfir](https://www.youtube.com/watch?v=qDa4xjQq1RE&t=1951s)
-   [Casper FFG in Depth by Carl](https://www.youtube.com/watch?v=uQ3IqLDf-oo)
-   [Ethereum & Scalability Technology from Asia Pacific ETH meet up by Hsiao Wei](https://www.youtube.com/watch?v=GhuWWShfqBI)

## Necessary Go Knowledge & Readings

-   [The Go Programming Language (Only Recommended Book)](https://www.amazon.com/Programming-Language-Addison-Wesley-Professional-Computing/dp/0134190440)
-   [Ethereum Development with Go](https://goethereumbook.org)
-   [How to Write Go Code](http://golang.org/doc/code.html)
-   [The Go Programming Language Tour](http://tour.golang.org/)
-   [Getting Started With Go](http://www.youtube.com/watch?v=2KmHtgtEZ1s)
-   [Go Official Website](https://golang.org/)
