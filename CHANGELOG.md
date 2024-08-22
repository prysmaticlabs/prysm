# Changelog 

All notable changes to this project will be documented in this file.

The format is based on Keep a Changelog, and this project adheres to Semantic Versioning.

## [Unreleased](https://github.com/prysmaticlabs/prysm/compare/v5.1.0...HEAD)

### Added


### Changed


### Deprecated


### Removed


### Fixed


### Security


## [v5.1.0](https://github.com/prysmaticlabs/prysm/compare/v5.0.4...v5.1.0) - 2024-08-20

This release contains 171 new changes and many of these are related to Electra! Along side the Electra changes, there are nearly 100 changes related to bug fixes, feature additions, and other improvements to Prysm. Updating to this release is recommended at your convenience.

⚠️ Deprecation Notice: Removal of gRPC Gateway and Gateway Flag Renaming ⚠️

In an upcoming release, we will be deprecating the gRPC gateway and renaming several associated flags. This change will result in the removal of access to several internal APIs via REST, though the gRPC endpoints will remain unaffected. We strongly encourage systems to transition to using the beacon API endpoints moving forward. Please refer to PR #14089 for more details.

### Added

- Electra work #13907 #13918 #13905 #13923 #13924 #13937 #13921 #13946 #13933 #13974 #13976 #13937 #13919 #13975 #13984 #13985 #13987 #13991 #13978 #13981 #13982 #13992 #13994 #13993 #13997 #13999 #13983 #14002 #14000 #14029 #14001 #14047 #14003 #14037 #14031 #14055 #14027 #14091 #14085 #14138 #14121 #14139 #14146 #14152 #14010 #13944 #14005 #14164 #14115 #14163 #14177 #14176 #14158 #14200 #14212 #14203 #14211 #14180 #14181 #14213 #14215 #14219 #14221 #14220 #14224 #14209 #14227 #14229 #14235 #14272
- Fork-specific consensus-types interfaces #13937 #14241 #14243 #14238 #14239 #14173 #14257
- Fuzz ssz roundtrip marshalling, cloner fuzzing #14006 #14246 #14254 #14255 #14265 #14294
- Add support for multiple beacon nodes in the REST API #13433
- Add middleware for Content-Type and Accept headers #14075 #14093
- Add debug logs for proposer settings #14016
- Add tracing to beacon api package #14125
- Add support for persistent validator keys when using remote signer. --validators-external-signer-public-keys and --validators-external-signer-key-file See the docs page for more info. #13682
- Add AggregateKeyFromIndices to beacon state to reduce memory usage when processing attestations #14178
- Add GetIndividualVotes endpoint #14198
- Implement is_better_update for light client #14186
- HTTP endpoint for GetValidatorParticipation #14261
- HTTP endpoint for GetChainHead #14262
- HTTP endpoint for GetValidatorActiveSetChanges #14264
- Check locally for min-bid and min-bid-difference

### Changed

- Refactored slasher operations to their logical order #14322
- Refactored Gwei and Wei types from math to primitives package. #14026
- Unwrap payload bid from ExecutionData #14035
- Change ZeroWei to a func to avoid shared ptr #14043
- Updated go-libp2p to v0.35.2 and go-libp2p-pubsub to v0.11.0 #14060 #14192
- Use genesis block root in epoch 1 for attester duties #14059
- Cleanup validator client code #14048
- Old attestations log moved to debug. "Attestation is too old to broadcast, discarding it" #14072
- Modify ProcessEpoch not to return the state as a returned value #14069
- Updated go-bitfield to latest release #14120
- Use go ticker instead of timer #14134
- process_registry_updates no longer makes a full copy of the validator set #14130 #14197
- Validator client processes sync committee roll separately #13995
- Use vote pointers in forkchoice to reduce memory churn #14196
- Avoid Cloning When Creating a New Gossip Message #14201
- Proposer filters invalid attestation signatures #14225
- Validator now pushes proposer settings every slot #14155 #14285
- Get all beacon committees at once #14282 #14284
- Committee-aware attestation packing #14245

### Deprecated

- `--enable-debug-rpc-endpoints` is deprecated and debug rpc points are on by default. #14015

### Removed

- Removed fork specific getter functions (i.e. PbCapellaBlock, PbDenebBlock, etc) #13941

### Fixed

- Fixed debug log "upgraded stake to $fork" to only log on upgrades instead of every state transition #14316
- Fixed nil block panic in API #14063
- Fixed mockgen script #14068
- Do not fail to build block when block value is unknown #14111
- Fix prysmctl TUI when more than 20 validators were listed #14140
- Revert peer backoff changes from #14137. This was causing some sync committee performance issues. #14148
- Increased attestation seen cache expiration to two epochs #14156
- Fixed slasher db disk usage leak #14151
- fix: Multiple network flags should prevent the BN to start #14169
- Correctly handle empty payload from GetValidatorPerformance requests #14240
- Fix Event stream with carriage return support #14250
- Fix panic on empty block result in REST API #14280
- engine_getPayloadBodiesByRangeV1 - fix, adding hexutil encoding on request parameters #14314


### Security

- Go version updated to 1.22 #13965

## [v5.0.4](https://github.com/prysmaticlabs/prysm/compare/v5.0.3...v5.0.4) - 2024-07-21

This release has many wonderful bug fixes and improvements. Some highlights include p2p peer fix for windows users, beacon API fix for retrieving blobs older than the minimum blob retention period, and improvements to initial sync by avoiding redundant blob downloads.

Updating to this release is recommended at your earliest convenience, especially for windows users.

### Added

- Beacon-api: broadcast blobs in the event of seen block #13830
- P2P: Add QUIC support #13786 #13872

### Changed

- Use slices package for various slice operations #13834 #13837 #13838 #13835 #13839 #13836
- Initsync skip local blobs #13827 #13871
- Use read only validators in Beacon API #13873
- Return syncing status when node is optimistic #13875
- Upgrade the Beacon API e2e evaluator #13868
- Don't return error that can be internally handled #13887
- Allow consistent auth token for validator apis #13747
- Change example.org DNS record #13904
- Simplify prune invalid by reusing existing fork choice store call #13878
- use [32]byte keys in the filesystem cache #13885
- Update Libp2p Dependencies #13960
- Parallelize Broadcasting And Processing Each Blob #13959
- Substantial VC cleanup #13593 #14040
- Only log error when aggregator check fails #14046
- Update Libp2p Dependencies #14060
- Change Attestation Log To Debug #14072
- update codegen dep and cleanup organization #14127

### Deprecated

- Remove eip4881 flag (--disable-eip-4881) #13826

### Removed

- Remove the Goerli/Prater support #13846
- Remove unused IsViableForCheckpoint #13879
- Remove unused validator map copy method #13954

### Fixed

- Various typos and other cosmetic fixes #13833 #13843
- Send correct state root with finalized event stream #13842
- Extend Broadcast Window For Attestations #13858
- Beacon API: Use retention period when fetching blobs #13869 #13874
- Backfill throttling #13855
- Use correct port for health check in Beacon API e2e evaluator #13892
- Do not remove blobs DB in slasher. #13881
- use time.NewTimer() to avoid possible memory leaks #13800
- paranoid underflow protection without error handling #14044
- Fix CommitteeAssignments to not return every validator #14039
- Fix dependent root retrival genesis case #14053
- Restrict Dials From Discovery #14052
- Always close cache warm chan to prevent blocking #14080
- Keep only the latest value in the health channel #14087

### Security

- Bump golang.org/x/net from 0.21.0 to 0.23.0 #13895

## [v5.0.3](https://github.com/prysmaticlabs/prysm/compare/v5.0.2...v5.0.3) - 2024-04-04

Prysm v5.0.3 is a small patch release with some nice additions and bug fixes. Updating to this release is recommended for users on v5.0.0 or v5.0.1. There aren't many changes since last week's v5.0.2 so upgrading is not strictly required, but there are still improvements in this release so update if you can!

### Added

- Testing: spec test coverage tool #13718
- Add bid value metrics #13804
- prysmctl: Command-line interface for visualizing min/max span bucket #13748
- Explicit Peering Agreement implementation #13773

### Changed

- Utilize next slot cache in block rewards rpc #13684
- validator: Call GetGenesis only once when using beacon API #13796
- Simplify ValidateAttestationTime #13813
- Various typo / commentary improvements #13792
- Change goodbye message from rate limited peer to debug verbosity #13819
- Bump libp2p to v0.33.1 #13784
- Fill in missing debug logs for blob p2p IGNORE/REJECT #13825

### Fixed

- Remove check for duplicates in pending attestation queue #13814
- Repair finalized index issue #13831
- Maximize Peer Capacity When Syncing #13820
- Reject Empty Bundles #13798

### Security

No security updates in this release.

## [v5.0.2](https://github.com/prysmaticlabs/prysm/compare/v5.0.1...v5.0.2) - 2024-03-27

This release has many optimizations, UX improvements, and bug fixes. Due to the number of important bug fixes and optimizations, we encourage all operators to update to v5.0.2 at their earliest convenience.

In this release, there is a notable change to the default value of --local-block-value-boost from 0 to 10. This means that the default behavior of using the builder API / mev-boost requires the builder bid to be 10% better than your local block profit. If you want to preserve the existing behavior, set --local-block-value-boost=0.


### Added

- API: Add support for sync committee selections #13633
- blobs: call fsync between part file write and rename (feature flag --blob-save-fsync) #13652
- Implement EIP-3076 minimal slashing protection, using a filesystem database (feature flag --enable-minimal-slashing-protection) #13360
- Save invalid block to temp --save-invalid-block-temp #13722 #13725 #13736
- Compute unrealized checkpoints with pcli #13692
- Add gossip blob sidecar verification ms metric #13737
- Backfill min slot flag (feature flag --backfill-oldest-slot) #13729
- adds a metric to track blob sig cache lookups #13755
- Keymanager APIs - get,post,delete graffiti #13474
- Set default LocalBlockValueBoost to 10 #13772
- Add bid value metrics #13804
- REST VC metrics #13588

### Changed

- Normalized checkpoint logs #13643
- Normalize filesystem/blob logs #13644
- Updated gomock libraries #13639
- Use Max Request Limit in Initial Sync #13641
- Do not Persist Startup State #13637
- Normalize backfill logs/errors #13642
- Unify log fields #13654
- Do Not Compute Block Root Again #13657
- Optimize Adding Dirty Indices #13660
- Use a Validator Reader When Computing Unrealized Balances #13656
- Copy Validator Field Trie #13661
- Do not log zero sync committee messages #13662
- small cleanup on functions: use slots.PrevSlot #13666
- Set the log level for running on <network> as INFO. #13670
- Employ Dynamic Cache Sizes #13640
- VC: Improve logging in case of fatal error #13681
- refactoring how proposer settings load into validator client #13645
- Spectest: Unskip Merkle Proof test #13704
- Improve logging. #13708
- Check Unrealized Justification Balances In Spectests #13710
- Optimize SubscribeCommitteeSubnets VC action #13702
- Clean up unreachable code; use new(big.Int) instead of big.NewInt(0) #13715
- Update bazel, rules_go, gazelle, and go versions #13724
- replace receive slot with event stream #13563
- New gossip cache size #13756
- Use headstate for recent checkpoints #13746
- Update spec test to official 1.4.0 #13761
- Additional tests for KZG commitments #13758
- Enable Configurable Mplex Timeouts #13745
- Optimize SubmitAggregateSelectionProof VC action #13711
- Re-design TestStartDiscV5_DiscoverPeersWithSubnets test #13766
- Add da waited time to sync block log #13775
- add log message if in da check at slot end #13776
- Log da block root in hex #13787
- Log the slot and blockroot when we deadline waiting for blobs #13774
- Modify the algorithm of updateFinalizedBlockRoots #13486
- Rename payloadattribute Timestamps to Timestamp #13523
- Optimize GetDuties VC action #13789
- docker: Add bazel target for building docker tarball #13790
- Utilize next slot cache in block rewards rpc #13684
- Spec test coverage report #13718
- Refactor batch verifier for sharing across packages #13812

### Removed

- Remove unused bolt buckets #13638
- config: Remove DOMAIN_BLOB_SIDECAR. #13706
- Remove unused deneb code #13712
- Clean up: remove some unused beacon state protos #13735
- Cleaned up code in the sync package #13636
- P2P: Simplify code #13719

### Fixed

- Slasher: Reduce surrounding/surrounded attestations processing time #13629
- Fix blob batch verifier pointer receiver #13649
- db/blobs: Check non-zero data is written to disk #13647
- avoid part path collisions with mem addr entropy #13648
- Download checkpoint sync origin blobs in init-sync #13665 #13667
- bazel: Update aspect-build/bazel-lib to v2.5.0 #13675
- move setting route handlers to registration from start #13676
- Downgrade Level DB to Stable Version #13671
- Fix failed reorg log #13679
- Fix Data Race in Epoch Boundary #13680
- exit blob fetching for cp block if outside retention #13686
- Do not check parent weight on early FCU #13683
- Fix VC DB conversion when no proposer settings is defined and add Experimental flag in the --enable-minimal-slashing-protection help. #13691
- keymanager api: lowercase statuses #13696
- Fix unrealized justification #13688
- fix race condition when pinging peers #13701
- Fix/race receive block #13700
- Blob verification spectest #13707
- Ignore Pubsub Messages Hitting Context Deadlines #13716
- Use justified checkpoint from head state to build attestation #13703
- only update head at 10 seconds when validating #13570
- Use correct gossip validation time #13740
- fix 1-worker underflow; lower default batch size #13734
- handle special case of batch size=1 #13646
- Always Set Inprogress Boolean In Cache #13750
- Builder APIs: adding headers to post endpoint #13753
- Rename mispelled variable #13759
- allow blob by root within da period #13757
- Rewrite Pruning Implementation To Handle EIP 7045 #13762
- Set default fee recipient if tracked val fails #13768
- validator client on rest mode has an inappropriate context deadline for events #13771
- validator client should set beacon API endpoint in configurations #13778
- Fix get validator endpoint for empty query parameters #13780
- Expand Our TTL for our Message ID Cache #13770
- fix some typos #13726
- fix handling of goodbye messages for limited peers #13785
- create the log file along with its parent directory if not present #12675
- Call GetGenesis only once #13796

### Security

- Go version has been updated from 1.21.6 to 1.21.8. #13724

## [v5.0.1](https://github.com/prysmaticlabs/prysm/compare/v5.0.0...v5.0.1) - 2024-03-08

This minor patch release has some nice improvements over the recent v5.0.0 for Deneb. We have minimized this patch release to include only low risk and valuable fixes or features ahead of the upcoming network upgrade on March 13th.

Deneb is scheduled for mainnet epoch 269568 on March 13, 2024 at 01:55:35pm UTC. All operators MUST update their Prysm software to v5.0.0 or later before the upgrade in order to continue following the blockchain.

### Added

- A new flag to ensure that blobs are flushed to disk via fsync immediately after write. --blob-save-fsync #13652

### Changed

- Enforce a lower maximum batch limit value to prevent annoying peers #13641
- Download blobs for checkpoint sync block before starting sync #13665 #13667 #13686
- Set justified epoch to the finalized epoch in Goerli to unstuck some Prysm nodes on Goerli #13695

### Fixed

- Data race in epoch boundary cache #13680
- "Failed reorg" log was misplaced #13679
- Do not check parent weights on early fork choice update calls #13683
- Compute unrealized justification with slashed validators #13688 #13710
- Missing libxml dependency #13675

### Security

Prysm version v5.0.0 or later is required to maintain participation in the network after the Deneb upgrade.

## [v5.0.0](https://github.com/prysmaticlabs/prysm/compare/v4.2.1...v5.0.0)

Behold the Prysm v5 release with official support for Deneb on Ethereum mainnet!

Deneb is scheduled for mainnet epoch 269568 on March 13, 2024 at 01:55:35pm UTC. All operators MUST update their Prysm software to v5.0.0 or later before the upgrade in order to continue following the blockchain.

This release brings improvements to the backfill functionality of the beacon node to support backfilling blobs. If running a beacon node with checkpoint sync, we encourage you to test the backfilling functionality and share your feedback. Run with backfill enabled using the flag --enable-experimental-backfill.

Known Issues

- --backfill-batch-size with a value of 1 or less breaks backfill. #13646
- Validator client on v4.2.0 or older uses some API methods that are incompatible with beacon node v5. Ensure that you have updated the beacon node and validator client to v4.2.1 and then upgrade to v5 or update both processes at the same time to minimize downtime.

### Added

- Support beacon_committee_selections #13503
- /eth/v1/beacon/deposit_snapshot #13514
- Docker images now have coreutils pre-installed #13564
- da_waited_time_milliseconds tracks total time waiting for data availablity check in ReceiveBlock #13534
- blob_written, blob_disk_count, blob_disk_bytes new metrics for tracking blobs on disk #13614
- Backfill supports blob backfilling #13595
- Add mainnet deneb fork epoch config #13601

### Changed

- --clear-db and --force-clear-db flags now remove blobs as well as beaconchain.db #13605
- EIP-4881 is now on by default. #13555
- Updates filtering logic to match spec #13464
- Verbose signature verification is now on by default #13556
- gossip_block_arrival_milliseconds and gossip_block_verification_milliseconds measure in
- milliseconds instead of nanoseconds #13540
- aggregate_attestations_t1 histogram buckets have been updated #13607
- Reduce lookahead period from 8 to 4. This reduces block batch sizes during sync to account for
- larger blocks in deneb. #13599
- Update gohashtree to v0.0.4-beta #13569
- Various logging improvements #13571 #13582 #13561 #13573 #13598 #13608 #13611 #13502 #13627
- Improved operations during syncing #13580
- Backfill starts after initial-sync is complete #13623

### Deprecated

The following flags have been removed entirely:

- --enable-reorg-late-blocks #13536
- --disable-vectorized-htr #13537
- --aggregate-parallel #13538
- --build-block-parallel #13539
- --enable-registration-cache, disable-gossip-batch-aggregation #13606
- --safe-slots-to-import-optimistically #13624
- --show-deposit-data #13618


### Removed

- Prysm gRPC slasher endpoints are removed #13594
- Remove /eth/v1/debug/beacon/states/{state_id} #13619
- Prysm gRPC endpoints that were marked as deprecated in v4 have been removed #13600
- Remove /eth/v1/beacon/blocks/{block_id} #13628

### Fixed

- Return unaggregated if no aggregated attestations available in GetAggregateAttestation #13533
- Fix JWT auth checks in certain API endpoints used by the web UI #13565 #13568
- Return consensus block value in wei units #13575
- Minor fixes in protobuf files #13512
- Fix 500 error when requesting blobs from a block without blobs #13585
- Handle cases were EL client is syncing and unable to provide payloads #13597
- /eth/v1/beacon/blob_sidecars/{block_id} correctly returns an error when invalid indices are requested #13616
- Fix head state fetch when proposing a failed reorg #13579
- Fix data race in background forkchoice update call #13602
- Correctly return "unavailable" response to peers requesting batches before the node completes
- backfill. #13587
- Many significant improvements and fixes to the prysm slasher #13549 #13589 #13596 #13612 #13620
- Fixed slashing gossip checks, improves peer scores for slasher peers #13574
- Log warning if attempting to exit more than 5 validators at a time #13542
- Do not cache inactive public keys #13581
- Validator exits prints testnet URLs #13610 #13308
- Fix pending block/blob zero peer edge case #13625
- Check non-zero blob data is written to disk #13647
- Avoid blob partial filepath collisions with mem addr entropy #13648


### Security

v5.0.0 of Prysm is required to maintain participation in the network after the Deneb upgrade.

## [v4.2.1](https://github.com/prysmaticlabs/prysm/compare/v4.2.0...v4.2.1) - 2024-01-29

Welcome to Prysm Release v4.2.1! This release is highly recommended for stakers and node operators, possibly being the final update before V5.

⚠️ This release will cause failures on Goerli, Sepolia and Holeski testnets, when running on certain older CPUs without AVX support (eg Celeron) after the Deneb fork. This is not an issue for mainnet.

### Added

- Linter: Wastedassign linter enabled to improve code quality (#13507).
- API Enhancements:
  - Added payload return in Wei for /eth/v3/validator/blocks (#13497).
  - Added Holesky Deneb Epoch for better epoch management (#13506).
- Testing Enhancements:
  - Clear cache in tests of core helpers to ensure test reliability (#13509).
  - Added Debug State Transition Method for improved debugging (#13495).
  - Backfilling test: Enabled backfill in E2E tests for more comprehensive coverage (#13524).
- API Updates: Re-enabled jwt on keymanager API for enhanced security (#13492).
- Logging Improvements: Enhanced block by root log for better traceability (#13472).
- Validator Client Improvements:
  - Added Spans to Core Validator Methods for enhanced monitoring (#13467).
  - Improved readability in validator client code for better maintenance (various commits).

### Changed

- Optimizations and Refinements:
  - Lowered resource usage in certain processes for efficiency (#13516).
  - Moved blob rpc validation closer to peer read for optimized processing (#13511).
  - Cleaned up validate beacon block code for clarity and efficiency (#13517).
  - Updated Sepolia Deneb fork epoch for alignment with network changes (#13491).
  - Changed blob latency metrics to milliseconds for more precise measurement (#13481).
  - Altered getLegacyDatabaseLocation message for better clarity (#13471).
  - Improved wait for activation method for enhanced performance (#13448).
  - Capitalized Aggregated Unaggregated Attestations Log for consistency (#13473).
  - Modified HistoricalRoots usage for accuracy (#13477).
  - Adjusted checking of attribute emptiness for efficiency (#13465).
- Database Management:
  - Moved --db-backup-output-dir as a deprecated flag for database management simplification (#13450).
  - Added the Ability to Defragment the Beacon State for improved database performance (#13444).
- Dependency Update: Bumped quic-go version from 0.39.3 to 0.39.4 for up-to-date dependencies (#13445).

### Removed

- Removed debug setting highest slot log to clean up the logging process (#13488).
- Deleted invalid blob at block processing for data integrity (#13456).

### Fixed

- Bug Fixes:
  - Fixed off by one error for improved accuracy (#13529).
  - Resolved small typo in error messages for clarity (#13525).
  - Addressed minor issue in blsToExecChange validator for better validation (#13498).
  - Corrected blobsidecar json tag for commitment inclusion proof (#13475).
  - Fixed ssz post-requests content type check (#13482).
  - Resolved issue with port logging in bootnode (#13457).
- Test Fixes: Re-enabled Slasher E2E Test for more comprehensive testing (#13420).

### Security

No security issues in this release.

## [v4.2.0](https://github.com/prysmaticlabs/prysm/compare/v4.1.1...v4.2.0) - 2024-01-11

Happy new year! We have an incredibly exciting release to kick off the new year. This release is **strongly recommended** for all operators to update as it has many bug fixes, security patches, and features that will improve the Prysm experience on mainnet. This release has so many wonderful changes that we've deviated from our normal release notes format to aptly categorize the changes.

### Highlights

#### Upgrading / Downgrading Validators

There are some API changes bundled in this release that require you to upgrade or downgrade in particular order. If the validator is updated before the beacon node, it will see repeated 404 errors at start up until the beacon node is updated as it uses a new API endpoint introduced in v4.2.0.

:arrow_up_small:  **Upgrading**: Upgrade the beacon node, then the validator.
:arrow_down_small: **Downgrading**: Downgrade the validator to v4.1.1 then downgrade the beacon node.

#### Deneb Goerli Support
This release adds in full support for the upcoming deneb hard fork on goerli next week on January 17th.

#### Networking Parameter Changes
This release increases the default peer count to 70 from 45. The reason this is done is so that node's running
with default peer counts can perform their validator duties as expected. Users who want to use the old peer count
can add in `--p2p-max-peers=45` as a flag.

#### Profile Guided Optimization
This release has binaries built using PGO, for more information on how it works feel free to look here: https://tip.golang.org/doc/pgo . 
This allows the go compiler to build more optimized Prysm binaries using production profiles and workloads.  

#### ARM Supported Docker Images

Our docker images now support amd64 and arm64 architecture! This long awaited feature is finally here for Apple Silicon and Raspberry Pi users. 

### Deneb

#### Core

- Use ROForkchoice in blob verifier (#13426)
- Add Goerli Deneb Fork Epoch (#13390)
- Use deneb key for deneb state in saveStatesEfficientInternal (#13374)
- Initialize Inactivity Scores Correctly (#13375)
- Excluse DA wait time for chain processing time (#13335)
- Initialize sig cache for verification.Initializer (#13295)
- Verify roblobs (#13245)
- KZG Commitment inclusion proof verifier (#13174)
- Merkle Proofs of KZG commitments (#13159)
- Add RO blob sidecar (#13144)
- Check blob index duplication for blob notifier (#13123)
- Remove sidecars with invalid proofs (#13070)
- Proposer: better handling of blobs bundle (#12956)
- Update proposer RPC to new blob sidecar format (#13189)
- Implement Slot-Dependent Caching for Blobs Bundle (#13205)
- Verified roblobs (#13190)

#### Networking

- Check sidecar index in BlobSidecarsByRoot response (#13180)
- Use proposer index cache for blob verification (#13423)
- VerifiedROBlobs in initial-sync (#13351)
- Reordered blob validation (#13347)
- Initialize blob storage for initial sync service (#13312)
- Use verified blob for gossip checks (#13294)
- Update broadcast method to use `BlobSidecar` instead of `SingedBlobSidecar` (#13221)
- Remove pending blobs queue (#13166)
- Reject Blob Sidecar Incorrect Index (#13094)
- Check return and request lengths for blob sidecar by root (#13106)
- Fix blob sidecar subnet check (#13102)
- Add pending blobs queue for missing parent block (#13005)
- Verify blobs that arrived from by root request (#13044)
- Reject blobs with invalid parent (#13047)
- Add more blob and block checks for by range (#13043)
- Exit early if blob by root request is empty (#13038)
- Request missing blobs while processing pending queue (#13015)
- Check blob exists before requesting from peer (#13012)
- Passing block as arugment for sidecar validation (#13062)

#### Blob Management

- Remove old blob types (#13438)
- minimize syscalls in pruning routine (#13425)
- Prune dangling blob (#13424)
- Use Afero Walk for Pruning Blob (#13410)
- Initialize blob storage without pruning (#13412)
- Fix batch pruning errors  (#13355)
- Blob filesystem add pruning during blob write (#13275)
- Blob filesystem add pruning at startup (#13253)
- Ensure partial blob is deleted if there's an error (#13292)
- Split blob pruning into two funcs (#13285)
- Use functional options for `--blob-retention-epochs` (#13283)
- Blob filesystem: delete blobs  (#13233)
- Fix Blob Storage Path (#13222)
- Add blob getters (#13170)
- Blob filesystem: Save Blobs (#13129)
- Blob filesystem: prune blobs (#13147)
- blobstorage: Improve mkdirall error  (#13271)

#### Beacon-API

- Add rpc trigger for blob sidecar event (#13411)
- Do not skip mev boost in `v3` block production endpoint (#13365)
- Beacon APIs: re enabling blob events (#13315)
- Beacon API: update Deneb endpoints after removing blob signing (#13235)
- Beacon API: fix get blob returns 500 instead of empty (#13297)
- Fix bug in Beacon API getBlobs (#13100)
- Fix blob_sidecar SSE payload (#13050)
- fix(beacon-chain/rpc): blob_sidecar event stream handler (#12999)
- Improvements to `produceBlockV3` (#13027)
- Deneb: Produce Block V3 - adding consensus block value (#12948)

#### Validator Client

- Validator client: remove blob signing (#13169)
- Deneb - web3signer (#12767) 

#### Testing

- Enable Deneb For E2E Scenario Tests (#13317)
- Activate deneb in E2E (#13311)
- Deneb E2E (#13040)

#### Miscellaneous

- Update blob pruning log (#13417)
- Fix total pruned metric + add to logging (#13367)
- Check kzg commitment count from builder (#13394)
- Add error wrapping to blob initialization errors (#13366)
- Blob filesystem metrics (#13316)
- Check builder header kzg commitment (#13358)
- Add more color to sending blob by range req log (#13349)
- Move pruning log to after retention check (#13348)
- Enhance Pruning Logs  (#13331)
- Rename Blob retention epoch flag (#13124)
- Check that blobs count is correct when unblinding (#13118)
- Log blob's kzg commmitment at sync (#13111)
- Replace MAX_BLOB_EPOCHS usages with more accurate terms (#13098)
- Fix comment of `BlobSidecarsBySlot` (#13019)

### Core Prysm Work(Non-Deneb)

#### Core Protocol

- Only process blocks which haven't been processed (#13442)
- Initialize exec payload fields and enforce order (#13372)
- Add nil check for head in IsOptimistic (#13439)
- Unlock forkchoice store if attribute is empty (#13427)
- Make Aggregating In Parallel The Permanent Default (#13407)
- Break out several helpers from `postBlockProcess` (#13419)
- Don't hardcode 4 seconds in forkchoice (#13416)
- Simplify fcu 4 (#13403)
- Remove the getPayloadAttribute call from updateForkchoiceWithExecution (#13402)
- Simplify fcu 2 (#13400)
- Remove getPayloadAttributes from FCU call (#13399)
- Simplify fcu 1 (#13387)
- Remove unsafe proposer indices cache (#13385)
- Rewrite `ProposeBlock` endpoint (#13380)
- Remove blind field from block type (#13389)
- update shuffling caches before calling FCU on epoch boundaries (#13383)
- Return SignedBeaconBlock from ReadOnlySignedBeaconBlock.Copy (#13386)
- Use advanced epoch cache when preparing proposals (#13377)
- refactor Payload Id caches (#12987)
- Use block value correctly when proposing a block (#13368)
- use different keys for the proposer indices cache (#13272)
- Use a cache of one entry to build attestation (#13300)
- Remove signed block requirement from no-verify functions (#13314)
- Allow requests for old target roots (#13281)
- Remove Redundant Hash Computation in Cache (#13261)
- Fix FFG LMD Consistency Check (Option 2) (#13258)
- Verify lmd without ancestor (#13250)
- Track target in forkchoice (#13249)
- Return early from ReceiveBlock if already sycned (#13089)
 
#### Builder

- Adding builder boost factor to get block v3 (#13409)
- Builder API: Fix max field check on toProto function (#13334)
- Add sanity checks for bundle from builder (#13319)
- Update Prysm Proposer end points for Builder API (#13240)
- Builder API: remove blinded blob sidecar (#13202)
- Allow validators registration batching on Builder API `/eth/v1/builder/validators` (#13178)

#### State-Management

- Add Detailed Multi Value Metrics (#13429)
- Optimize Multivalue Slice For Trie Recomputation (#13238)
- Fix Multivalue Slice Deadlock (#13087)
- Set Better Slice Capacities in the State (#13068)

#### Networking

- Refactor Network Config Into Main Config (#13364)
- Handle potential error from newBlockRangeBatcher (#13344)
- Clean Up Goodbye Stream Errors (#13325)
- Support New Subnet Backbone (#13179)
- Increase Networking Defaults (#13278)
- Bump Up Gossip Queue Size (#13277)
- Improve Gossipsub Rejection Metric (#13236)
- Add Gossipsub Queue Flag (#13237)
- Fix Deadlock With Subscriber Checker (#13234)
- Add Additional Pubsub Metrics (#13226)
- Verify Block Signatures On Insertion Into Pending Queue (#13183)
- Enhance Validation for Block by Root RPC Requests (#13184)
- Add a helper for max request block (#13173)
- Fix Pending Queue Deadline Bug (#13145)
- Add context deadline for pending queue's receive block (#13114)
- Fix Pending Queue Expiration Bug (#13104)
- sync only up to previous epoch on phase 1 (#13083)
- Use correct context for sendBatchRootRequest (#13061)
- Refactor Pending Block Queue Logic in Sync Package (#13026)
- Check block exists in pending queue before requesting from peer (#13013)
- Set Verbosity of Goodbye Logs to Trace (#13077)
- use read only head state (#13014)

#### Beacon-API

_Most of the PRs here involve shifting our http endpoints to using vanilla http handlers(without the API middleware)._

- http endpoint cleanup (#13432)
- Revert "REST VC: Subscribe to Beacon API events  (#13354)" (#13428)
- proposer and attester slashing sse (#13414)
- REST VC: Subscribe to Beacon API events  (#13354)
- Simplify error handling for JsonRestHandler (#13369)
- Update block publishing to 2.4.2 spec (#13376)
- Use `SkipMevBoost` properly during block production (#13352)
- Handle HTTP 404 Not Found in `SubmitAggregateAndProof` (#13320)
- beacon-chain/rpc: use BalanceAtIndex instead of Balances to reduce memory copy (#13279)
- HTTP endpoints cleanup (#13251)
- APIs: reusing grpc cors middleware for rest (#13284)
- Beacon API: routes unit test (#13276)
- Remove API Middleware (#13243)
- HTTP validator API: beacon and account endpoints (#13191)
- REST VC: Use POST to fetch validators (#13239)
- HTTP handler for Beacon API events (#13207)
- Move weak subjectivity endpoint to HTTP (#13220)
- Handle non-JSON responses from Beacon API (#13213)
- POST version of GetValidators and GetValidatorBalances (#13199)
- [2/5] light client http api (#12984)
- HTTP validator API: wallet endpoints (#13171)
- HTTP Validator API: slashing protection import and export (#13165)
- Config HTTP endpoints (#13168)
- Return 404 from `eth/v1/beacon/headers` when there are no blocks (#13185)
- Pool slashings HTTP endpoints (#13148)
- Validator HTTP endpoints (#13167)
- Debug HTTP endpoints (#13164)
- HTTP validator API: health endpoints (#13149)
- HTTP Validator API:  `/eth/v1/keystores` (#13113)
- Allow unknown fields in Beacon API responses (#13131)
- HTTP state endpoints (#13099)
- HTTP Validator API: `/eth/v1/validator/{pubkey}/feerecipient` (#13085)
- HTTP Validator API: `/eth/v1/validator/{pubkey}/gas_limit` (#13082)
- HTTP VALIDATOR API: remote keymanager api `/eth/v1/remotekeys` (#13059)
- rpc/apimiddleware: Test all paths can be created (#13073)
- HTTP Beacon APIs for blocks (#13048)
- HTTP VALIDATOR API: `/eth/v1/validator/{pubkey}/voluntary_exit` (#13032)
- HTTP Beacon APIs: 3 state endpoints (#13001)
- HTTP Beacon APIs for node (#13010)
- HTTP API: `/eth/v1/beacon/pool/bls_to_execution_changes` (#12963)
- Register sync subnet when fetching sync committee duties through Beacon API (#12972)

#### Validator Client

- Refactor validator client help. (#13401)
- `--validatorS-registration-batch-size` (add `s`) (#13396)
- Validator client: Always use the `--datadir` value. (#13392)
- Hook to slot stream instead of block stream on the VC (#13327)
- CLI: fixing account import ux bugs (#13328)
- `filterAndCacheActiveKeys`: Stop filtering out exiting validators (#13305)
- Gracefully handle unknown validator index in the REST VC (#13296)
- Don't fetch duties for unknown keys (#13269)
- Fix Domain Data Caching (#13263)
- Add `--jwt-id` flag (#13218)
- Make Prysm VC compatible with the version `v5.3.0` of the slashing protections interchange tests. (#13232)
- Fix handling POST requests in the REST VC (#13215)
- Better error handling in REST VC (#13203)
- Fix block proposals in the REST validator client (#13116)
- CLEANUP: validator exit prompt (#13057)
- integrate validator count endpoint in validator client (#12912)

#### Build/CI Work

- Bazel 7.0.0 (#13321)
- Sort static analyzers, add more, fix violations (#13441)
- For golangci-lint, enable all by default (#13353)
- Enable mirror linter and fix findings (#13342)
- Enable usestdlibvars linter and fix findings (#13339)
- Fix docker image version strings in CI (#13356)
- fixing sa4006 (#13350)
- Enable errname linter and fix findings (#13341)
- Remove rules_docker, make multiarch images canonical (#13324)
- Fix staticcheck violations (#13301)
- Add staticchecks to bazel builds (#13298)
- CI: Add merge queue events trigger for github workflows (#13282)
- Update bazel and other CI improvements (#13246)
- bazel: Run buildifier, general cleanup (#13193)
- pgo: Enable pgo behind release flag (#13158)
- pgo: remove default pprof profile (#13150)
- zig: Update zig to recent main branch commit (#13142)
- Enable profile guided optimization for beacon-chain (#13035)
- Refactor Exported Names to Follow Golang Best Practices (#13075)
- Update rules_go and gazelle to 0.42 & 0.33 (latest releases) (#13021)
- Fix image deps (#13022)

#### Dependency Updates

- Update go to 1.21.6 (#13440)
- Update Our Golang Crypto Library (#13415)
- Update libp2p/go-libp2p-asn-util to v0.4.1 (#13370)
- Update Libp2p To v0.32.1 and Go to v1.21.5 (#13304)
- Bump google.golang.org/grpc from 1.53.0 to 1.56.3 (#13119)
- Update go to 1.20.10 (#13120)

#### Testing

- Enable Profiling for Long Running E2E Runs (#13421)
- Fetch Goroutine Traces in E2E (#13404)
- Fix Up Builder Evaluator (#13395)
- Increase Blob Batch Parameters in E2E (#13398)
- Uncomment e2e flakiness (#13326)
- Update spectests to 1.4.0-beta.5 (#13318)
- Test improvement TestValidateVoluntaryExit_ValidExit (#13313)
- Simplify post-evaluation in Beacon API evaluator (#13309)
- Run Evaluator In the Middle Of An Epoch (#13303)
- Simplify Beacon API evaluator (#13265)
- Fix Optimistic Sync Evaluator (#13262)
- Add test helpers to produce commitments and proofs (#13242)
- Redesign of Beacon API evaluator (#13229)
- Drop Transaction Count for Transaction Generator (#13228)
- Add concurrency test for getting attestation state (#13196)
- Add `construct_generic_block_test` to build file (#13195)
- Implement Merkle proof spectests (#13146)
- Remove `/node/peers/{peer_id}` from Beacon API evaluator (#13138)
- Update spectest and changed minimal preset for field elements (#13090)
- Better Beacon API evaluator part 1 (#13084)
- beacon-chain/blockchain: fix some datarace in go test (#13036)
- beacon-node/rpc: fix go test datarace (#13018)
- Fix Builder Testing For Multiclient Runs (#13091)
- Fill state attestations (#13121)
- beacon-chain/sync: fix some datarace in go test (#13039)
- beacon-chain/execution: fix a data race in testcase (#13016)
- Add state not found test case (#13034)

#### Feature Updates

- Make New Engine Methods The Permanent Default (#13406)
- Make Reorging Of Late Blocks The Permanent Default (#13405)

#### Miscellaneous

- Update teku's bootnode (#13437)
- fix metric for exited validator (#13379)
- Fix typos (#13435)
- Replace validator count with validator indices in update fee recipient log (#13384)
- Log value of local payload when proposing (#13381)
- Small encoding fixes on logs and http error code change (#13345)
- typo fix (#13357)
- Fix error string generation for missing commitments (#13338)
- Increase buffer of events channel (#13329)
- Fix missing testnet versions. Issue #13288 (#13323)
- Update README.md (#13302)
- Only run metrics for canonical blocks (#13289)
- Relax file permissions check on existing directories (#13274) 
- forkchoice.Getter wrapper with locking wrappers (#13244)
- Initialize cancellable root context in main.go (#13252)
- Fix forkchoice pkg's comments grammar (#13217)
- lock RecentBlockSlot (#13212)
- Comment typo  (#13209)
- Optimize `ReplayBlocks` for Zero Diff (#13198)
- Remove default value of circuit breaker flags (#13186)
- Fix Withdrawals (#13181)
- Remove no-op cancel func (#13069)
- Update Terms of Service (#13163)
- fix head slot in log (#13139)
- DEPRECTATION: Remove exchange transition configuration call (#13127)
- fix segmentation fork when Capella for epoch is MaxUint64 (#13126)
- Return Error Gracefully When Removing 4881 Flag (#13096)
- Add zero length check on indices during NextSyncCommitteeIndices (#13117)
- Replace Empty Slice Literals with Nil Slices (#13093)
- Refactor Error String Formatting According to Go Best Practices (#13092)
- Fix redundant type converstion (#13076)
- docs: fix typo (#13023)
- Add Clarification To Sync Committee Cache (#13067)
- Fix typos (#13053)
- remove bad comment (#13056)
- Remove confusing comment (#13045)
- Log when sending FCU with payload attributes (#13137)
- Fix Withdrawals Marshalling (#13066)
- beacon-chain/execution: no need to reread and unmarshal the eth1Data twice (#12826)

## [v4.1.1](https://github.com/prysmaticlabs/prysm/compare/v4.1.0...v4.1.1) - 2023-10-24

This patch release includes two cherry-picked changes from the develop branch to resolve critical issues that affect a small set of users.

### Fixed

- Fix improperly registered REST API endpoint for validators using Prysm's REST API with an external builder #13071
- Fix deadlock when using --enable-experimental-state feature #13087

### Security

No security issues in thsi release.

## [v4.1.0](https://github.com/prysmaticlabs/prysm/compare/v4.0.8...v4.1.0) - 2023-08-22

- **Fundamental Deneb Support**: This release lays the foundation for Deneb support, although features like backwards syncing and filesystem-based blob storage are planned for Q4 2024.
- **Multi-Value Slices for Beacon State**: Implemented multi-value slices to reduce the memory footprint and optimize certain processing paths. This data structure allows for storing values shared between state instances more efficiently. This feature is controller by the `--enable-experimental-state` flag.
- **EIP-4881 Deposit Tree**: Integrated the EIP-4881 Deposit Tree into Prysm to optimize runtime block processing and production. This feature is controlled by a flag: `--enable-eip-4881`
- **BLST version 0.3.11**: Introduced a significant improvement to the portable build's performance. The portable build now features runtime detection, automatically enabling optimized code paths if your CPU supports it.
- **Multiarch Containers Preview Available**: multiarch (:wave: arm64 support :wave:) containers will be offered for preview at the following locations:
    - Beacon Chain: [gcr.io/prylabs-dev/prysm/beacon-chain:v4.1.0](gcr.io/prylabs-dev/prysm/beacon-chain:v4.1.0)
    - Validator: [gcr.io/prylabs-dev/prysm/validator:v4.1.0](gcr.io/prylabs-dev/prysm/validator:v4.1.0)
    - Please note that in the next cycle, we will exclusively use these containers at the canonical URLs.

### Added

#### EIP-4844:
##### Core:
- **Deneb State & Block Types**: New state and block types added specifically for Deneb. (#12375, #12368)
- **Deneb Protobufs**: Protocol Buffers designed exclusively for Deneb. (#12363)
- **Deneb Engine API**: Specialized API endpoints for Deneb. (#12384)
- **Deneb Config/Params**: Deneb-specific configurations and parameters from the deneb-integration branch. (#12783)

##### Blob Management:
- **Blob Retention Epoch Period**: Configurable retention periods for blobs. (#12941)
- **Blob Arrival Gossip Metric**: Metrics for blob arrivals via gossip protocol. (#12888)
- **Blob Merge Function**: Functionality to merge and validate saved/new blobs. (#12868)
- **Blob Channel**: A channel dedicated to blob processing. (#12753)
- **Save Blobs to DB**: Feature to save blobs to the database for subscribers. (#12734)

##### Logging and Validation:
- **Logging for Blob Sidecar**: Improved logging functionalities for Blob Sidecar. (#12883)
- **Blob Commitment Count Logging**: Introduced logging for blob commitment counts. (#12723)
- **Blob Validation**: A feature to validate blobs. (#12574)

##### Additional Features and Tests:
- **Deneb Changes & Blobs to Builder**: Deneb-specific changes and blob functionality added to the builder. (#12477)
- **Deneb Blob Sidecar Events**: Blob sidecar events added as part of the Deneb release. (#12928)
- **KZG Commitments**: Functionality to copy KZG commitments when using the builder block. (#12923)
- **Deneb Validator Beacon APIs**: New REST APIs specifically for the Deneb release. (#12871)
- **Deneb Tests**: Test cases specific to the Deneb version. (#12680, #12610)
- **PublishBlockV2 for Deneb**: The `publishblockv2` endpoint implemented specifically for Deneb. (#12662)
- **Builder Override & Builder Flow for Deneb**: An override for the builder and a new RPC to handle the builder flow in Deneb. (#12601, #12554)
- **SSZ Detection for Deneb**: SSZ detection capabilities added for Deneb. (#12537)
- **Validator Signing for Deneb**: Validators can now sign Deneb blocks. (#12449)
- **Deneb Upgrade Function**: A function to handle the upgrade to Deneb. (#12433)

#### Rest of EIPs
- **EIP-4788**: Added support for Beacon block root in the EVM (#12570).
- **EIP-7044** and **EIP-7045**: Implemented support for Perpetually Valid Signed Voluntary Exits and increased the max attestation inclusion slot (#12577, #12565).

#### Beacon API:

*Note: All Beacon API work is related with moving endpoints into pure HTTP handlers. This is NOT new functionality.*

##### Endpoints moved to HTTP:
- `/eth/v1/beacon/blocks` and `/eth/v1/beacon/blinded_blocks` (#12827).
- `/eth/v1/beacon/states/{state_id}/committees` (#12879).
- `/eth/v1/config/deposit_contract` (#12872).
- `/eth/v1/beacon/pool/sync_committees` (#12782).
- `/eth/v1/beacon/states/{state_id}/validators`, `/eth/v1/beacon/states/{state_id}/validators/{validator_id}` and `/eth/v1/beacon/states/{state_id}/validator_balances` (#12887).
- `/eth/v1/validator/duties/attester/{epoch}`, `/eth/v1/validator/duties/proposer/{epoch}` and `/eth/v1/validator/duties/sync/{epoch}` (#12810).
- `/eth/v1/validator/register_validator` (#12758).
- `/eth/v1/validator/prepare_beacon_proposer` (#12781).
- `/eth/v1/beacon/headers` (#12817).
- `/eth/v1/beacon/blocks/{block_id}/root` (#12716).
- `/eth/v1/validator/attestation_data` (#12634).
- `/eth/v1/validator/sync_committee_contribution` (#12698).
- `/eth/v1/beacon/genesis` and `/eth/v1/beacon/states/{state_id}/finality_checkpoints` (#12902).
- `/eth/v1/node/syncing` (#12706).
- `/eth/v1/beacon/pool/voluntary_exits` (#12777).
- `/eth/v1/beacon/headers/{block_id}` and `/eth/v1/validator/liveness/{epoch}` (#12916).

##### Miscellaneous:
- **Comma-Separated Query Params**: Support for comma-separated query parameters added to Beacon API (#12966).
- **Middleware for Query Params**: Middleware introduced for handling comma-separated query parameters (#12995).
- **Content-Type Header**: Compliance improved by adding Content-Type header to VC POST requests (#12942).
- **Node Version**: REST-based node version endpoint implemented (#12809).

#### Other additions
##### Protocol:
- **Multi-Value Slice for Beacon State**: Enhanced the beacon state by utilizing a multi-value slice. (#12549)
- **EIP-4881 Deposit Tree**: EIP-4881 Deposit Tree integrated into Prysm, controlled by a feature flag (#11942).
- **New Engine Methods**: New engine methods set as the default (#12997).
- **Light Client Sync Protocol**: Initiation of a 5-part light client sync protocol (#12853).
- **Block Commitment Checks**: Functionality to reject blocks with excessive commitments added (#12863).

##### State Management:
- **Alloc More Items**: Modified beacon-node/state to allocate an additional item during appends (#12832).
- **GetParentBlockHash Helper**: Refactoring of `getLocalPayloadAndBlobs` with a new helper function for fetching parent block hashes (#12951).
- **RW Lock for Duties**: Read-Write lock mechanism introduced for managing validator duties (#12861).

##### Build and CI/CD Improvements:
- **Manual Build Tag**: A "manual" build tag introduced to expedite CI build times (#12967).
- **Multiarch Docker Containers**: Support for multiple architectures in Docker containers added (#12428).

##### Testing:
- **Init-Sync DA Tests**: Tests for initial sync Data Availability (DA) included (#12873).
- **Fuzz List Timeout**: Github workflow for fuzz testing now includes a timeout setting (#12768).
- **Go Fuzzing Workflow**: New Github workflow for Go fuzzing on a cron schedule (#12756).

##### Logging and Monitoring:
- **FFG-LMD Consistency Logging**: Enhanced logging for Finality Gadget LMD (FFG-LMD) consistency (#12763).
- **Validator Count Endpoint**: New endpoint to count the number of validators (#12752).

##### User Interface and Web:
- **Web UI Release**: Prysm Web UI v2.0.4 released with unspecified updates and improvements (#12746).

##### Testnet support:
- **Holesky Support**: Support for Holesky decompositions integrated into the codebase (#12821).

##### Error Handling and Responses:
- **Validation Error in ForkchoiceUpdatedResponse**: Included validation errors in fork choice update responses (#12828).
- **Wrapped Invalid Block Error**: Improved error handling for cases where an invalid block error is wrapped. (#12982).

### Changed

#### General:
- **Skip MEV-Boost Flag**: Updated `GetBlock` RPC to utilize `skip mev-boost` flag (#12969).
- **Portable Version of BLST**: Transitioned to portable BLST version as default (#12720).
- **Teku Mainnet Bootnodes**: Refreshed Teku mainnet bootnodes ENRs (#12962).
- **Geth Version Updates**: Elevated geth to version v1.13.1 for additional stability and features (#12911).
- **Parallel Block Building**: Deprecated sequential block building path (#13008)

#### Deneb-Specific Changes:
- **Deneb Spectests Release**: Upgraded to Deneb spectests v1.4.0-beta.2-hotfix (#12959).
- **Deneb API and Builder Cleanup**: Conducted clean-up activities for Deneb-specific API and builder (#12852, #12921).
- **Deneb Block Versioning**: Introduced changes related to Deneb produce block version 3 (#12708).
- **Deneb Database Methods**: Adapted database methods to accommodate Deneb (#12379).
- **Unused Code Removal**: Eliminated an unused function and pending blobs queue (#12920, #12913).
- **Blob Sidecar Syncing**: Altered behavior when value is 0 (#12892).

#### Code Cleanup and Refactor:
- **API Types Cleanup**: Reorganized API types for improved readability (#12961).
- **Geth Client Headers**: Simplified code for setting geth client headers (#11748).
- **Bug Report Template**: Revised requirements for more clarity (#12937, #12891).

#### Flags and Configuration:
- **Safe Slots to Import Flag**: Deprecated this flag for standard alignment (#12964).
- **Holesky Config**: Revised the Holesky configuration for new genesis (#12919).

#### Logging:
- **Genesis State Warning**: Will log a warning if the genesis state size is under 1KB (#12897).
- **Debug Log Removal**: Excised debug logs for cleaner output (#12836).

#### Miscellaneous:
- **First Aggregation Timing**: Default setting for first aggregation is 7 seconds post-genesis (#12876).
- **Pointer Usage**: Modified execution chain to use pointers, reducing copy operations (#12818).

#### Dependency Updates:
- **Go Version Update**: Updated to Go version 1.20.7 (#12707).
- **Go Version Update**: Updated to Go version 1.20.9 for better security. (#13009)
- **Various Dependencies**: Updated multiple dependencies including Geth, Bazel, rules_go, Gazelle, BLST, and go-libp2p (#12731, #12725, #12721, #12718, #12717, #12709).

### Removed

- **Remote Slashing Protection**: Eliminated the remote slashing protection feature (#12989).
- **Go-Playground/Validator**: Removed the go-playground/validator dependency from the Beacon API (#12973).
- **Revert Cache Proposer ID**: Reverted the caching of proposer ID on GetProposerDuties (#12986).
- **Go-Playground/Validator**: Removed go-playground/validator from Beacon API (#12973).
- **Reverted Cache Proposer ID**: Reversed the change that cached proposer ID on GetProposerDuties (#12986).
- **Cache Proposer ID**: Reversed the functionality that cached proposer ID on GetProposerDuties (#12939).
- **Quadratic Loops in Exiting**: Eliminated quadratic loops that occurred during voluntary exits, improving performance (#12737).
- **Deprecated Go Embed Rules**: Removed deprecated `go_embed` rules from rules_go, to stay up-to-date with best practices (#12719).
- **Alpine Images**: Removed Alpine images from the Prysm project (#12749).

### Fixed

#### Deneb-Specific Bug Fixes:
- **Deneb Builder Bid HTR**: Fixed an issue related to HashTreeRoot (HTR) in Deneb builder bid (#12906).
- **PBV2 Condition**: Corrected conditions related to PBV2 (#12812).
- **Route Handler and Cleanup**: Updated the route handler and performed minor cleanups (#12726).
- **Devnet6 Interop Issues**: Resolved interoperability issues specific to Devnet6 (#12545).
- **Sepolia Version**: Updated the version information for the Sepolia testnet (#12792).
- **No Blob Bundle Handling**: Rectified the handling when no blob bundle exists (#12838).
- **Blob Sidecar Prefix**: Corrected the database prefix used for blob sidecars (#12849).
- **Blob Retrieval Error**: Added specific error handling for blob retrieval from the database (#12889).
- **Blob Sidecar Count**: Adjusted metrics for accurate blob sidecar count (#12865).
- **Sync/RPC Blob Usage**: Rectified blob usage when requesting a block by root in Sync/RPC (#12837).

#### Cache Fixes:
- **Don't Prune Proposer ID Cache**: Fixed a loop erroneously pruning the proposer ID cache (#12996).
- **LastRoot Adjustment**: Altered `LastRoot` to return the head root (#12985).
- **Last Canonical Root**: Modified forkchoice to return the last canonical root of the epoch (#12954).

#### Block Processing fixes:
- **Block Validation**: Fixed an issue where blocks were incorrectly marked as bad during validation (#12983).
- **Churn Limit Helpers**: Improved churn limit calculations through refactoring (#12971).
- **Churn with 0 Exits**: Rectified a bug that calculated churn even when there were 0 exits (#12976).
- **Proposer Duties Sorting**: Resolved sorting issues in proposer duties (#12909).
- **Duplicate Block Processing**: Eliminated redundant block processing (#12905).

#### Error Handling and Logging:
- **RpcError from Core Service**: Ensured that `RpcError` is returned from core services (#12974).
- **Unhandled Error**: Enhanced error management by handling previously unhandled errors (#12938).
- **Error Handling**: Wrapped `ctx.Err` for improved error handling (#12859).
- **Attestation Error**: Optimized error management in attestation processing (#12813).

#### Test and Build Fixes:
- **Racy Tests in Blockchain**: Resolved race conditions in blockchain tests (#12957).
- **TestService_ReceiveBlock**: Modified `TestService_ReceiveBlock` to work as expected (#12953).
- **Build Issue with @com_github_ethereum_c_kzg_4844**: Resolved build issues related to this specific library (#12890).
- **Fuzz Testing**: Addressed fuzz testing issues in the `origin/deneb-integration` 
- **Long-Running E2E Tests**: Fixed issues that were causing the end-to-end tests to run for an extended period. (#13000)

#### Additional Fixes:
- **Public Key Copies During Aggregation**: Optimized to avoid unnecessary public key copies during aggregation (#12944).
- **Epoch Participations**: Fixed the setting of current and previous epoch participations (#12814).
- **Verify Attestations**: Resolved an attestation verification issue in proposer logic (#12704).
- **Empty JSON/YAML Files**: Fixed an issue where `prysmctl` was writing empty configuration files (#12599).
- **Generic Fixes**: Addressed various unspecified issues (#12932, #12917, #12915).
- **Phase0 Block Parsing**: Resolved parsing issues in phase0 blocks on submit (#12857).
- **Hex Handling**: Upgraded the hex handling in various modules (#12979).
- **Initial Sync PreProcessing**: Resolved an issue affecting the initial sync preprocessing. (#13007)


### Security

No security updates in this release.

## [v4.0.8](https://github.com/prysmaticlabs/prysm/compare/v4.0.7...v4.0.8) - 2023-08-22

Welcome to Prysm Release v4.0.8! This release is recommended. Highlights:

- Parallel hashing of validator entries in the beacon state. This results in a faster hash tree root. ~3x reduction #12639
- Parallel validations of consensus and execution checks. This results in a faster block verification #12590
- Aggregate parallel is now the default. This results in faster attestation aggregation time if a node is subscribed to multiple beacon attestation subnets. ~3x reduction #12699
- Better process block epoch boundary cache usages and bug fixes
- Beacon-API endpoints optimizations and bug fixes

### Added

- Optimization: parallelize hashing for validator entries in beacon state #12639
- Optimization: parallelize consensus & execution validation when processing beacon block #12590
- Optimization: integrate LRU cache (above) for validator public keys #12646
- Cache: threadsafe LRU with non-blocking reads for concurrent readers #12476
- PCLI: add deserialization time in benchmark #12620
- PCLI: add allocation data To benchmark #12641
- Beacon-API: GetSyncCommitteeRewards endpoint #12633
- Beacon-API: SSZ responses for the Publishblockv2 #12636
- Beacon-API client: use GetValidatorPerformance #12581
- Spec tests: mainnet withdrawals and bls spec tests #12655
- Spec tests: random and fork transition spec tests #12681
- Spec tests execution payload operation tests #12685
- Metric: block gossip arrival time #12670
- Metric: state regen duration #12672
- Metric: validator is in the next sync committee #12650
- New data structure: multi-value slice #12616

### Changed

- Build: update Go version to 1.20.6 #12617
- Build: update hermetic_cc_toolchain #12631
- Optimization: aggregate parallel is now default #12699
- Optimization: do not perform full copies for metrics reporting #12628
- Optimization: use GetPayloadBodies in Execution Engine Client #12630
- Optimization: better nil check for reading validator #12677
- Optimization: better cache update at epoch boundary #12679
- Optimization: improve InnerShuffleList for shuffling #12690
- Optimization: remove span for converting to indexed attestation #12687`
- Beacon-API: optimize GetValidatorPerformance as POST #12658
- Beacon-API: optimize /eth/v1/validator/aggregate_attestation #12643
- Beacon-API: optimize /eth/v1/validator/contribution_and_proofs #12660
- Beacon-API: optimize /eth/v1/validator/aggregate_and_proofs #12686
- Beacon-API: use struct in beacon-chain/rpc/core to store dependencies #12701
- Beacon-API: set CoreService in beaconv1alpha1.Server #12702
- Beacon-API: use BlockProcessed event in certain endpoints #12625
- Syncing: exit sync early with 0 peers to sync #12659
- Cache: only call epoch boundary processing on canonical blocks #12666
- Build: update server-side events dependency #12676
- Refactor: slot tickers with intervals #12440
- Logging: shift Error Logs To Debug #12739
- Logging: clean up attestation routine logs #12653

### Fixed

- Cache: update shuffling caches at epoch boundary #12661
- Cache: committee cache correctly for epoch + 1 #12667 #12668
- Cache: use the correct context for UpdateCommitteeCache #12691
- Cache: proposer-settings edge case for activating validators #12671
- Cache: prevent the public key cache from overwhelming runtime #12697
- Sync: correctly set optimistic status in the head when syncing #12748
- Sync: use last optimistic status on batch #12741
- Flag: adds local boost flag to main/usage #12615
- Beacon-API: correct header for get block and get blinded block calls #12600
- Beacon-API: GetValidatorPerformance endpoint #12638
- Beacon-API: return correct historical roots in Capella state #12642
- Beacon-API: use the correct root in consensus validation #12657
- Prysm API: size of SyncCommitteeBits #12586
- Mev-boost: builder gas limit fix default to 0 in some cases #12647
- PCLI: benchmark deserialize without clone and init trie #12626
- PCLI: state trie for HTR duration #12629
- Metric: adding fix pending validators balance #12665
- Metric: effective balance for unknown/pending validators #12693
- Comment: comments when receiving block #12624
- Comment: cleanups to blockchain pkg #12640

### Security

No security updates in this release.

## [v4.0.7](https://github.com/prysmaticlabs/prysm/compare/v4.0.6...v4.0.7) - 2023-07-13

Welcome to the v4.0.7 release of Prysm! This recommended release contains many essential optimizations since v4.0.6.

Highlights:

- The validator proposal time for slot 0 has been reduced by 800ms. Writeup and PR
- The attestation aggregation time has been reduced by 400ms—roughly 75% with all subnets subscribed. Flag --aggregate-parallel. PR. This is only useful if running more than a dozen validator keys. The more subnets your node subscribe to, the more useful.
- The usage of fork choice lock has been reduced and optimized, significantly reducing block processing time. This results in a higher proposal and attest rate. PR
- The block proposal path has been optimized with more efficient copies and a better pruning algorithm for pending deposits. PR and PR
- Validator Registration cache is enabled by default, this affects users who have used webui along with mevboost. Please review PR for details.

Note: We remind our users that there are two versions of the cryptographic library BLST, one is "portable" and less performant, and another is "non-portable" or "modern" and more performant. Most users would want to use the second one. You can set the environment variable USE_PRYSM_MODERN=true when using prysm.sh. The released docker images are using the non-portable version by default.

### Added

- Optimize multiple validator status query #12487
- Track optimistic status on head #12552
- Get attestation rewards API end point #12480
- Expected withdrawals API #12519
- Validator voluntary exit endpoint #12299
- Aggregate atts using fixed pool of go routines #12553
- Use the incoming payload status instead of calling forkchoice #12559
- Add hermetic_cc_toolchain for a hermetic cc toolchain #12135
- Cache next epoch proposers at epoch boundary #12484
- Optimize Validator Roots Computation #12585
- Log Finalized Deposit Insertion #12593
- Move consensus and execution validation outside of onBlock #12589
- Add metric for ReceiveBlock #12597
- Prune Pending Deposits on Finalization #12598
- GetValidatorPerformance http endpoint #12557
- Block proposal copy Bytes Alternatively #12608
- Append Dynamic Adding Trusted Peer Apis #12531

### Changed

- Do not validate merge transition block after Capella #12459
- Metric for balance displayed for public keys without validator indexes #12535
- Set blst_modern=true to be the bazel default build #12564
- Rename payloadHash to lastValidHash in setOptimisticToInvalid #12592
- Clarify sync committee message validation #12594
- Checkpoint sync ux #12584
- Registration Cache by default #12456

### Removed

- Disable nil payloadid log on relayers flags #12465
- Remove unneeded helper #12558
- Remove forkchoice call from notify new payload #12560

### Fixed

- Late block task wait for initial sync #12526
- Log the right block number #12529
- Fix for keystore field name to align with EIP2335 #12530
- Fix epoch participation parsing for API #12534
- Spec checker, ensure file does not exit or error #12536
- Uint256 parsing for builder API #12540
- Fuzz target for execution payload #12541
- Contribution doc typo #12548
- Unit test TestFieldTrie_NativeState_fieldConvertersNative #12550
- Typo on beacon-chain/node/node.go #12551
- Remove single bit aggregation for aggregator #12555
- Deflake cloners_test.go #12566
- Use diff context to update proposer cache background #12571
- Update protobuf and protobuf deps #12569
- Run ineffassign for all code #12578
- Increase validator client startup proposer settings deadline #12533
- Correct log level for 'Could not send a chunked response' #12562
- Rrune invalid blocks during initial sync #12591
- Handle Epoch Boundary Misses #12579
- Bump google.golang.org/grpc from 1.40.0 to 1.53.0 #12595
- Fix bls signature batch unit test #12602
- Fix Context Cancellation for insertFinalizedDeposits #12604
- Lock before saving the poststate to db #12612

### Security

No security updates in this release.

## [v4.0.6](https://github.com/prysmaticlabs/prysm/compare/v4.0.5...v4.0.6) - 2023-07-15

Welcome to v4.0.6 release of Prysm! This recommended release contains many essential optimizations since v4.0.5. Notable highlights:

Better handling of state field trie under late block scenario. This improves the next slot proposer's proposed time
Better utilization of next slot cache under various conditions

**Important read:**

1.) We use this opportunity to remind you that two different implementations of the underlying cryptographic library BLST exist.

- portable: supports every CPU made in the modern era
- non-portable: more performant but requires your CPU to support special instructions

Most users will want to use the "non-portable" version since most CPUs support these instructions. Our docker builds are now non-portable by default. Most users will benefit from the performance improvements. You can run with the "portable" versions if your CPU is old or unsupported. For binary distributions and to maintain backward compatibility with older versions of prysm.sh or prysm.bat, users that want to benefit from the non-portable performance improvements need to add an environment variable, like so: USE_PRYSM_MODERN=true prysm.sh beacon-chain prefix, or download the "non-portable" version of the binaries from the github repo.

2.) A peering bug that led to nodes losing peers gradually and eventually needing a restart has been patched. Nodes previously affected by it can remove the --disable-resource-manager flag from v4.0.6 onwards.

### Added

- Copy state field tries for late block #12461
- Utilize next slot cache correctly under late block scenario #12462
- Epoch boundary uses next slot cache #12515
- Beacon API broadcast_validation to block publishing #12432
- Appropriate Size for the P2P Attestation Queue #12485
- Flag --disable-resource-manager to disable resource manager for libp2p #12438
- Beacon RPC start and end block building time logs #12452
- Prysmctl: output proposer settings #12181
- Libp2p patch #12507
- Handle trusted peers for libp2p #12492
- Spec test v1.4.0-alpha.1 #12489

### Changed

- Use fork-choice store to validate sync message faster #12430
- Proposer RPc unblind block workflow #12240
- Restore flag disable-peer-scorer #12386
- Validator import logs improvement #12429
- Optimize zero hash comparisons in forkchoice #12458
- Check peer threshold is met before giving up on context deadline #12446
- Cleanup of proposer payload ID cache #12474
- Clean up set execution data for proposer RPC #12466
- Update Libp2p to v0.27.5 #12486
- Always Favour Yamux for Multiplexing #12502
- Ignore Phase0 Blocks For Monitor #12503
- Move hash tree root to after block broadcast #12504
- Use next slot cache for sync committee #12287
- Log validation time for blocks #12514
- Change update duties to handle all validators exited check #12505
- Ignore late message log #12525

### Removed

- SubmitblindBlock context timeout #12453
- Defer state feed In propose block #12524

### Fixed

- Sandwich attack on honest reorgs #12418 #12450
- Missing config yamls for specific domains #12442
- Release lock before panic for feed #12464
- Return 500 in `/eth/v1/node/peers` interface #12483
- Checkpoint sync uses correct slot #12447

### Security

No security updates in this release.

## [v4.0.5](https://github.com/prysmaticlabs/prysm/compare/v4.0.4...v4.0.5) - 2023-05-22

Welcome to v4.0.5 release of Prysm! This release contains many important improvements and bug fixes since v4.0.4, including significant improvements to attestation aggregation. See @potuz's notes [here](https://hackmd.io/TtyFurRJRKuklG3n8lMO9Q). This release is **strongly** recommended for all users.

Note: The released docker images are using the portable version of the blst cryptography library. The Prysm team will release docker images with the non-portable blst library as the default image. In the meantime, you can compile docker images with blst non-portable locally with the `--define=blst_modern=true` bazel flag, use the "-modern-" assets attached to releases, or set environment varaible USE_PRYSM_MODERN=true when using prysm.sh.

### Added

- Added epoch and root to "not a checkpt in forkchoice" log message #12400
- Added cappella support for eth1voting tool #12402
- Persist validator proposer settings in the validator db. #12354
- Add flag to disable p2p resource management. This flag is for debugging purposes and should not be used in production for extended periods of time. Use this flag if you are experiencing significant peering issues. --disable-resource-manager #12438

### Changed

- Improved slot ticker for attestation aggregation #12377 #12412 #12417
- Parallel block production enabled by default. Opt out with --disable-build-block-parallel if issues are suspected with this feature. #12408
- Improve attestation aggregation by not using max cover on unaggregated attestations and not checking subgroup of previously validated signatures. #12350
- Improve sync message processing by using forkchoice #12430 #12445

### Fixed

- Fixed --slasher flag. #12405
- Fixed state migration for capella / bellatrix #12423
- Fix deadlock when using --monitor-indices #12427

### Security

No security updates in this release.

## [v4.0.4](https://github.com/prysmaticlabs/prysm/compare/v4.0.3...v4.0.4) - 2023-05-15

Welcome to v4.0.4 release of Prysm! This is the first full release following the recent mainnet issues and it is very important that all stakers update to this release as soon as possible.

Aside from the critical fixes for mainnet, this release contains a number of new features and other fixes since v4.0.3.

### Added

- Feature to build consensus and execution blocks in parallel. This feature has shown a noticeable reduction (~200ms) in block proposal times. Enable with --build-block-parallel #12297
- An in memory cache for validator registration can be enabled with --enable-registration-cache. See PR description before enabling. #12316
- Added new linters #12273 #12270 #12271
- Improved tracing data for builder pipeline #12302 #12332
- Improved withdrawal phrasing in validator withdrawal tooling #12306
- Improved blinded block error message #12310 #12309
- Added test for future slot tolerance #12344
- Pre-populate bls pubkey cache #11482
- Builder API support in E2E tests #12343

### Changed

- Updated spectests to v1.3 #12300
- Cleanup duplicated code #12304
- Updated method signature for UnrealizedJustifiedPayloadBlockHash() #12314
- Updated k8s.io/client-go to 0.20.0 #11972
- Removed unused method argument #12327
- Refactored / moved some errors to different package #12329
- Update next slot cache at an earlier point in block processing #12321
- Use next slot cache for payload attribute #12286
- Cleanup keymanager mock #12341
- Update to go 1.20 #12333
- Modify InsertFinalizedDeposits signature to return an error #12342
- Improved statefeed initialization #12285
- Use v1alpha1 server in block production #12336
- Updated go generated files #12359
- Typo corrections #12385

### Fixed

- Fixed e2e tx fuzzer nilerr lint issue #12313
- Fixed status for pending validators with multiple deposits #12318
- Use gwei in builder value evaluation #12291 #12370
- Return correct error when failing to unmarshal genesis state #12325
- Avoid double state copy in latestAncestor call #12326
- Fix mock v1alpha1 server #12319
- Fix committee race test #12338
- Fix flaky validator tests #12339
- Log correctly when the forkchoice head changed #12324
- Filter inactive keys from mev-boost / builder API validator registration #12322 #12358
- Save attestation to cache when calling SubmitAttestation in beacon API #12345
- Avoid panic on nil broadcast object #12369
- Fix initialization race #12374
- Properly close subnet iterator #12388
- ⚠️ Ignore untimely attestations #12387
- Fix inverted metric #12392
- ⚠️ Save to checkpoint cache if next state cache hits #12398

### Security

This release contains some important fixes that improve the resiliency of Ethereum Consensus Layer. See https://github.com/prysmaticlabs/prysm/pull/12387 and https://github.com/prysmaticlabs/prysm/pull/12398.

## [v4.0.3](https://github.com/prysmaticlabs/prysm/compare/v4.0.2...v4.0.3) - 2023-04-20

### Added

- Add REST API endpoint for beacon chain client's GetChainHead #12245
- Add prepare-all-payloads flag #12260
- support modifying genesis.json for capella #12283
- Add support for engine_exchangeCapabilities #12224
- prysmctl: Add support for writing signed validator exits to disk #12262

### Changed

- Enable misspell linter & fix findings #12272

### Fixed

- Fix Panic In Builder Service #12277
- prysmctl using the same genesis func as e2e #12268
- Check that Builder Is Configured #12279
- Correctly use Gwei to compare builder bid value #12290
- Fix Broken Dependency #12293 #12294
- Deflake TestWaitForActivation_AccountsChanged #12282
- Fix Attester Slashing Validation In Gossip #12295
- Keymanager fixes for bad file writes #12284
- windows: Fix build after PR 12293 #12296

### Security

No security updates in this release.

## [v4.0.2](https://github.com/prysmaticlabs/prysm/compare/v4.0.1...v4.0.2) - 2023-04-12

This release fixes a critical bug on Prysm interacting with mev-boost / relayer. You MUST upgrade to this release if you run Prysm with mev boost and relayer, or you will be missing block proposals during the first days after the Shapella fork while the block has bls-to-exec changes.
Post-mortem that describes this incident will be provided by the end of the week.

One of this release's main optimizations is revamping the next slot cache. It has been upgraded to be more performant across edge case re-org scenarios. This can help with the bad head attestation vote.

Minor fixes in this release address a bug that affected certain large operators querying RPC endpoints. This bug caused unexpected behavior and may have impacted the performance of affected operators. To resolve this issue, we have included a patch that ensures proper functionality when querying RPC endpoints.

### Added

- CLI: New beacon node flag local-block-value-boost that allows the local block value to be multiplied by the boost value #12227
- Smart caching for square root computation #12191
- Beacon-API: Implemented Block rewards endpoint #12020
- Beacon-API client: Implemented GetSyncStatus endpoint #12189
- Beacon-API client: Implemented GetGenesis endpoint #12168
- Beacon-API client: Implemented ListValidators  endpoint #12228

### Changed

- Block processing: Optimize next slot cache #12233 #12247
- Execution-API: Used unrealized justified block hash for FCU call #12196
- CLI: Improved voluntary exit confirmation prompt #12205
- Unit test: Unskip API tests #12222
- End to end test: Misc improvements #12211 #12207
- Build: Build tag to exclude mainnet genesis from prysmctl #12244
- Dependency: Update go-ethereum to v1.11.3 #12204
- Dependency: Update lighthouse to v4.0.1 #12204

### Fixed

- Builder: Unblind beacon block correctly with bls-to-exec changes #12263
- Block construction: Default to local payload on error correctly #12243
- Block construction: Default to local payload on nil value correctly #12236
- Block processing: Fallback in update head on error #12199
- Block processing: Add orphaned operations to the appropriate pool #12249
- Prysm-API: Fix Deadlock in StreamChainHead #12250
- Beacon-API: Get header error, nil summary returned from the DB #12214
- Beacon-API: Broadcast correct slashing object #12230

### Security

No security updates in this release.

## [v4.0.1](https://github.com/prysmaticlabs/prysm/compare/v4.0.0...v4.0.1)

This is a reissue of v4.0.0. See https://github.com/prysmaticlabs/prysm/issues/12201 for more information.

## [v4.0.0](https://github.com/prysmaticlabs/prysm/compare/v3.2.2...v4.0.0)

### Added

- Config: set mainnet capella epoch #12144
- Validator: enable proposer to reorg late block #12075 #121100
- Metric: bls-to-exec count in the operation pool #12133 #12155
- Metric: pubsub metrics racer #12178
- Metric: add late block metric #12091
- Engine-API: Implement GetPayloadBodies #11973
- Beacon-API: Implement GetPayloadAttribute SSE #12102 #12154 #12160 #12169
- Prysm CLI: add experimental flags to dev mode #12152
- Prysmctl utility: add eth1data to genesis state #12125
- Spec test: EIP4881 spec compliance tests #11754
- Spec test: forkchoice lock to fix flaskyness #12165

### Changed

- Prysm: upgrade v3 to v4 #12134
- Prysm: apply goimports to generated files #12170
- Validator: lower builder circuit breaker thresholds to 5 missed slots per epoch and updates off by 1 #12076
- Validator: reorg late block by default #12146 #12147
- Forkchoice: cleanups #12078
- Forkchoice: remove bouncing attack fix and strength equivocation discarding #12126
- Forkchoice: call FCU at 4s mark if there's no new head #12159
- Forkchoice: better locking on calls to retrieving ancestor root #12162
- Forkchoice: stricker visibility for blockchain package access #12174
- Block processing: optimizing validator balance retrieval by using epoch boundary cache #12083
- Block processing: reduce FCU calls #12091
- Block processing: increase attempted reorgs at the correct spot #12106
- Block processing: remove duplicated bls to exec message pruning #12085
- Block processing: skip hash tree root state when checking optimistic mode #12143
- Prysm-API: mark GetChainHead deprecated #12128
- Logging: add late block logs #12091
- Logging: enhancements and clean ups #12086
- Build: fix bazel remote cache upload #12108
- Build: update cross compile toolchains #12069
- Build: only build non-test targets in hack/update-go-pbs.sh #12101
- Build: update rules_go to v0.38.1 and go_version to 1.19.7 #12055
- Build: replace bazel pkg_tar rule with canonical @rules_pkg pkg_tar #12120
- Build: update bazel to 6.1.0 #12121
- Libp2p: updated to latest version #12096 #12132
- Libp2p: make peer scorer permanent default #12138
- Test: disable e2e slasher test #12150
- CLI: derecate the following flags #12148 #12151


### Deprecated

The following flags have been deprecated.

- disable-peer-scorer
- disable-vectorized-htr
- disable-gossip-batch-aggregation

### Removed

- Prsym remote signer #11895
- CLI: Prater feature flag #12082
- CLI: Deprecated flags #12139 #12140 #12141
- Unit test: unused beacon chain altair mocks #12095
- Validator REST API: unused endpoints #12167

The following flags have been removed.

- http-web3provider
- enable-db-backup-webhook
- bolt-mmap-initial-size
- disable-discv5
- enable-reorg-late-blocks
- disable-attesting-history-db-cache
- enable-vectorized-htr
- enable-peer-scorer
- enable-forkchoice-doubly-linked-tree
- enable-back-pull
- enable-duty-count-down
- head-sync
- enable-gossip-batch-aggregation
- enable-larger-gossip-history
- fallback-web3provider
- disable-native-state
- enable-only-blinded-beacon-blocks
- ropsten
- interop-genesis-state
- experimental-enable-boundary-checks
- disable-back-pull
- disable-forkchoice-doubly-linked-tree

### Fixed

- Validator: startup deadline #12049
- Prysmctl: withdrawals fork checking logic #12130
- End-to-end test: fix flakes #12074
- End-to-end test: fix altair transition #12124
- Unit test: fix error message in #12123

### Security

This release is required to participate in the Capella upgrade.

## [v3.2.2](https://github.com/prysmaticlabs/prysm/compare/v3.2.2...v3.2.1) - 2023-05-10

Gm! ☀️ We are excited to announce our release for upgrading Goerli testnet to Shanghai / Capella! 🚀

This release is MANDATORY for Goerli testnet. You must upgrade your Prysm beacon node and validator client to this release before Shapella hard fork time epoch=162304 or UTC=14/03/2023, 10:25:36 pm.

This release is a low-priority for the mainnet.
This release is the same commit as v3.2.2-rc.3. If you are already running v3.2.2-rc.3, then you do not need to update your client.

### Added

- Capella fork epoch #12073
- Validator client REST implementation GetFeeRecipientByPubKey #11991
- New end-to-end test for post-attester duties #11899

### Changed

- Storing blind beacon block by default for new Prysm Database #11591
- Raise the max grpc message size to a very large value by default #12072
- Update rules docker to v0.25.0 #12054
- Update distroless base images #12061
- Update protoc-gen-go-cast to suppress tool output #12062
- Update deps for Capella #12067
- Remove gRPC fallback client from validator REST API #12051
- Prysmctl now verifies capella fork for bls to exec message change #12039
- Core block processing cleanup #12046 #12068
- Better locking design around forkchoice store #12036
- Core process sync aggregate function returns reward amount #12047
- Use Epoch boundary cache to retrieve balances #12083
- Misc end-to-end test improvements and fixes #12059
- Add slot number to proposal error log #12071

### Deprecated

- Deprecate flag --interop-genesis-state #12008

### Removed

- Remove Ropsten testnet config and feature flag #12058 #12082

### Security

This release is required for Goerli to upgrade to Capella.

## [v3.2.1](https://github.com/prysmaticlabs/prysm/compare/v3.2.0...v3.2.1) - 2023-02-13

We are excited to announce the release of Prysm v3.2.1 🎉

This is the first release to support Capella / Shanghai. The Sepolia testnet Capella upgrade time is currently set to 2/28/2023, 4:04:48 AM UTC. The Goerli testnet and Mainnet upgrade times are still yet to be determined. In Summary:

- This is a mandatory upgrade for Sepolia nodes and validators
- This is a recommended upgrade for Goerli and Mainnet nodes and validators

There are some known issues with this release.

- mev-boost, relayer, and builder support for Capella upgrade are built in but still need to be tested. Given the lack of testing infrastructure, none of the clients could test this for withdrawals testnet. There may be hiccups when using mev-boost on the Capella upgraded testnets.

### Added

- Capella Withdrawal support #11718 #11751 #11759 #11732 #11796 #11775 #11773 #11760 #11801 #11762 #11822 #11845 #11854 #11848 #11842 #11863 #11862 #11872 #11870 #11878 #11866 #11865 #11883 #11888 #11879 #11896 #11902 #11790 #11904 #11923 #11873 #11936 #11932 #11949 #11906 #11959
- Add Capella fork epoch for Sepolia #11979
- Various Validator client REST implementations (Part of EPF) #11671 #11731 #11757 #11772 #11784 #11785 #11786 #11816 #11827 #11824 #11826 #11812 #11800 #11875 #11835 #11893
- Various Beacon API additions #11788 #11806 #11818 #11849 #11874
- Cache Fork Digest Computation to save compute #11931
- Beacon node can bootstrap from non-genesis state (i.e bellatrix state) #11746
- Refactor bytesutil, add support for go1.20 slice to array conversions #11838
- Add Span information for attestation record save request #11742
- Matric addition #11860
- Identify invalid signature within batch verification #11582 #11741
- Support for getting consensus values from beacon config #11798
- EIP-4881: Spec implementation #11720
- Test helper to generate valid bls-to-exec message #11836
- Spec tests v1.3.0 rc.2 #11929

### Changed

- Prysm CLI utility support for exit #11735
- Beacon API improvement #11789 #11947
- Prysm API get block RPC #11834
- Prysm API cleanups #11808 #11829 #11828
- Block processing cleanup #11764, #11763 #11795 #11796 #11833 #11858
- Forkchoice logging improvements #11849
- Syncing logging improvement #11851
- Validator client set event improvement for readability and error handling #11797
- Engine API implementation cleanups #11717
- End to end test improvements #11704 #11730 #11733 #11738 #11734 #11802 #11810 #11811 #11850 #11856 #11867
- Prysm CLI withdrawal ux improvement #11909
- Better log for the block that never became head #11917

### Removed

- Remove cache lookup and lock request for database boltdb transaction #11745

### Fixed

- Beacon API #11755 #11749 #11723 #11783
- Use the correct attribute if there's a payload ID cache miss #11918
- Call FCU with an attribute on non-head block #11919
- Sparse merkle trie bug fix #11778
- Waiting For Bandwidth Issue While Syncing #11853
- State Fetcher to retrieve correct epoch #11820
- Exit properly with terminal block hash #11892
- PrepareBeaconProposer API duplicating validator indexes when not persisted in DB #11912
- Multiclient end-to-end #11803
- Deep source warnings #11814

### Security

There are no security updates in this release.

## [v3.2.0](https://github.com/prysmaticlabs/prysm/compare/v3.1.2...v3.2.0) - 2022-12-16

This release contains a number of great features and improvements as well as progress towards the upcoming Capella upgrade. This release also includes some API changes which are reflected in the minor version bump. If you are using mev-boost, you will need to update your prysm client to v3.2.0 before updating your mev-boost instance in the future. See [flashbots/mev-boost#404](https://github.com/flashbots/mev-boost/issues/404) for more details.

### Added

- Support for non-english mnemonic phrases in wallet creation. #11543
- Exit validator without confirmation prompt using --force-exit flag #11588
- Progress on Capella and eip-4844 upgrades #11566 #11596 #11601 #11614 #11618 #11624 #11634 #11638 #11644 #11646 #11655 #11647 #11656 #11657 #11661 #11662 #11660 #11658 #11631 #11686 #11688 #11684 #11689 #11690 #11687 #11683 #11713 #11716
- Added randao json endpoint. /eth/v1/beacon/states/{state_id}/randao #11609 #11650 #11710 #11708
- Added liveness endpoint /eth/v1/validator/liveness/{epoch} #11617
- Progress on adding json-api support for prysm validator #11612 #11633 #11654 #11682 #11679 #11695 #11712
- Prysmctl can now generate genesis.ssz for forks after phase0. #11677

### Changed

- --chain-config-file now throws an error if used concurrently with --network flag. #10863
- Added more histogram metrics for block arrival latency times block_arrival_latency_milliseconds #11589
- Priority queue RetrieveByKey now uses read lock instead of write lock #11603
- Use custom types for certain ethclient requests. Fixes an issue when using prysm on gnosis chain. #11586
- Updted forkchoice endpoint /eth/v1/debug/forkchoice (was /eth/v1/debug/beacon/forkchoice) #11680
- Include empty fields in builder json client. #11673
- Computing committee assignments for slots older than the oldest historical root in the beacon state is now forbidden #11722

### Removed

- Deprecated protoarray tests have been removed #11607

### Fixed

- Unlock pending block queue if there is any error on inserting a block #11600
- Prysmctl generate-genesis yaml file now uses the correct format #11635
- ENR serialization now correctly serializes some inputs that did not work previously #11648
- Use finalized block hash if a payload ID cache miss occurs #11653
- prysm.sh now works correctly with Mac M1 chips (it downloads darwin-arm64 binaries) #11675
- Use the correct block root for block events api #11666
- Users running a VPN should be able to make p2p dials. #11599
- Several minor typos and code cleanups #11572 #11594 #11593 #11598 #11602 #11622 #11705 #11670 #11711 #11726

### Security

- Go is updated to 1.19.4. #11630 #11727

## [v3.1.2](https://github.com/prysmaticlabs/prysm/compare/v3.1.1...v3.1.2) - 2022-10-27

### Added

- Timestamp field to forkchoice node json responses #11394 #11403
- Further tests to non-trivial functions of the builder service #11214
- Support for VotedFraction in forkchoice #11421
- Metrics for reorg distance and depths #11435
- Support for optimistic sync spectests #11391
- CLI flag for customizing engine endpoint timeout --engine-endpoint-timeout-seconds #11489
- Support for lodestar identification in p2p monitoring #11536
- --enable-full-ssz-data-logging to display debug ssz data on gossip messages that fail validation #11524
- Progress on capella and withdrawals support #11552 #11563 #11141 #11569
- Validator exit can be performed from prysmctl #11515 #11575
- Blinded block support through the json API #11538

### Changed

- Refactoring / cleanup of keymanager #11331
- Refactoring / improvements in initial sync #11344
- Forkchoice hardening #11397 #11402
- Improved log warnings when fee recipient is not set #11395
- Changed ready for merge log frequency to 1 minute #11410
- Move log Unable to cache headers for execution client votes to debug #11398
- Rename field in invalid pruned blocks log #11411
- Validate checkpoint slot #11396
- Return an error if marshaling invalid Uint256 #11347
- Fallback to uncached getPayload if timeout #11404
- Update bazel to 5.3.0 #11427
- godocs cleanup and other cleanups #11428
- Forkchoice track highest received root #11434
- Metrics updated block arrival time histograms #11424
- Log error and continue when proposer boost roots are missing #11459
- Do not return on error during on_tick #11463
- Do not return on error after update head #11470
- Update default RPC HTTP timeout to 30s #11487
- Improved fee recipient UX. #11307
- Produce block skips mev-boost #11488
- Builder getPayload timeout set to 3s #11413
- Make stategen aware of forkchoice #11439
- Increase verbosity of warning to error when new head cannot be determined when receiving an attestation #11514
- Provide justified balances to forkchoice #11513
- Update head continues without attestations #11503
- Migrate historical states in another goroutine to avoid blocking block execution #11501
- Made API middleware structs public #11547
- Updated web UI to v2.0.2 #11559
- Default value for --block-batch-limit-burst-factor changed from 10 to 2. #11546
- Vendored leaky bucket implementation with minor modifications #11560

### Deprecated

- --disable-native-state flag and associated feature #11268

### Removed

- Unused WithTimeout for builder client #11420
- Optimistic sync candidate check #11453
- Cleans up proto states #11445 #11561
- Protoarray implementation of forkchoice #11455

### Fixed

- Block fields to return a fixed sized array rather than slice #11375
- Lost cancel in validator runner #11429
- Release held lock on error #11412
- Properly submit blinded blocks #11483
- Unwanted wrapper of gRPC status errors #11486
- Sync tests fixed and updated spectests to 1.2.0 #11498 #11492
- Prevent timeTillDuty from reporting a negative value #11512
- Don't mark /healthz as unhealthy when mev-boost relayer is down #11506
- Proposer index cache and slot is used for GetProposerDuties #11521
- Properly retrieve values for validator monitoring flag from cli #11537
- Fee recipient fixes and persistence #11540
- Handle panic when rpc client is not yet initialized #11528
- Improved comments and error messages #11550
- SSL support for multiple gRPC endpoints #11556
- Addressed some tool feedback and code complaints #11555
- Handle unaggregated attestations in the event feed #11558
- Prune / expire payload ID cache entries when using beacon json API #11568
- Payload ID cache may have missed on skip slots due to incorrect key computation #11567

### Security

- Libp2p updated to v0.22.0 #11309

## [v3.1.1](https://github.com/prysmaticlabs/prysm/compare/v3.1.0...v3.1.1) - 2022-09-09

This is another highly recommended release. It contains a forkchoice pruning fix and a gossipsub optimization. It is recommended to upgrade to this release before the Merge next week, which is currently tracking for Wed Sept 14 (https://bordel.wtf/). Happy staking! See you on the other side!

### Fixed

- Fix memory leaks in fork choice store which leads to node becoming slower #11387
- Improve connectivity and solves issues connecting with peers #11425

### Security

No security updates in this release.

## [v3.1.0](https://github.com/prysmaticlabs/prysm/compare/v3.1.0...v3.0.0) - 2022-09-05

Updating to this release is highly recommended as it contains several important fixes and features for the merge. You must be using Prysm v3 or later before Bellatrix activates on September 6th. 

**Important docs links**

- [How to prepare for the merge](https://docs.prylabs.network/docs/prepare-for-merge)
- [How to check merge readiness status](https://docs.prylabs.network/docs/monitoring/checking-status)


### Added

- Add time until next duty in epoch logs for validator #11301
- Builder API: Added support for deleting gas limit endpoint #11290
- Added debug endpoint GetForkChoice for doubly-linked-tree #11312 #11325
- Added support for engine API headers. --execution-headers=key=value #11330
- New merge specific metrics. See #11367

### Changed

- Deposit cache now returns shallow copy of deposits #11273
- Updated go-ethereum dependency to v1.10.23
- Updated LLVM compiler version to 13.0.1
- Builder API: filter 0 bid and empty tx root responses #11313
- Allow attestations/blocks to be received by beacon node when the nodes only optimistically synced #11319 #11320
- Add depth and distance to CommonAncestorRoot reorg object #11315
- Allocate slice array to expected length in several methods #11317
- Updated lighthouse to version v3 in E2E runner
- Improved handling of execution client errors #11321 #11359
- Updated web3signer version in E2E runner #11339
- Improved error messages for db unmarshalling failures in ancestor state lookup #11342
- Only updated finalized checkpoints in database if its more recent than previous checkpoint #11356

### Removed

- Dead / unused code delete #11326 #11345

### Fixed

- Fixed improper wrapping of certain errors #11282
- Only log fee recipient message if changed #11295 #11296
- Simplify ListAttestations RPC method #11292 fixes #11291
- Fix several RPC methods to be aware of the appropriate fork #11274
- Fixed encoding issue with builder API register validator method. #11299 fixes #11297
- Improved blinded block handling in API. #11304 fixes #11293
- Fixed IPC path for windows users #11324
- Fix proposal of blinded blocks #11346
- Prysm no longer crashes on start up if builder endpoint is not available #11380

### Security

There are no security updates in this release.

## [v3.0.0](https://github.com/prysmaticlabs/prysm/compare/v3.0.0...v2.1.4) 2022-08-22

### Added

- Passing spectests v1.2.0-rc.3 #11261
- prysmctl: Generate genesis state via prysmctl testnet generate-genesis [command options] [arguments...] #11259
- Keymanager: Add support for setting the gas limit via API. #11155
- Merge: Mainnet merge epoch and TTD defined! #11207
- Validator: Added expected wait time for pending validator activation in log message. #11213
- Go: Prysm now uses proper versioning suffix v3 for this release. GoDocs and downstream users can now import prysm as expected for go projects. #11083
- Builder API: Register validator via HTTP REST Beacon API endpoint /eth/v1/validator/register_validator #11225
- Cross compilation support for Mac ARM64 chips (Mac M1, M2) #10981

### Changed

- **Require an execution client** `--execution-endpoint=...`. The default value has changed to `localhost:8551` and you must use the jwt flag `--jwt-secret=...`. Review [the docs](https://docs.prylabs.network/docs/prepare-for-merge) for more information  #10921
- `--http-web3provider` has been renamed to `--execution-endpoint`. Please update your configuration as `--http-web3provider` will be removed in a future release. #11275 #11133
- Insert attestations into forkchoice sooner #11260
- Builder API: `gas_limit` changed from int to string to support JSON / YAML configs. `--suggested-gas-limit` changed from int to string. #11264
- Fork choice: Improved handling of double locks / deadlocks #11271 #11269
- Lower libp2p log level #11266
- Improved re-org logs with additional metadata #11253
- Improved error messages found by semgrep #11244
- Prysm Web UI updated to release v2.0.1 #11240
- Protobuf message renaming (non-breaking changes) #11096
- Enabled feature to use gohashtree by default. Disable with `--disable-vectorized-htr` #11229 #11224
- Enabled fork choice doubly linked tree feature by default. Disable with `--disable-forkchoice-doubly-linked-tree` #11212
- Remote signer: Renamed some field names to better represent block types (non-breaking changes for gRPC users, possibly breaking change for JSON API users) #11099
- Builder API: require header and payload root match. #11223
- Improved responses for json-rpc requests batching when using blinded beacon blocks. #11210
- Builder API: Improved error messages #11199
- Builder API: Issue warning when validator expects builder ready beacon node, but beacon node is not configured with a relay. #10203
- Execution API: Improved payload ID to handle reorg scenarios #11186

### Deprecated

- Several features have been promoted to stable or removed. The following flags are now deprecated and will be removed in a future release. `--enable-db-backup-webhook`, `--bolt-mmap-initial-size`, `--disable-discv5`, `--disable-attesting-history-db-cache`, `--enable-vectorized-htr`, `--enable-peer-scorer`, `--enable-forkchoice-doubly-linked-tree`, `--enable-duty-count-down`, `--head-sync`, `--enable-gossip-batch-aggregateion`, `--enable-larger-gossip-history`, `--fallback-web3provider`, `--use-check-point-cache`. #11284 #11281 #11276 #11231 #10921 #11121
- Several beacon API endpoints marked as deprecated #10946

### Removed

- Logging: Removed phase0 fields from validator performance log messages #11265
- Deprecated slasher protos have been removed #11257
- Deprecated beacon API endpoints removed: `GetBeaconState`, `ProduceBlock`, `ListForkChoiceHeads`, `ListBlocks`, `SubmitValidatorRegistration`, `GetBlock`, `ProposeBlock` #11251 #11243 #11242 #11106
- API: Forkchoice method `GetForkChoice` has been removed. #11105
- All previously deprecated feature flags have been removed. `--enable-active-balance-cache`, `--correctly-prune-canonical-atts`, `--correctly-insert-orphaned-atts`, `--enable-next-slot-state-cache`, `--enable-batch-gossip-verification`, `--enable-get-block-optimizations`, `--enable-balance-trie-computation`, `--disable-next-slot-state-cache`, `--attestation-aggregation-strategy`, `--attestation-aggregation-force-opt-maxcover`, `--pyrmont`, `--disable-get-block-optimizations`, `--disable-proposer-atts-selection-using-max-cover`, `--disable-optimized-balance-update`, `--disable-active-balance-cache`, `--disable-balance-trie-computation`, `--disable-batch-gossip-verification`, `--disable-correctly-prune-canonical-atts`, `--disable-correctly-insert-orphaned-atts`, `--enable-native-state`, `--enable-peer-scorer`, `--enable-gossip-batch-aggregation`, `--experimental-disable-boundry-checks` #11125
- Validator Web API: Removed unused ImportAccounts and DeleteAccounts rpc options #11086

### Fixed

- Keymanager API: Status enum values are now returned as lowercase strings. #11194
- Misc builder API fixes #11228
- API: Fix GetBlock to return canonical block #11221
- Cache: Fix cache overwrite policy for bellatrix proposer payload ID cache. #11191
- Fixed string slice flags with file based configuration #11166

### Security

- Upgrade your Prysm beacon node and validator before the merge!

## [v2.1.4](https://github.com/prysmaticlabs/prysm/compare/v2.1.4...v2.1.3) - 2022-08-10

As we prepare our `v3` mainnet release for [The Merge](https://ethereum.org/en/upgrades/merge/), `v2.1.4` marks the end of the `v2` era. Node operators and validators are **highly encouraged** to upgrade to release `v2.1.4` - many bug fixes and improvements have been included in preparation for The Merge. `v3` will contain breaking changes, and will be released within the next few weeks. Using `v2.1.4` in the meantime will give you access to a more streamlined user experience. See our [v2.1.4 doc](https://docs.prylabs.network/docs/vnext/214-rc) to learn how to use v2.1.4 to run a Merge-ready configuration on the Goerli-Prater network pair.

### Added

- Sepolia testnet configs `--sepolia` #10940 #10962 #11131
- Goerli as an alias to Prater and testnet configs `--prater` or `--goerli` #11065 #11072 #11152 
- Fee recipient API for key manager #10850
- YML config flag support for web3 signer #11041
- Validator registration API for web3 signer #10964 #11055 
- JSON tcontent type with optional metadata #11058
- Flashbots MEV boost support #10894 #10907 #10924 #10908 #10942 #10953 #10954 #10992 #11004 #11021 #11124 #11176 
- Store blind block (i.e block with payload header) instead of full block (i.e. block with payload) for storage efficiency (currently only available when the `enable-only-blinded-beacon-blocks` feature flag is enabled) #11010 
- Pcli utility support to print blinded block #11067 
- New Web v2.0 release into Prysm #11007

### Changed

- Native state improvement is enabled by default #10898 
- Use native blocks instead of protobuf blocks #10885 #11158 
- Peer scorer is enabled by default #11115
- Enable fastssz to use vectorized HTR hash algorithm improvement #10819 
- Forkchoice store refactor and cleanups #10893 #10905 #10903 #10953 #10977 #10955 #10840 #10980 #10979
- Update libp2p library dependency #10958 
- RPC proposer duty is now allowed next epoch query #11015 
- Do not print traces with `log.withError(err)` #11116
- Testnets are running with pre-defined feature flags #11098

### Removed

- Deprecate Step Parameter from our Block By Range Requests #10914 

### Fixed

- Ignore nil forkchoice node when saving orphaned atts #10930 
- Sync: better handling of missing state summary in DB #11167 
- Validator: creates invalid terminal block using the same timestamp as payload #11129 
- P2P: uses incorrect goodbye codes #11168 
- P2p: defaults Incorrectly to using Mplex, which results in losing Teku peers #11169 
- Disable returning future state for API #10915
- Eth1 connection API panic #10938

### Security

There are no security updates in this release.

## [v2.1.3](https://github.com/prysmaticlabs/prysm/compare/v2.1.2...v2.1.3) - 2022-07-06

### Added

- Many fuzz test additions #10682 #10668 #10757 #10798
- Support bellatrix blocks with web3signer #10590
- Support for the Sepolia testnet with `--terminal-total-difficulty-override 17000000000000000`. The override flag is required in this release. #10700 #10868 #10880 #10886
- Support for the Ropsten testnet. No override flag required #10762 #10817
- JSON API allows SSZ-serialized blocks in `publishBlock` #10663
- JSON API allows SSZ-serialized blocks in `publishBlindedBlock` #10679
- JSON API allows SSZ-serialized requests in `produceBlockV2` and `produceBlindedBlock` #10697
- Progress towards Builder API and MEV boost support (not ready for testing in this release) #10724 #10785 #10789 #10749 #10825 #10882 #10883
- Support for `DOMAIN_APPLICATION_MARK` configuration #10740
- Ignore subset aggregates if a better aggregate has been seen already #10674 
- Reinsertion of reorg'd attestations #10767
- Command `beacon-chain generate-auth-secret` to assist with generating a hex encoded secret for engine API #10733
- Return optimistic status to `ChainHead` related grpc service #10842
- TTD log and prometheus metric #10851
- Panda ascii art banner for the merge! #10773

### Changed

- Improvements to forkchoice #10675 #10651 #10705 #10658 #10702 #10659 #10768 #10776 #10783 #10801 #10774 #10784 #10831 #10823
- Invalid checksummed (or no checksum) addresses used for fee recipient will log a warning. #10664 fixes #10631, #10684 
- Use cache backed `getBlock` method in several places of blockchain package #10688
- Reduced log frequency of "beacon node doesn't have a parent in db with root" error #10689
- Improved nil checks for state management #10701
- Enhanced debug logs for p2p block validation #10698
- Many helpful refactoring and cosmetic changes #10686 #10710 #10706 #10726 #10729 #10707 #10731 #10736 #10732 #10727 #10756 #10816 #10824 #10841 #10874 #10704 #10862
- Move WARN level message about weak subjectivity sync and improve message content #10699
- Handle connection closing for web3/eth1 nil connection #10714
- Testing improvements #10711 #10728 #10665 #10753 #10756 #10775
- E2E test improvements #10717 #10715 #10708 #10696 #10751 #10758 #10769 #10778 #10808 #10849 #10836 #10878
- Increase file descriptor limit up to the maximum by default #10650 
- Improved classification of "bad blocks" #10681
- Updated engine API error code handling #10730
- Improved "Synced new block" message to include minimal information based on the log verbosity. #10724 #10792
- Add nil checks for nil finalized checkpoints #10748 #10881
- Change weak subjectivity sync to use the most recent finalized state rather than the oldest state within the current period. #10723
- Ensure a finalized root can't be all zeros #10791
- Improved db lookup of HighestSlotBlocksBelow to start from the end of the index rather than the beginning. #10772 #10802 
- Improved packing of state balances for hashtreeroot #10830
- Improved field trie recomputation #10884

### Removed

-  Removed handling of `INVALID_TERMINAL_BLOCK` response from engine API #10646 

### Fixed

- `/eth/v1/beacon/blinded_blocks` JSON API endpoint #10673
- SSZ handling of JSON API payloads #10687 #10760
- Config registry fixes #10694 fixed by #10683 
- Withdrawal epoch overflows #10739
- Race condition with blockchain service Head() #10741
- Race condition with validator's highest valid slot accessor #10722
- Do not update cache with the result of a cancelled request #10786
- `validator_index` should be a string integer rather than a number integer per spec. #10814
- Use timestamp heuristic to determine deposits to process rather than simple calculation of follow distance #10806
- Return `IsOptimistic` in `ValidateSync` responses #10692

### Security

There are no security updates in this release.

## [v2.1.2](https://github.com/prysmaticlabs/prysm/compare/v2.1.1...v2.1.2) - 2022-05-16

### Added

- Update forkchoice head before produce block #10653
- Support for blst modern builds on linux amd64 #10567
- [Beacon API support](ethereum/beacon-APIs#194) for blinded block #10331
- Proposer index and graffiti fields in Received block debug log for verbosity #10564
- Forkchoice removes equivocating votes for weight accounting #10597

### Changed

- Updated to Go [1.18](https://github.com/golang/go/releases/tag/go1.18) #10577
- Updated go-libp2p to [v0.18.0](https://github.com/libp2p/go-libp2p/releases/tag/v0.18.0) #10423
- Updated beacon API's Postman collection to 2.2.0 #10559
- Moved eth2-types into Prysm for cleaner consolidation of consensus types #10534

### Removed

- Prymont testnet support #10522
- Flag `disable-proposer-atts-selection-using-max-cover` which disables defaulting max cover strategy for proposer selecting attestations #10547
- Flag `disable-get-block-optimizations` which disables optimization with beacon block construction #10548
- Flag `disable-optimized-balance-update"` which disables optimized effective balance update #10549
- Flag `disable-active-balance-cache` which disables active balance cache #10550
- Flag `disable-balance-trie-computation` which disables balance trie optimization for hash tree root #10552 
- Flag `disable-batch-gossip-verification` which disables batch gossip verification #10553
- Flag `disable-correctly-insert-orphaned-atts` which disables the fix for orphaned attestations insertion #10622

### Fixed

- `end block roots don't match` bug which caused beacon node down time #10680
- Doppelganger off by 1 bug which introduced some false-positive #10582
- Fee recipient warning log is only disabled after Bellatrix fork epoch #10543

### Security

There are no security updates in this release.

## [v2.1.1](https://github.com/prysmaticlabs/prysm/compare/v2.1.0...v2.1.1) - 2022-05-03

This patch release includes 3 cherry picked fixes for regressions found in v2.1.0.

View the full changelist from v2.1.0: https://github.com/prysmaticlabs/prysm/compare/v2.1.0...v2.1.1

If upgrading from v2.0.6, please review the [full changelist](https://github.com/prysmaticlabs/prysm/compare/v2.0.6...v2.1.1) of both v2.1.0 and v2.1.1.

This release is required for users on v2.1.0 and recommended for anyone on v2.0.6.

The following known issues exist in v2.1.0 and also exist in this release.
    
- Erroneous warning message in validator client when bellatrix fee recipient is unset. This is a cosmetic message and does not affect run time behavior in Phase0/Altair. Fixed by #10543
- In Bellatrix/Kiln: Fee recipient flags may not work as expected. See #10555 for a fix and more details.

### Fixed

- Doppelganger false positives may have caused a failure to start in the validator client. Fixed by #10582
- Connections to execution layer clients were not properly cleaned up and lead to resource leaks when using ipc. #10573 fixed by #10574
- Initial sync (or resync when beacon node falls out of sync) could lead to a panic. #10570 fixed by #10568

### Security

There are no security updates in this release.

## [v2.1.0](https://github.com/prysmaticlabs/prysm/compare/v2.0.6...v2.1.0) - 2022-04-26 

There are two known issues with this release:

- Erroneous warning message in validator client when bellatrix fee recipient is unset. This is a cosmetic message and does not affect run time behavior in Phase0/Altair. Fixed by #10543
- In Bellatrix/Kiln: Fee recipient flags may not work as expected. See #10555 for a fix and more details.

### Added

- Web3Signer support. See the [documentation](https://docs.prylabs.network/docs/next/wallet/web3signer) for more details.
- Bellatrix support. See [kiln testnet instructions](https://hackmd.io/OqIoTiQvS9KOIataIFksBQ?view)
- Weak subjectivity sync / checkpoint sync. This is an experimental feature and may have unintended side effects for certain operators serving historical data. See the [documentation](https://docs.prylabs.network/docs/next/prysm-usage/checkpoint-sync) for more details.
- A faster build of blst for beacon chain on linux amd64. Use the environment variable `USE_PRYSM_MODERN=true` with prysm.sh, use the "modern" binary, or bazel build with `--define=blst_modern=true`. #10229
- Vectorized sha256. This may have performance improvements with use of the new flag `--enable-vectorized-htr`. #10166
- A new forkchoice structure that uses a doubly linked tree implementation. Try this feature with the flag `--enable-forkchoice-doubly-linked-tree` #10299
- Fork choice proposer boost is implemented and enabled by default. See PR #10083 description for more details.

### Changed

- **Flag Default Change** The default value for `--http-web3provider` is now `localhost:8545`. Previously was empty string. #10498
- Updated spectest compliance to v1.1.10. #10298
- Updated to bazel 5.0.0 #10352
- Gossip peer scorer is now part of the `--dev` flag. #10207

### Removed

- Removed released feature for next slot cache. `--disable-next-slot-state-cache` flag has been deprecated and removed. #10110

### Fixed

Too many bug fixes and improvements to mention all of them. See the [full changelist](https://github.com/prysmaticlabs/prysm/compare/v2.0.6...v2.1.0)

### Security

There are no security updates in this release.

## [v2.0.6](https://github.com/prysmaticlabs/prysm/compare/v2.0.5...v2.0.6) 2022-01-31

### Added

- Bellatrix/Merge progress #10000 #9981 #10026 #10027 #10049 #10039 #10028 #10030 #10054 #10060 #10062 #10014 #10040 #10085 #10077 #10089 #10072
- Light client support merkle proof retrieval for beacon state finalized root and sync committees #10029
- Web3Signer support (work in progress) #10016 #10061 #10084 #10088
- Implement state management with native go structs (work in progress) #10069 #10079 #10080 #10089
- Added static analysis for mutex lock management #10066
- Add endpoint to query eth1 connections #10073 #10103
- Batch gossipsub verification enabled #10111
- Get block optimizations enabled #10106
- Batch decompression for signatures #10105
- Balance trie feature enabled #10112

### Changed

- Use build time constants for field lengths. #10007 #10012 #10019 #10042
- Monitoring service logging improvements / cleanup #10013
- Renamed state v3 import alias #10022
- Spec tests passing at tag 1.1.8 #10033 #10071
- Bazel version updated to 4.2.2
- Renamed github.com/eth2-clients -> github.com/eth-clients #10057
- p2p reduce memory allocation in gossip digest calculation #10055
- Allow comma separated formatting for event topics in API requests #10052
- Update builder image from buster to bullseye #10025
- Renaming "merge" to "bellatrix" #10044
- Refactoring / code dedupication / general clean up #10081 #10090 #10074 #10093 #10101 #10065 #10104
- Update libp2p #10082
- Reduce state copy in state upgrades #10102
- Deduplicate sync committee messages from pool before retrieval #10106

### Removed

- tools/deployContract: removed k8s specific logic #10075

### Fixed

- Sync committee API endpoint can now be queried for future epochs #10015
- Initialize merkle layers and recompute dirty fields in beacon state proofs #10032
- Fixed data race in API calls #10050

### Security

- Clean variable filepaths in validator wallet back up commands, e2e tests, and other tooling (gosec G304) #10115

## [v2.0.5](https://github.com/prysmaticlabs/prysm/compare/v2.0.4...v2.0.5) - 2021-12-13

### Added

- Implement import keystores standard API #9924
- Added more fields to "Processed attestation aggregation" log #9937
- Incremental changes to support The Merge hardfork #9906 #9939 #9944 #9966 #9878 #9986 #9987 #9982
- Implement validator monitoring service in beacon chain node via flag `--monitor-indices`. #9933
- Added validator log to display "aggregated since launch" every 5 epochs. #9943
- Add HTTP client wrapper for interfacing with remote signer #9991 See #9994
- Update web UI to version v1.0.2 #10009.

### Changed

- Refactor beacon state to allow for a single cached hasher #9922
- Default config name to "devnet" when not provided in the config yaml. #9949
- Alter erroneously capitalized error messages #9952
- Bump spec tests to version v1.1.6 #9955
- Improvements to Doppelganger check #9964
- Improvements to "grpc client connected" log. #9956
- Update libp2p to v0.15.1 #9960
- Resolve several checks from deepsource #9961
- Update go-ethereum to v1.10.13 #9967
- Update some flags from signed integer flags to unsigned flags. #9959
- Filter errored keys from slashing protection history in standard API. #9968
- Ensure slashing protection exports and key manager api work according to spec #9938
- Improve memory performance by properly allocating slice size #9977
- Typos fix #9980 #9979
- Remove unused imports #9983
- Use cashed finalized state when pruning deposits #9985
- Significant slasher improvements #9833 #9989
- Various code cleanups #9992
- Standard API improvements for keymanager API #9936 #9995
- Use safe sub64 for safer math #9993
- Fix CORS in middleware API #9999
- Add more fields to remote signer request object #10004
- Refactoring to support checkpoint or genesis origin. #9976

### Deprecated

Please be advised that Prysm's package path naming will change in the next release. If you are a downstream user of Prysm (i.e. import prysm libraries into your project) then you may be impacted. Please see issue https://github.com/prysmaticlabs/prysm/issues/10006.

### Fixed

- Allow API requests for next sync committee. Issue #9940 fixed by #9945
- Check sync status before performing a voluntary exit. Issue #9950 fixed by #9951
- Fixed issue where historical requests for validator balances would time out by removing the 30s timeout limitation. Issue #9973 fixed by #9957.
- Add missing ssz spec tests #10003

### Security

- Add justifications to gosec security finding suppression. #10005

## [v2.0.4](https://github.com/prysmaticlabs/prysm/compare/v2.0.3...v2.0.4) - 2021-11-29

### Added

- Several changes for The Merge #9915 #9916 #9918 #9928 #9929
- More monitoring functionality for blocks and sync committees #9923 #9910

### Changed

- Improvements to block proposal computation when packing deposits. #9806
- Renaming SignatureSet -> SignatureBatch #9926

### Deprecated

### Fixed

- Revert PR [9830](https://github.com/prysmaticlabs/prysm/pull/9830) to remove performance regression. See: issue [9935](https://github.com/prysmaticlabs/prysm/issues/9935)

### Security

No security updates in this release.

## [v2.0.3](https://github.com/prysmaticlabs/prysm/compare/v2.0.2...v2.0.3) - 2021-11-22

This release also includes a major update to the web UI. Please review the v1 web UI notes [here](https://github.com/prysmaticlabs/prysm-web-ui/releases/tag/v1.0.0)

### Added

- Web v1 released #9858
- Updated Beacon API to v2.1.0 #9797
- Add validation of keystores via validator client RPC endpoint to support new web UI #9799
- GitHub actions: errcheck and gosimple lint #9729
- Event API support for `contribution_and_proof` and `voluntar_exit` events. #9779
- Validator key management standard API schema and some implementation #9817 #9886 #9863
- Add helpers for The Merge fork epoch calculation #9879
- Add cli overrides for certain constants for The Merge #9891
- Add beacon block and state structs for The Merge #9887 #9888 #9908 #9914
- Validator monitoring improvements #9898 #9899 #9901 #9921
- Cache deposits to improve deposit selection/processing #9885
- Emit warning upon empty validator slashing protection export #9909 #9919
- Add balance field trie cache and optimized hash trie root operations. `--enable-balance-trie-computation` #9793

### Changed

- Updated to spectests v1.1.5 #9875
- Refactor web authentication #9740
- Added uint64 overflow protection #9807
- Sync committee pool returns empty slice instead of nil on cache miss #9808
- Improved description of datadir flag #9809
- Simplied web password requirements #9814
- Web JWT tokens no longer expire. #9813
- Updated keymanager protos #9827
- Watch and update jwt secret when auth token file updated on disk. #9810
- Update web based slashing protection export from POST to GET #9838
- Reuse helpers to validate fully populated objects. #9834
- Rename interop-cold-start to deterministic-genesis #9841
- Validate password on RPC create wallet request #9848
- Refactor for weak subjectivity sync implementation #9832
- Update naming for Atlair previous epoch attester #9840
- Remove duplicate MerkleizeTrieLeaves method. #9847
- Add explict error for validator flag checks on out of bound positions #9784
- Simplify method to check if the beacon chain client should update the justified epoch value. #9837
- Rename web UI performance endpoint to "summary" #9855
- Refactor powchain service to be more functional #9856
- Use math.MaxUint64 #9857
- Share / reused finalized state on prysm start up services #9843
- Refactor slashing protection history code packages #9873
- Improve RNG commentary #9892
- Use next slot cache in more areas of the application #9884
- Improve context aware p2p peer scoring loops #9893
- Various code clean up #9903
- Prevent redundant processing of blocks from pending queue #9904
- Enable Altair tests on e2e against prior release client #9920
- Use lazy state balance cache #9822

### Deprecated

- Web UI login has been replaced. #9858
- Web UI bar graph removed. #9858

### Removed

- Prysmatic Labs' [go-ethereum fork](https://github.com/prysmaticlabs/bazel-go-ethereum) removed from build tooling. Upstream go-ethereum is now used with familiar go.mod tooling. #9725
- Removed duplicate aggergation validation p2p pipelines. #9830
- Metrics calculation removed extra condition #9836
- Removed superflous errors from peer scoring parameters registration #9894

### Fixed

- Allow submitting sync committee subscriptions for next period #9798
- Ignore validators without committee assignment when fetching attester duties #9780
- Return "version" field for ssz blocks in beacon API #9801
- Fixed bazel build transitions for dbg builds. Allows IDEs to hook into debugger again. #9804
- Fixed case where GetDuties RPC endpoint might return a false positive for sync committee selection for validators that have no deposited yet #9811
- Fixed validator exits in v1 method, broadcast correct object #9819
- Fix Altair individual votes endpoint #9825 #9829 #9831
- Validator performance calculations fixed #9828
- Return correct response from key management api service #9846
- Check empty genesis validators root on slashing protection data export #9849
- Fix stategen with genesis state. #9851 #9852 #9866
- Fixed multiple typos #9868
- Fix genesis state registration in interop mode #9900
- Fix network flags in slashing protection export #9905 #9907  

### Security

- Added another encryption key to security.txt. #9896

## [v2.0.2](https://github.com/prysmaticlabs/prysm/compare/v2.0.1...v2.0.2) - 2021-10-18

### Added

- Optimizations to block proposals. Enabled with `--enable-get-block-optimizations`. See [issue 8943](https://github.com/prysmaticlabs/prysm/issues/8943) and [issue 9708](https://github.com/prysmaticlabs/prysm/issues/9708) before enabling.
- Beacon Standard API: register v1alpha2 endpoints #9768

### Changed

- Beacon Standard API: Improved sync error messages #9750
- Beacon Standard API: Omit validators without sync duties #9756
- Beacon Standard API: Return errors for unknown state/block versions #9781
- Spec alignment: Passing spec vectors at v1.1.2 #9755
- Logs: Improved "synced block.." #9760
- Bazel: updated to v4.2.1 #9763
- E2E: more strict participation checks #9718
- Eth1data: Reduce disk i/o saving interval #9764

### Deprecated

- ⚠️ v2 Remote slashing protection server disabled for now ⚠️ #9774

### Fixed

- Beacon Standard API: fetch sync committee duties for current and next period's epoch #9720 #9728
- Beacon Standard API: remove special treatment to graffiti in block results #9770
- Beacon Standard API: fix epoch calculation in sync committee duties #9767
- Doppelganger: Fix false positives #9748
- UI: Validator gRPC gateway health endpoint fixed #9747

### Security

- Spec alignment: Update Eth2FastAggregateVerify to match spec #9742
- Helpers: enforce stronger slice index checks #9758
- Deposit Trie: Handle impossible non-power of 2 trie leaves #9761
- UI: Add security headers #9775

## [v2.0.1](https://github.com/prysmaticlabs/prysm/compare/v2.0.0...v2.0.1) - 2021-10-06

### Fixed

- Updated libp2p transport library to stop metrics logging errors on windows. #9733
- Prysm's web UI assets serve properly #9732
- Eth2 api returns full validator balance rather than effective balance #9722
- Slashing protection service registered properly in validator. #9735

### Security

We've updated the Prysm base docker images to a more recent build. #9727 #9674

## [v2.0.0](https://github.com/prysmaticlabs/prysm/compare/v1.4.4...v2.0.0)

This release is the largest release of Prysm to date. v2.0.0 includes support for the upcoming Altair hard fork on the mainnet Ethereum Beacon Chain.
This release consists of [380 changes](https://github.com/prysmaticlabs/prysm/compare/v1.4.4...f7845afa575963302116e673d400d2ab421252ac) to support Altair, improve performance of phase0 beacon nodes, and various bug fixes from v1.4.4.

### Upgrading From v1

Please update your beacon node to v2.0.0 prior to updating your validator. The beacon node can serve requests to a v1.4.4 validator, however a v2.0.0 validator will not start against a v1.4.4 beacon node. If you're operating a highly available beacon chain service, ensure that all of your beacon nodes are updated to v2.0.0 before starting the upgrade on your validators.

### Added

- Full Altair support. [Learn more about Altair.](https://github.com/ethereum/annotated-spec/blob/8473024d745a3a2b8a84535d57773a8e86b66c9a/altair/beacon-chain.md)
- Added bootnodes from the Nimbus team. #9656
- Revamped slasher implementation. The slasher functionality is no longer a standalone binary. Slasher functionality is available from the beacon node with the `--slasher` flag. Note: Running the slasher has considerably increased resource requirements. Be sure to review the latest documentation before enabling this feature. This feature is experimental. #8331
- Support for standard JSON API in the beacon node. Prysm validators continue to use Prysm's API. #7510
- Configurable subnet peer requirements. Increased minimum desired peers per subnet from 4 to 6. This can be modified with `--minimum-peers-per-subnet` in the beacon node. #9657.
- Support for go build on darwin_arm64 devices (Mac M1 chips). Cross compiling for darwin_arm64 is not yet supported. #9600.
- Batch verification of pubsub objects. This should improve pubsub processing performance on multithreaded machines. #9344
- Improved attestation pruning. This feature should improve block proposer performance and overall network attestation inclusion rates. Opt-out with `--disable-correctly-prune-canonical-atts` in the beacon node. #9444
- Active balance cache to improve epoch processing. Opt-out with `--disable-active-balance-cache` #9567
- Experimental database improvements to reduce history state entry space usage in the beaconchain.db. This functionality can be permanently enabled with the flag `--enable-historical-state-representation`. Enabling this feature can realize a 25% improvement in space utilization for the average user , while 70 -80% for power users(archival node operators). Note: once this feature is toggled on, it modifies the structure of the database with a migration and cannot be rolled back. This feature is experimental and should only be used in non-serving beacon nodes in case of database corruption or other critical issue. #8954

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
| `p2p_sync_committee_subnet_attempted_broadcasts` | The number of sync committees that were attempted to be broadcast                                     | #9390       |
| `p2p_subscribed_topic_peer_total`                | The number of peers subscribed to topics that a host node is also subscribed to                       | #9538       |
| `saved_orphaned_att_total`                       | Count the number of times an orphaned attestation is saved                                            | #9442       |

### Changed

- Much refactoring of "util" packages into more canonical packages. Please review Prysm package structure and godocs.
- Altair object keys in beacon-chain/db/kv are prefixed with "altair". BeaconBlocks and BeaconStates are the only objects affected by database key changes for Altair. This affects any third party tooling directly querying Prysm's beaconchain.db.
- Updated Teku bootnodes. #9656
- Updated Lighthouse bootnodes. #9656
- End to end testing now collects jaeger spans #9341
- Improvements to experimental peer quality scoring. This feature is only enabled with `--enable-peer-scorer`. #8794
- Validator performance logging behavior has changed in Altair. Post-Altair hardfork has the following changes: Inclusion distance and inclusion slots will no longer be displayed. Correctly voted target will only be true if also included within 32 slots. Correctly voted head will only be true if the attestation was included in the next slot. Correctly voted source will only be true if attestation is included within 5 slots. Inactivity score will be displayed. #9589
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
- Kafka support is no longer available in the beacon node. This functionality was never fully completed and did not fulfill many desirable use cases. This removed the flag `--kafka-url` (beacon node). See #9470.
- Removed tools/faucet. Use the faucet in [prysmaticlabs/periphery](https://github.com/prysmaticlabs/periphery/tree/c2ac600882c37fc0f2a81b0508039124fb6bcf47/eth-faucet) if operating a testnet faucet server.
- Tooling for prior testnet contracts has been removed. Any of the old testnet contracts with `drain()` function have been removed as well. #9637
- Toledo tesnet config is removed.
- Removed --eth-api-port (beacon node). All APIs interactions have been moved to --grpc-gateway-port. See #9640.

### Fixed

- Database lock contention improved in block database operations. #9428
- JSON API now returns an error when unknown fields are provided. #9710
- Correctly return `epoch_transition` field in `head` JSON API events stream. #9668 #9704
- Various fixes in standard JSON API #9649
- Finalize deposits before initializing the beacon node. This may improve missed proposals #9639 #9610
- JSON API returns header "Content-Length" 0 when returning an empty JSON object. #9531 #9540
- Initial sync fixed when there is a very long period of missing blocks. #9450 #9452
- Fixed log statement when a web3 endpoint failover occurs. #9272
- Windows prysm.bat is fixed #9266 #9260

### Security

- You MUST update to v2.0.0 or later release before epoch 74240 or your client will fork off from the rest of the network.
- Prysm's JWT library has been updated to a maintained version of the previous JWT library. JWTs are only used in the UI. #9357

Please review our newly updated [security reporting policy](https://github.com/prysmaticlabs/prysm/blob/develop/SECURITY.md).
- Fix subcommands such as validator accounts list #9236

### Security

There are no security updates in this release.

# Older than v2.0.0

For changelog history for releases older than v2.0.0, please refer to https://github.com/prysmaticlabs/prysm/releases
