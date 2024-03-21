# Prysm Changelog

See [keep a changelog v1.1.0](https://keepachangelog.com/en/1.1.0/) for guidelines on maintaining this changelog.

## Unreleased

## v5.0.0

2024-02-22

### Added

**API**

- Support [beacon_committee_selections](https://ethereum.github.io/beacon-APIs/#/Validator/submitBeaconCommitteeSelections) #13503
- `/eth/v1/beacon/deposit_snapshot` #13514

**Docker**

- Docker images now have [coreutils](https://www.gnu.org/software/coreutils/manual/html_node/index.html) pre-installed #13564

**Metrics**

- `da_waited_time_milliseconds` tracks total time waiting for data availablity check in ReceiveBlock
  #13534
- `blob_written`, `blob_disk_count`, `blob_disk_bytes` new metrics for tracking blobs on disk #13614

**Uncategorized**

- Backfill supports blob backfilling #13595
- Add mainnet deneb fork epoch config #13601

### Changed

**Database**

- `--clear-db` and `--force-clear-db` flags now remove blobs as well as beaconchain.db #13605

**Flags**

- EIP-4881 is now on by default. #13555
-
**Fork Choice**

- Updates filtering logic to match spec #13464

 Verbose signature verification is now on by default #13556

**Metrics**

- `gossip_block_arrival_milliseconds` and `gossip_block_verification_milliseconds`  measure in
  milliseconds instead of nanoseconds #13540
- `aggregate_attestations_t1` histogram buckets have been updated #13607

**p2p**

- Reduce lookahead period from 8 to 4. This reduces block batch sizes during sync to account for
  larger blocks in deneb. #13599

**Uncategorized**

- Update gohashtree to v0.0.4-beta #13569
- Various logging improvements #13571 #13582 #13561 #13573 #13598 #13608 #13611 #13502 #13627
- Improved operations during syncing #13580
- Backfill starts after initial-sync is complete #13623

### Deprecated

**Flag removal**

The following flags have been removed entirely:

- `--enable-reorg-late-blocks` #13536
- `--disable-vectorized-htr` #13537
- `--aggregate-parallel` #13538
- `--build-block-parallel` #13539
- `--enable-registration-cache`, `disable-gossip-batch-aggregation` #13606
- `--safe-slots-to-import-optimistically` #13624
- `--show-deposit-data` #13618

### Removed

**API**

- Prysm gRPC slasher endpoints are removed #13594
- Remove /eth/v1/debug/beacon/states/{state_id} #13619
- Prysm gRPC endpoints that were marked as deprecated in v4 have been removed #13600
- Remove /eth/v1/beacon/blocks/{block_id} #13628

### Fixed

**API**

- Return unaggregated if no aggregated attestations available in `GetAggregateAttestation` #13533
- Fix JWT auth checks in certain API endpoints used by the web UI #13565 #13568
- Return consensus block value in wei units #13575
- Minor fixes in protobuf files #13512
- Fix 500 error when requesting blobs from a block without blobs #13585
- Handle cases were EL client is syncing and unable to provide payloads #13597
- `/eth/v1/beacon/blob_sidecars/{block_id}` correctly returns an error when invalid indices are requested #13616

**Fork choice**

- Fix head state fetch when proposing a failed reorg #13579
- Fix data race in background forkchoice update call #13602

**p2p**

- Correctly return "unavailable" response to peers requesting batches before the node completes
  backfill. #13587

**Slasher**

- Many significant improvements and fixes to the prysm slasher #13549 #13589 #13596 #13612 #13620
- Fixed slashing gossip checks, improves peer scores for slasher peers #13574

**Uncategorized**

- Log warning if attempting to exit more than 5 validators at a time #13542 
- Do not cache inactive public keys #13581
- Validator exits prints testnet URLs #13610 #13308
- Fix pending block/blob zero peer edge case #13625
- Check non-zero blob data is written to disk #13647
- Avoid blob partial filepath collisions with mem addr entropy #13648

### Security

v5.0.0 of Prysm is required to maintain participation in the network after the Deneb upgrade.

