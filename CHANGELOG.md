# Changelog

## [2.0.0]

This release is the largest release of Prysm to date. v2.0.0 includes support for the upcoming Altair hard fork on the mainnet Ethereum Beacon Chain.
This release consists of [380 changes](https://github.com/prysmaticlabs/prysm/compare/v1.4.4...4f31ba648977a00668cafa10bc3aa93f41fbe31e) to support Altair, improve performance of phase0 beacon nodes, and various bug fixes from v1.4.4.

### Added

- Full Altair support. [Learn more about Altair.](https://github.com/ethereum/annotated-spec/blob/8473024d745a3a2b8a84535d57773a8e86b66c9a/altair/beacon-chain.md)
- Added bootnodes from Nimbus team. #9656
- Revamped slasher implementation. The slasher functionality is no longer a standalone binary. Slasher functionality is available from the beacon node with the `--slasher` flag. Note: Running the slasher has considerably increased resource requirements. Be sure to review the latest documentation before enabling this feature. #8331
- Support for standard JSON API in the beacon node. Prysm validators continue to use Prysm's API. #7510
- Configurable subnet peer requirements. Increased minimum desired peers per subnet from 4 to 6. This can be modified with `--minimum-peers-per-subnet` in the beacon node. #9657.
- Support for go build on darwin_arm64 devices (Mac M1 chips). Cross compile for darwin_arm64 is not yet supported. #9600.
- Batch verfication of pubsub objects. This should improve pubsub processing performance on multithreaded machines. #9344
- Improved attestation pruning. This feature should improve block proposer performance and overall network attestation inclusion rates. Opt-out with `--disable-correctly-prune-canonical-atts` in the beacon node. #9444
- Active balance cache to improve epoch processing. Opt-out with `--disable-active-balance-cache` #9567

#### New Metrics

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

- Much refactoring of "util" packages into more canonical packages. Please review Prysm package structure and godocs.
- Altair object keys in beacon-chain/db/kv are prefixed with "altair". BeaconBlocks and BeaconStates are the only objects affected by database key changes for Altair. This effects any third party tooling direct querying Prysm's beaconchain.db.
- Updated Teku bootnodes. #9656
- Updated Lighthouse bootnodes. #9656
- End to end testing now collects jaeger spans #9341
- Improvements to experimental peer quality scoring. This feature is only enabled with `--enable-peer-scorer`. #8794
- Validator performance logging behavior has changed in Altair. Post-Altair hardfork has the following changes: Inclusion distance and inclusion slots will no longer be displayed. Correctly voted target will only be true if also included within 32 slots. Currectly voted head will only be true if the attestation was included in the next slot. Correctly voted source will only be true if attestation is included within 5 slots. Inactivity score will be displayed. #9589
- Increased pubsub message queue size from 256 to 600 to support larger networks and higher message volume. #9702
- The default attestation aggregation changed to the improved optimized max cover algorithm. #9684 #8365
- Prysm is passing spectests at v1.1.0 (latest available release). #9680
- `--subscribe-all-subnets` will subscribe to all attestation subnets and sync subnets in post-altair hard fork #9631.
- "eth2" is now an illegal term. If you say it or type it then something bad might happen. #9425
- Improved cache hit ratio for validator entry cache. #9310
- Reduced memory overhead during database migrations. #9298
- Improvements to beacon state writes to database. #9291

#### Changed Metrics

**Beacon chain node**
| Metric                | Old Name             | Description                                          | References |
|-----------------------|----------------------|------------------------------------------------------|------------|
| `beacon_reorgs_total` | `beacon_reorg_total` | Count the number of times a beacon chain has a reorg | #9623      |

### Deprecated

These flags are hidden from the help text and no longer modify the behavior of Prysm. These flags should be removed from user runtime configuration as the flags will eventually be removed entirely and Prysm will fail to start if a deleted or unknown flag is provided.

- `--enable-active-balance-cache` #9567
- `--correctly-prune-canonical-atts` #9576
- `--correctly-insert-orphaned-atts` #9575
- `--enable-next-slot-state-cache` #9602

### Removed
Note: Removed flags will block starting up with an error "flag provided but not defined:". 
Please check that you are not using any of the removed flags in this section!

- Prysm's standalone slasher application (cmd/slasher) has been fully removed. Use the `--slasher` flag with a beacon chain node for full slasher functionality.
- `--disable-blst` (beacon node and validator). [blst](https://github.com/supranational/blst) is the only BLS library offered for Prysm.
- `--disable-sync-backtracking` and `--enable-sync-backtracking` (beacon node). This feature has been released for some time. See #7734.
- `--diable-pruning-deposit-proofs` (beacon node). This feature has been released for some time. See #7504.
- `--disable-eth1-data-majority-vote` (beacon node). This feature is no longer in use in Prysm. See #6766, #8298.
- `--proposer-atts-selection-using-max-cover` (beacon node). This feature has been released for some time. See #8353.
- `--update-head-timely` (beacon node). This feature was released in v1.4.4. See #8412.
- `--enable-optimized-balance-update` (beacon node). This feature was released in v1.4.4. See #9225.
- Kafka support is no longer available in beacon node. This functionality was never fully completed and did not fulfill many desirable use cases. This removed the flag `--kafka-url` (beacon node). See #9470.
- Removed tools/faucet. Use the faucet in [prysmaticlabs/periphery](https://github.com/prysmaticlabs/periphery/tree/c2ac600882c37fc0f2a81b0508039124fb6bcf47/eth-faucet) if operating a testnet faucet server.
- Tooling for prior testnet contracts has been removed. Any of the old testnet contracts with `drain()` function have been removed as well. #9637
- Toledo tesnet config is removed.

### Fixed

- Database lock contention improved in block database operations. #9428
- JSON API now returns an error when unknown fields are provided. #9710
- Correctly return `epoch_transition` field in `head` JSON API events stream. #9668 #9704
- Various fixes in standard JSON API #9649
- Finalize deposits before initializing beacon node. This may improved missed proposals #9639 #9610
- JSON API returns header "Content-Length" 0 when returning empty JSON object. #9531 #9540
- Initial sync fixed when there is a very long period of missing blocks. #9450 #9452
- Fixed log statement when a web3 endpoint failover occurs. #9272
- Windows prysm.bat is fixed #9266 #9260

### Security

- You MUST update to v2.0.0 or later release before epoch 74240 or your client will fork off from the rest of the network.
- Prysm's JWT library has been updated to a maintained version of the previous JWT library. JWTs are only used in the UI. #9357


Please review our newly updated [security reporting policy](https://github.com/prysmaticlabs/prysm/blob/develop/SECURITY.md). 