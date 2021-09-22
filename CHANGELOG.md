# Changelog

## [2.0.0]

### Added

- Full Altair support. [Learn more about Altair.](https://github.com/ethereum/annotated-spec/blob/8473024d745a3a2b8a84535d57773a8e86b66c9a/altair/beacon-chain.md)

#### Metrics

**Beacon chain node**

| Metric                                           | Description                                                                                           | References  |
|--------------------------------------------------|-------------------------------------------------------------------------------------------------------|-------------|
| `p2p_message_ignored_validation_total`           | Count of messages that were ignored in validation                                                     | #9538       |
| `beacon_current_active_validators`               | Current total active validators                                                                       | #9623       |
| `beacon_processed_deposits_total`                | Total number of deposits processed                                                                    | #9623       |
| `sync_head_state_miss`                           | The number of sync head state requests that are not present in the cache                              | #9422       |
| `sync_head_state_hit`                            | The number of sync head state requests that are present in the cache                                  | #9422       |
| `total_effective_balance_cache_miss`             | The number of get requests that are not present in the cache                                          | #9456       |
| `total_effective_balance_cache_hit`              | The number of get requests that are present in the cache                                              | #9456       |
| `sync_committee_index_cache_miss_total`          | The number of committee requests that aren't present in the sync committee index cache                | #9317       |
| `sync_committee_index_cache_hit_total`           | The number of committee requests that are present in the sync committee index cache                   | #9317       |
| `next_slot_cache_hit`                            | The number of cache hits on the next slot state cache                                                 | #8357       |
| `next_slot_cache_miss`                           | The number of cache misses on the next slot state cache                                               | #8357       |
| `validator_entry_cache_hit_total`                | The number of cache hits on the validator entry cache                                                 | #9155 #9310 |
| `validator_entry_cache_miss_total`               | The number of cache misses on the validator entry cache                                               | #9155 #9310 |
| `validator_entry_cache_delete_total`             | The number of cache deletes on the validator entry cache                                              | #9310       |
| `saved_sync_committee_message_total`             | The number of saved sync committee message total                                                      | #9203       |
| `saved_sync_committee_contribution_total`        | The number of saved sync committee contribution total                                                 | #9203       |
| `libp2p_peers`                                   | Tracks the total number of libp2p peers                                                               | #9623       |
| `p2p_status_message_missing`                     | The number of attempts the connection handler rejects a peer for a missing status message             | #9505       |
| `p2p_sync_committee_subnet_recovered_broadcasts` | The number of sync committee messages that were attempted to be broadcast with no peers on the subnet | #9390       |
| `p2p_sync_committee_subnet_attempted_broadcasts` | The number of sync committee that were attempted to be broadcast                                      | #9390       |
| `p2p_subscribed_topic_peer_total`                | The number of peers subscribed to topics that a host node is also subscribed to                       | #9538       |
| `saved_orphaned_att_total`                       | Count the number of times an orphaned attestation is saved                                            | #9442       |


### Changed

- Altair object keys in beacon-chain/db/kv are prefixed with "altair". BeaconBlocks and BeaconStates are the only objects affected by database key changes for Altair. This effects any third party tooling direct querying Prysm's beaconchain.db.

#### Metrics

**Beacon chain node**
| Metric                | Old Name             | Description                                          | References |
|-----------------------|----------------------|------------------------------------------------------|------------|
| `beacon_reorgs_total` | `beacon_reorg_total` | Count the number of times a beacon chain has a reorg | #9623      |

### Deprecated

TODO: Deprecated features

### Removed
Note: Removed flags will block starting up with an error "flag provided but not defined:". 
Please check that you are not using any of the removed flags in this section!

- Prysm's standalone slasher application (cmd/slasher) has been fully removed. Use the `--slasher` flag with a beacon chain node for full slasher functionality.
- Removed `--disable-blst` (beacon node and validator). [blst](https://github.com/supranational/blst) is the only BLS library offered for Prysm.
- Removed `--disable-sync-backtracking` and `--enable-sync-backtracking` (beacon node). This feature has been released for some time. See #7734.
- Removed `--diable-pruning-deposit-proofs` (beacon node). This feature has been released for some time. See #7504.
- Removed `--disable-eth1-data-majority-vote` (beacon node). This feature is no longer in use in Prysm. See #6766, #8298.
- Removed `--proposer-atts-selection-using-max-cover` (beacon node). This feature has been released for some time. See #8353.
- Removed `--update-head-timely` (beacon node). This feature was released in v1.4.4. See #8412.
- Removed `--enable-optimized-balance-update` (beacon node). This feature was released in v1.4.4. See #9225.
- Removed kafka support, removed `--kafka-url` (beacon node). See #9470.
- Removed tools/faucet. Use the faucet in [prysmaticlabs/periphery](https://github.com/prysmaticlabs/periphery/tree/c2ac600882c37fc0f2a81b0508039124fb6bcf47/eth-faucet) if operating a testnet faucet server.

### Fixed

- Database lock contention improved in block database operations. See #9428.

### Security

TODO: any security highlights.