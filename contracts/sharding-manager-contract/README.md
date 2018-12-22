## Sharding Manager Contract (Deprecated)

In the original sharding approach, the system was designed to be handled via a smart contract
known as sharding manager contract (SMC) on the main chain. This contract would allow for implicit finality
using transactions submitting shard block headers to SMC which would then be mined onto a main chain block.
However, this system is bounded by gas and the limited functionality of EVM 1.0. That is,
the number of shards realistically could only grow as much as SMC could handle.

In the current approach, SMC was deprecated for beacon chain that has links to the main chain by containing hashes of canonical main chain blocks
within its own block construction. Check out our [design doc](https://docs.google.com/document/d/1lTDUy6JwRGNE4rDKiyzaV-lxhG2SZ2F6wCthM-QCSFQ/edit?usp=sharing)
which summarizes our thoughts for replacing SMC with a minimal viable beacon chain and merge aspects of our current work in geth-sharding into this
new beacon chain that we can use for demonstration purposes.