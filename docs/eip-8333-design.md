# EIP-8333: Align Checkpoint with Epoch Boundary Block. Prysm design doc

Status: implemented behind `HEZE_FORK_EPOCH` (defaults to `FAR_FUTURE_EPOCH`, matching upstream configs, so mainnet and all testnets are unaffected until Heze is scheduled). The EIP text calls the activation parameter `EIP8333_FORK_EPOCH`; upstream config files and Lodestar assign it to the Heze fork, and Prysm follows that naming.

EIP: <https://eips.ethereum.org/EIPS/eip-8333>

## 1. What the EIP changes

Today the FFG checkpoint for epoch `N` resolves to the most recent block at or before the *first slot of epoch `N`* (`get_block_root`). EIP-8333 anchors it instead to the *epoch boundary block*: the most recent block at or before the last slot of epoch `N - 1`. Checkpoint epochs, justification bits, finalization rules, slashing conditions and all containers are unchanged; only the block root a checkpoint maps to moves.

The spec delta is:

- New `get_checkpoint_slot(epoch)`:
  - `GENESIS_EPOCH` resolves to `GENESIS_SLOT`,
  - epochs before `EIP8333_FORK_EPOCH` resolve to `compute_start_slot_at_epoch(epoch)` (the previous anchoring),
  - epochs at or after the activation epoch resolve to `compute_start_slot_at_epoch(epoch) - 1`.
- New `get_checkpoint_root(state, epoch) = get_block_root_at_slot(state, get_checkpoint_slot(epoch))`.
- `get_attestation_participation_flag_indices`: the matching-target check compares `data.target.root` against `get_checkpoint_root(state, data.target.epoch)`.
- `weigh_justification_and_finalization`: newly recorded justified checkpoints take their root from `get_checkpoint_root`.
- Fork choice `get_checkpoint_block(store, root, epoch)`: resolves the ancestor at `get_checkpoint_slot(epoch)`.
- Honest validator: the FFG target is always `get_checkpoint_root(head_state, current_epoch)`; the first-slot-of-epoch special case (head block is its own target) disappears.
- Fork transition: epochs before activation MUST keep resolving under the previous anchoring. This keeps pre-activation finalized/justified checkpoints matching `get_checkpoint_block` on old blocks, and keeps attestations with pre-activation target epochs valid for one epoch after activation.

Key invariant that makes the change small: when the first slot of epoch `N` is empty, old and new anchoring already agree (both give the latest block before the epoch). Behavior changes only when a block occupies the epoch's first slot, and the only difference is "that block" versus "its parent chain's last pre-epoch block".

## 2. Visual overview of the implementation

### 2.1 What moves: the checkpoint anchor for epoch N

```text
 slot:        60      61      62      63   |   64      65      66
 block:     [B60]     --      --    [B63]  |  [B64]  [B65]     --
                                           |
                             epoch boundary (first slot of epoch N)

 old anchoring:  checkpoint(N) = B64   latest block at or before slot 64
 new anchoring:  checkpoint(N) = B63   latest block at or before slot 63

 if slot 64 is empty:  both anchorings give B63, the change is a no-op
 if slots 61-63 empty: new anchoring falls back to B60
 genesis epoch:        anchors at the genesis slot under both rules
```

### 2.2 One gate, two consumers

Everything funnels through a single epoch-gated helper, so the fork transition
rule (pre-activation epochs keep the old anchoring) holds everywhere at once:

```text
                 config/params/config.go:212
                 HezeForkEpoch (HEZE_FORK_EPOCH, default far-future)
                              |
                              v
                 time/slots/slottime.go:127
                 slots.CheckpointSlot(epoch)          <- get_checkpoint_slot
                 genesis -> slot 0
                 epoch <  fork epoch -> EpochStart(epoch)
                 epoch >= fork epoch -> EpochStart(epoch) - 1
                    |                             |
        state transition                     fork choice + blockchain
                    |                             |
                    v                             v
 beacon-chain/core/helpers/block.go:116   node target cache at insert
 helpers.CheckpointRoot(state, epoch)     beacon-chain/forkchoice/doubly-linked-tree/store.go:130
        <- get_checkpoint_root            (epoch-start block stops being its own target)
                    |                             |
     +--------------+--------------+              +-- targetRootForEpoch (walk unchanged)
     |              |              |              |   beacon-chain/forkchoice/doubly-linked-tree/forkchoice.go:846
     v              v              v              +-- IsViableForCheckpoint
 target match   justified cp   phase0 match       |   beacon-chain/forkchoice/doubly-linked-tree/forkchoice.go:257
 (Altair+)      recording      (uniformity)       +-- prune bound (checkpointMaxSlot)
 beacon-chain/  beacon-chain/  beacon-chain/      |   beacon-chain/forkchoice/doubly-linked-tree/store.go:309
 core/altair/   core/epoch/    core/epoch/        +-- equivocation prune bound
 attestation    precompute/    precompute/        |   beacon-chain/forkchoice/doubly-linked-tree/forkchoice.go:489
 .go:344        justification_ attestation        +-- fill-in walk stop bound
                finalization   .go:121            |   beacon-chain/blockchain/process_block_helpers.go:383
                .go:164,172                       +-- weak subjectivity window
                                                      beacon-chain/blockchain/weak_subjectivity_checks.go:44
```

### 2.3 How the new target reaches an attestation's life cycle

No caller-side changes: producers and validators already resolve targets
through fork choice, so fixing the cache fixes them all atomically.

```text
 produce                         validate (gossip + pool)        include in a block
 -------                         ------------------------        ------------------
 GetAttestationData              VerifyLmdFfgConsistency         participation flags
 beacon-chain/rpc/core/          beacon-chain/blockchain/        beacon-chain/core/altair/
 validator.go:537                receive_attestation.go:57       attestation.go:344
      |                               |                               |
      v                               v                               v
 TargetRootForEpoch              TargetRootForEpoch              helpers.CheckpointRoot
 (fork choice cache)             (fork choice cache)             (state accessor)

 epoch processing                                 fork choice weights
 ----------------                                 -------------------
 justified checkpoint roots                       justified balances at the checkpoint epoch
 beacon-chain/core/epoch/precompute/              beacon-chain/state/stategen/getter.go:104
 justification_finalization.go:164,172            via beacon-chain/forkchoice/doubly-linked-tree/forkchoice.go:768
```

### 2.4 Change map

| Area | Site | Change |
|---|---|---|
| Config | `config/params/config.go:212` | New `HEZE_FORK_EPOCH` spec parameter, far-future on all networks |
| Slot math | `time/slots/slottime.go:127` | New `slots.CheckpointSlot`, the literal `get_checkpoint_slot` |
| State accessor | `beacon-chain/core/helpers/block.go:116` | New `helpers.CheckpointRoot`, the literal `get_checkpoint_root` |
| Target matching | `beacon-chain/core/altair/attestation.go:344` | Matching-target compares against `CheckpointRoot` |
| Justification | `beacon-chain/core/epoch/precompute/justification_finalization.go:164` | Justified checkpoint roots come from `CheckpointRoot` |
| Phase 0 matching | `beacon-chain/core/epoch/precompute/attestation.go:121` | `SameTarget` uses `CheckpointRoot` for uniformity |
| Fork choice cache | `beacon-chain/forkchoice/doubly-linked-tree/store.go:130` | Post-activation epoch-start blocks target their parent, not themselves |
| Checkpoint viability | `beacon-chain/forkchoice/doubly-linked-tree/forkchoice.go:257` | Position checks use the checkpoint slot |
| Finalization pruning | `beacon-chain/forkchoice/doubly-linked-tree/store.go:309` | Conflict bound moves to the checkpoint slot; epoch-start children survive |
| Equivocation records | `beacon-chain/forkchoice/doubly-linked-tree/forkchoice.go:489` | Prune bound moves to the checkpoint slot |
| Fill-in walk | `beacon-chain/blockchain/process_block_helpers.go:383` | Stop bound moves to the checkpoint slot |
| Weak subjectivity | `beacon-chain/blockchain/weak_subjectivity_checks.go:44` | Search window ends at the boundary for post-activation epochs |
| Justified balances | `beacon-chain/state/stategen/getter.go:104` | Handler takes the checkpoint epoch and advances the boundary state to it |
| Block gossip | `beacon-chain/sync/validate_beacon_blocks.go:156` | "Would revert finalized" bound moves to the checkpoint slot |
| Blob gossip | `beacon-chain/verification/blob.go:148` | Same bound for blob sidecars |
| Column gossip | `beacon-chain/verification/data_column.go:268` | Same bound for data column sidecars |
| Envelope gossip | `beacon-chain/verification/execution_payload_envelope.go:109` | Same bound for Gloas payload envelopes |
| Peer status | `beacon-chain/sync/rpc_status.go:468` | Finalized-root sanity check accepts boundary roots with epoch-start children |
| Envelope serving | `beacon-chain/sync/rpc_execution_payload_envelopes_by_root.go:62` | By-root serving window opens at the checkpoint slot so the boundary block's envelope stays servable |
| Finalized state API | `beacon-chain/rpc/lookup/stater.go:146` | `state_id=finalized`/`justified` replay blocks only through the checkpoint slot, so checkpoint sync origins descend from the checkpoint root |
| Origin checkpoint | `beacon-chain/db/kv/wss.go:106` | `SaveOrigin` derives the checkpoint epoch from the state slot, not the (boundary) block slot |
| Test utility | `testing/util/attestation.go:200` | Generated attestations target `CheckpointRoot` |

Unchanged by design: the checkpoint state anchor (`getAttPreState` at `beacon-chain/blockchain/process_attestation_helpers.go:96` still advances to the epoch start), the target root query walk (`targetRootForEpoch` at `beacon-chain/forkchoice/doubly-linked-tree/forkchoice.go:846`), shuffling dependent roots (`dependentRootForEpoch` at `beacon-chain/forkchoice/doubly-linked-tree/forkchoice.go:816`), and every consumer of `TargetRootForEpoch`.

## 3. Where the equivalent logic lives in Prysm today

### 3.1 State transition (per-state accessors)

- `get_block_root` is `BlockRoot` at `beacon-chain/core/helpers/block.go:97`, built on `BlockRootAtSlot` at `beacon-chain/core/helpers/block.go:69`.
- Matching-target for participation flags (Altair and later): `MatchingStatus` at `beacon-chain/core/altair/attestation.go:341` calls `helpers.BlockRoot(beaconState, data.Target.Epoch)` at `beacon-chain/core/altair/attestation.go:344`. It is reached from `AttestationParticipationFlagIndices` at `beacon-chain/core/altair/attestation.go:283`, which is used by block processing, the upgrade-to-Altair migration and the rewards APIs.
- `weigh_justification_and_finalization` is `computeCheckpoints` at `beacon-chain/core/epoch/precompute/justification_finalization.go:153`; the new justified checkpoint roots come from `helpers.BlockRoot` at `beacon-chain/core/epoch/precompute/justification_finalization.go:164` and `beacon-chain/core/epoch/precompute/justification_finalization.go:172`. The same function backs `UnrealizedCheckpoints` at `beacon-chain/core/epoch/precompute/justification_finalization.go:19`, which fork choice uses for unrealized justification (`pullTips` at `beacon-chain/forkchoice/doubly-linked-tree/unrealized_justification.go:62`).
- Phase 0 target matching: `SameTarget` at `beacon-chain/core/epoch/precompute/attestation.go:120` (calls `helpers.BlockRoot` at `beacon-chain/core/epoch/precompute/attestation.go:121`).

### 3.2 Fork choice (per-chain resolution, `get_checkpoint_block`)

Prysm does not walk ancestors per query. Each node caches a `target` pointer (`beacon-chain/forkchoice/doubly-linked-tree/types.go:61`) assigned at insert time in `beacon-chain/forkchoice/doubly-linked-tree/store.go:130`:

- a node at an exact epoch start slot is its own target,
- a node whose parent is in the same epoch inherits the parent's target,
- a node whose parent is in an earlier epoch takes the parent as target.

Queries go through `targetRootForEpoch` at `beacon-chain/forkchoice/doubly-linked-tree/forkchoice.go:846`, exposed as `TargetRootForEpoch` at `beacon-chain/forkchoice/doubly-linked-tree/forkchoice.go:804` and consumed by:

- attestation production: `GetAttestationData` at `beacon-chain/rpc/core/validator.go:537` (both the gRPC and Beacon API endpoints route through it, results are cached in the attestation-data cache at `beacon-chain/rpc/core/validator.go:565`),
- LMD/FFG consistency for gossip and pool attestations: `VerifyLmdFfgConsistency` at `beacon-chain/blockchain/receive_attestation.go:56`, called from `beacon-chain/sync/validate_beacon_attestation.go:133` and `beacon-chain/sync/validate_aggregate_proof.go:164`,
- block-packing filters: `beacon-chain/rpc/prysm/v1alpha1/validator/proposer_attestations.go:537` and `beacon-chain/rpc/prysm/v1alpha1/validator/proposer_attestations.go:546`,
- data column verification state lookup: `beacon-chain/verification/data_column.go:386`.

Related fork-choice logic that encodes "the checkpoint block for epoch `E` sits at or before `EpochStart(E)`":

- `IsViableForCheckpoint` at `beacon-chain/forkchoice/doubly-linked-tree/forkchoice.go:257` (checkpoint-position viability, used by `getAttPreState` at `beacon-chain/blockchain/process_attestation_helpers.go:134`),
- `prune` at `beacon-chain/forkchoice/doubly-linked-tree/store.go:309` (`checkpointMaxSlot`, prunes finalized-block children that conflict with the finalized checkpoint),
- `UpdateFinalizedCheckpoint` at `beacon-chain/forkchoice/doubly-linked-tree/forkchoice.go:489` (equivocation-record pruning bound),
- `fillInForkChoiceMissingBlocks` at `beacon-chain/blockchain/process_block_helpers.go:383` (walk-back stop bound when re-inserting DB blocks into fork choice),
- `dependentRootForEpoch` at `beacon-chain/forkchoice/doubly-linked-tree/forkchoice.go:816`, which today derives the (unchanged) shuffling dependent root by stepping the old-style target back one node when it sits exactly at the epoch start.

### 3.3 Checkpoint states, justified balances

- The checkpoint *state* is unchanged by the EIP: `getAttPreState` at `beacon-chain/blockchain/process_attestation_helpers.go:96` still advances the checkpoint block's post-state to `EpochStart(target.Epoch)` before computing committees. That matches the spec's `store.checkpoint_states` before and after EIP-8333.
- Justified balances for fork-choice weights come from `updateJustifiedBalances` at `beacon-chain/forkchoice/doubly-linked-tree/forkchoice.go:768`, whose handler is `ActiveNonSlashedBalancesByRoot` at `beacon-chain/state/stategen/getter.go:104` (wired at `beacon-chain/state/stategen/service.go:118` through `BalancesByRooter` at `beacon-chain/forkchoice/interfaces.go:18`). It reads the state *at the justified root* and computes activeness at that state's own epoch. The spec reads the checkpoint state advanced to `EpochStart(E)`. Today the two agree whenever the justified block sits exactly at the epoch start (the common case). Under EIP-8333 the justified root is always a pre-epoch block, so the approximation would become a systematic one-epoch lag. See section 4.6.

### 3.4 Places verified to need no change

- `on_block` finalized-consistency: Prysm checks that the incoming block's ancestry reaches the finalized root (tree-rooted-at-finalized plus the walk in `fillInForkChoiceMissingBlocks`); it compares against the checkpoint root value, so it is agnostic to where that root sits (only the walk stop bound needs the off-by-one fix in 4.4).
- `AttestationTargetState` clock validation at `beacon-chain/blockchain/receive_attestation.go:42` uses `EpochStart(target.Epoch)` as a "not from the future" bound. The checkpoint state anchor stays the epoch start, so this stays.
- Shuffling dependent roots (`dependentRootForEpoch`, proposer/attester duties) are defined as "last block before the epoch" already; EIP-8333 makes checkpoints match them, not the other way around.
- DB finalized index, stategen migration, backfill and checkpoint-sync origin handling operate on root values and block ancestry, never on "checkpoint root must be an epoch-start block". `CleanUpDirtyStates` at `beacon-chain/db/kv/state.go:1015` keeps states from `EpochStart(finalized epoch)` on and keeps the finalized-root state explicitly, which stays safe (merely conservative) under the new anchoring.
- Slashing protection, slasher and doppelganger logic key on target epochs, not target root positions.
- Light client code fetches the finalized block by root from the state's finalized checkpoint; no positional assumption.
- Consensus containers, gossip encodings and the attestation pool are unchanged (the EIP moves a value, not a type).

## 4. Design

### 4.1 New config parameter

`HezeForkEpoch` (`HEZE_FORK_EPOCH`, `spec:"true"`) in `config/params/config.go`, default `FAR_FUTURE_EPOCH` in the mainnet config (testnet configs copy mainnet, so they inherit it) and `math.MaxUint64` in the minimal config, mirroring `FuluForkEpoch` at `config/params/config.go:209`. Upstream config files already carry `HEZE_FORK_EPOCH` (far-future) and `HEZE_FORK_VERSION` (`0x08000000`), tracked in `specrefs/configs.yml:366`; the epoch key is now mapped (removed from `placeholderFields` in `config/params/loader_test.go`), while the fork version stays a placeholder because this EIP needs no new fork version, state version or digest, so `ForkVersionSchedule` is untouched. Full Heze fork scaffolding (a `version.Heze` enum, upgrade functions and digest scheduling) belongs to whatever change ships the rest of the Heze fork.

The parameter is logged in `config/params/loader.go` next to the fork epochs and appears in the Beacon API spec endpoint (count bump in `beacon-chain/rpc/eth/config/handlers_test.go:248`).

### 4.2 New helpers (single source of truth)

- `slots.CheckpointSlot(epoch)` in `time/slots/slottime.go`: the literal `get_checkpoint_slot`. Genesis epoch resolves to the genesis slot, pre-activation epochs to `EpochStart(epoch)`, post-activation epochs to `EpochStart(epoch) - 1`. It lives in `time/slots` (not `core/helpers`) so fork choice can use it without importing state-aware packages.
- `helpers.CheckpointRoot(state, epoch)` in `beacon-chain/core/helpers/block.go`: the literal `get_checkpoint_root`, `BlockRootAtSlot(state, CheckpointSlot(epoch))`.

Because both helpers gate on the *checkpoint's epoch* against the activation epoch (never on wall time or state version), they can replace `BlockRoot` unconditionally at every checkpoint-resolution site: pre-activation epochs keep byte-identical behavior, which is exactly the EIP's fork-transition rule. This also means attestations with pre-activation target epochs included after activation are still evaluated under the old anchoring, as required.

### 4.3 State transition changes

Swap `helpers.BlockRoot` for `helpers.CheckpointRoot` at the three checkpoint-resolution sites:

- `MatchingStatus` at `beacon-chain/core/altair/attestation.go:344` (matching-target for participation flags; source and head matching unchanged),
- `computeCheckpoints` at `beacon-chain/core/epoch/precompute/justification_finalization.go:164` and `beacon-chain/core/epoch/precompute/justification_finalization.go:172` (new justified checkpoint roots; the unrealized-checkpoint path inherits this automatically),
- `SameTarget` at `beacon-chain/core/epoch/precompute/attestation.go:121` (phase 0 matching; a no-op unless someone configures activation during phase 0, kept for uniformity).

`helpers.BlockRoot` itself stays: it still implements `get_block_root`, which remains a spec function with other uses.

### 4.4 Fork choice changes

**Insert-time target cache** (`beacon-chain/forkchoice/doubly-linked-tree/store.go:130`): the only behavioral difference between old and new anchoring at insert time is the exact-epoch-start node. Old rule: it is its own target. New rule: its target is its parent (the boundary block). Nodes whose parent is in an earlier epoch already take the parent as target, which is correct under both anchorings, and same-epoch nodes inherit unchanged. So the change is to skip the "own target" branch when the node's epoch is at or past activation (with a genesis-epoch carve-out so a genesis-activated configuration still anchors epoch 0 at the genesis block):

```
epoch := slots.ToEpoch(slot)
if slot is an epoch start and (epoch < HezeForkEpoch or epoch == GenesisEpoch) {
    n.target = n
} else if parent != nil {
    same-epoch parent: inherit parent.target; earlier-epoch parent: target = parent
}
```

The rule keys off the node's own slot, so replaying the same blocks after a restart reproduces the same targets, and mixed pre/post-activation trees are handled per node.

**Query path** (`targetRootForEpoch` at `beacon-chain/forkchoice/doubly-linked-tree/forkchoice.go:846`): no change needed. The three branches remain correct:

- `epoch > nodeEpoch` returns the node itself: the node is the latest block at or before `EpochStart(epoch) - 1` too, since it precedes `epoch` entirely.
- `epoch == nodeEpoch` returns the cached target, which now carries the new semantics for post-activation nodes.
- `epoch < nodeEpoch` walks back through targets; its "step past a same-epoch target" branch only ever triggers for pre-activation nodes (post-activation targets are always in an earlier epoch), so mixed chains resolve each epoch under that epoch's own rule.

`dependentRootForEpoch` at `beacon-chain/forkchoice/doubly-linked-tree/forkchoice.go:816` also needs no change: its conditional step-to-parent only fires when the target sits inside the requested epoch, which post-activation targets never do, so dependent roots (which were already boundary-anchored) come out identical. This mirrors the EIP's rationale that the new checkpoint root equals the shuffling dependent root for the same chain view.

**Checkpoint-position logic**: replace `EpochStart(cp.Epoch)` with `slots.CheckpointSlot(cp.Epoch)` at:

- `IsViableForCheckpoint` at `beacon-chain/forkchoice/doubly-linked-tree/forkchoice.go:263`: a post-activation checkpoint root must sit at or before the boundary slot, and a chain containing a block in `(root.slot, boundary]` makes it non-viable. The structure of the checks is unchanged.
- `prune` at `beacon-chain/forkchoice/doubly-linked-tree/store.go:309`: children of the finalized block at or before the checkpoint slot conflict with finalization and are pruned; under the new anchoring a child exactly at the epoch start is compatible and must be kept.
- `UpdateFinalizedCheckpoint` at `beacon-chain/forkchoice/doubly-linked-tree/forkchoice.go:489`: equivocation records at the epoch start slot are not finalized under the new anchoring, so the pruning bound tightens by one slot.
- `fillInForkChoiceMissingBlocks` at `beacon-chain/blockchain/process_block_helpers.go:383`: the walk-back stop bound; without the fix, a legitimate descendant chain passing through a block exactly at the finalized epoch's start slot could be misreported as not descending from finalization.

The prune-time invariant at `beacon-chain/forkchoice/doubly-linked-tree/store.go:254` (a finalized node keeps its target only if it is its own target) needs no change: post-activation finalized nodes get a nil target, and queries for their own epoch already return the zero root for nil targets, which only affects attestations older than finalization (rejected upstream by `verifyAttTargetEpoch` at `beacon-chain/blockchain/process_attestation_helpers.go:168`).

### 4.5 Attestation production and validation

No direct changes. `GetAttestationData` at `beacon-chain/rpc/core/validator.go:537`, the packing filters at `beacon-chain/rpc/prysm/v1alpha1/validator/proposer_attestations.go:537`, and `VerifyLmdFfgConsistency` at `beacon-chain/blockchain/receive_attestation.go:56` all resolve targets through `TargetRootForEpoch`, so they inherit the new anchoring atomically with fork choice. This removes the honest-validator first-slot special case by construction: when the head is the first block of the epoch, its cached target is now its parent chain's boundary block rather than itself.

`getAttPreState` keeps advancing the target block's state to `EpochStart(target.Epoch)` (`beacon-chain/blockchain/process_attestation_helpers.go:149`), which is still the spec's checkpoint state.

### 4.6 Justified balances fidelity

Spec fork choice weighs votes with balances from the justified *checkpoint state* (the justified block's state advanced to `EpochStart(E)`). Prysm reads the state at the justified root without advancing (`ActiveNonSlashedBalancesByRoot` at `beacon-chain/state/stategen/getter.go:104`), which matches the spec only when the justified block sits at the epoch start. Under EIP-8333 that stops ever being the case, so the handler is made epoch-aware:

- `BalancesByRooter` at `beacon-chain/forkchoice/interfaces.go:18` gains the checkpoint epoch, passed by `updateJustifiedBalances` at `beacon-chain/forkchoice/doubly-linked-tree/forkchoice.go:768` from the justified checkpoint being adopted.
- When the state at the root is behind the checkpoint epoch, stategen advances a copy through the epoch boundary (using the next-slot cache when warm, which it typically is because the boundary block was head at the end of the previous epoch). This runs at most once per justified-checkpoint update, so roughly once per epoch.

This keeps Prysm's vote weights consistent with clients that implement `store.checkpoint_states` literally, and also fixes the same (previously rare) lag for skipped-first-slot checkpoints before activation.

### 4.7 Weak subjectivity checks

`NewWeakSubjectivityVerifier` at `beacon-chain/blockchain/weak_subjectivity_checks.go:40` derives the search window for the supplied checkpoint root as the slots of `wsc.Epoch`. Users obtain weak subjectivity checkpoints from finalized checkpoints, whose roots become boundary blocks after activation, so for post-activation epochs the window shifts one epoch back: the epoch of slots *ending at* the checkpoint slot. This carries the same tolerance as today's check (a one-epoch window, matching the existing behavior of not pinpointing the exact checkpoint slot). This is the "checkpoint sync providers and consumers" compatibility note from the EIP as it applies inside Prysm; the checkpoint-sync origin flow itself is root-based and unaffected.

### 4.8 Test utilities

`testing/util/attestation.go:200` builds attestation targets with `helpers.BlockRoot`; it switches to `helpers.CheckpointRoot` so generated attestations stay valid in tests that activate the EIP (a no-op for every existing test, since the default configs leave activation at `FAR_FUTURE_EPOCH`).

## 5. What changes on the wire and for users

- Nothing until `HEZE_FORK_EPOCH` is set in a config.
- After activation: attestation targets for epoch `N` name the boundary block; state-recorded justified/finalized checkpoint roots move accordingly; `finality_checkpoints`, fork choice dumps and light client finalized headers expose the new roots. Epoch semantics ("epoch N finalized") tighten to "all of epochs up to N - 1 finalized".
- Tooling that assumed "checkpoint root is a block in the checkpoint epoch" must adapt (inside Prysm that was only the weak subjectivity window, section 4.7).

## 6. Tests (implemented)

Mapped to the EIP's test-case list, all with `HezeForkEpoch` overridden in the test config:

1. `slots.CheckpointSlot`: genesis epoch, pre-activation epoch, activation epoch, activation at genesis, overflow guard. `TestCheckpointSlot` in `time/slots/slottime_test.go`.
2. `helpers.CheckpointRoot`: boundary block differs from `BlockRoot` when the first slot is occupied, equals it when the first slot is empty, falls back through empty trailing slots, genesis epoch resolves to the genesis root. `TestCheckpointRoot` in `beacon-chain/core/helpers/block_test.go`.
3. Target matching for participation flags: a boundary-block target matches post-activation, the epoch's first block does not, and pre-activation epochs keep the previous anchoring. `TestMatchingStatus_EIP8333` in `beacon-chain/core/altair/attestation_test.go`.
4. Justification: the recorded justified checkpoint root is the boundary root post-activation. `TestProcessJustificationAndFinalizationPreCompute_EIP8333` in `beacon-chain/core/epoch/precompute/justification_finalization_test.go`.
5. Fork choice targets: boundary anchoring for post-activation epochs including the exact-epoch-start-node case, old anchoring for pre-activation epochs across a mixed chain, empty-boundary fallback, future-epoch resolution, and unchanged dependent roots. `TestStore_TargetRootForEpoch_EIP8333` in `beacon-chain/forkchoice/doubly-linked-tree/eip8333_test.go`.
6. Checkpoint viability: the boundary slot is viable, the epoch start slot is not, post-activation. `TestForkChoice_IsViableForCheckpoint_EIP8333` in the same file.
7. Finalization pruning: an epoch-start child of the finalized boundary block survives post-activation and is pruned pre-activation. `TestStore_PruneIncompatibleChildren_EIP8333` in the same file.
8. LMD/FFG consistency: boundary targets accepted, first-block targets rejected, post-activation. `TestVerifyLMDFFGConsistent_EIP8333` in `beacon-chain/blockchain/receive_attestation_test.go`. Attestation production (`GetAttestationData` at `beacon-chain/rpc/core/validator.go:537`) is a straight delegation to the same `TargetRootForEpoch` and is covered through it.
9. Weak subjectivity: a boundary-block checkpoint passes verification and an epoch-start block fails it post-activation. `TestService_VerifyWeakSubjectivityRoot_EIP8333` in `beacon-chain/blockchain/weak_subjectivity_checks_test.go`.
10. Justified balances: a boundary-block state advances to the checkpoint epoch without mutating the cached state. `TestActiveNonSlashedBalancesByRoot_AdvancesToCheckpointEpoch` in `beacon-chain/state/stategen/getter_test.go`.
11. Gossip finalized bounds: sidecars and envelopes at the finalized epoch's first slot pass post-activation while boundary-slot ones stay ignored. `TestSlotAboveFinalized_Heze` in `beacon-chain/verification/blob_test.go`, `TestColumnSlotAboveFinalized_Heze` in `beacon-chain/verification/data_column_test.go`, `TestEnvelopeVerifier_VerifySlotAboveFinalized_Heze` in `beacon-chain/verification/execution_payload_envelope_test.go`.
12. Peer status: a boundary finalized root with a finalized child at the epoch start validates post-activation, an epoch-start root does not. `TestValidateStatusMessage_HezeBoundaryFinalizedRoot` in `beacon-chain/sync/rpc_status_test.go`.
13. Finalized state lookup: `state_id=finalized` and `justified` resolve through the checkpoint slot post-activation. The post-Heze subtest of `TestGetState` in `beacon-chain/rpc/lookup/stater_test.go`.
14. Checkpoint sync origin: an origin with a boundary block and epoch-start state records the checkpoint epoch from the state slot. `TestSaveOrigin_BoundaryBlockCheckpointEpoch` in `beacon-chain/db/kv/wss_test.go`.

### Live network verification

Two devnet-style runs were performed on 2026-09-07 with Heze activating at epoch 2 on a live chain (epochs 0 and 1 under the previous anchoring):

- Prysm's multi-node e2e (`TestEndToEnd_MinimalConfig` in `testing/endtoend/minimal_e2e_test.go`, which now schedules `HezeForkEpoch = 2`): four beacon nodes, validators, geth ELs, 10 epochs, ending with a checkpoint-synced late joiner. The final run passed completely (358 subtests), with every per-epoch evaluator green under Heze (`finalizes_at_epoch`, `validators_participating`, same-head, voluntary exits, withdrawals) and the joiner checkpoint-syncing from a boundary-block origin (`blockSlot=63, stateSlot=64`) to the live head. Earlier runs of the checkpoint-sync phase exposed two real bugs, both cases of peers correctly rejecting a joiner whose advertised finalized checkpoint was malformed: the `state_id=finalized` resolution served an origin whose header named the (not yet final) epoch-start block (`invalid finalized root`, fixed at `beacon-chain/rpc/lookup/stater.go:146`), and `SaveOrigin` derived the checkpoint epoch from the boundary block's slot, one epoch short (`invalid epoch`, fixed at `beacon-chain/db/kv/wss.go:106`, also a latent pre-Heze bug for checkpoints whose epoch starts with an empty slot). The harness's head-match wait at `testing/endtoend/endtoend_test.go:273` was also hardened to retry transient errors until its deadline instead of aborting on the first non-NotFound response.
- A kurtosis devnet (2 Prysm + 2 geth, gloas at 1, Heze at 2, minimal preset; args at `/Users/sdas/kurtosis/eip8333-heze-test.yaml`, assertions at `/Users/sdas/kurtosis/check_heze.sh`): the generated config carried `HEZE_FORK_EPOCH: 2`, finality ran at full speed across activation (finalized = head epoch - 2), and the justified and finalized checkpoint roots landed exactly on the boundary slots (`justified(5)` at slot 39, `finalized(4)` at slot 31, first post-activation `justified(2)` at slot 15), with both nodes agreeing on head.

The upstream consensus-spec-tests do not ship EIP-8333 vectors yet; when they do, the spectest runner picks them up through the same config plumbing. Existing spec vectors keep passing because every changed site resolves identically while `HEZE_FORK_EPOCH` is far-future.

## 7. Cross-client notes: Lodestar PR 9698

Lodestar's implementation (<https://github.com/ChainSafe/lodestar/pull/9698>, by EIP co-author lodekeeper, open draft as of 2026-09-04) agrees with this implementation on every site both clients share: the `computeCheckpointSlotAtEpoch`/`getCheckpointRoot` helpers, participation-flag target matching, phase 0 pending-attestation matching, justified checkpoint recording, fork choice target computation for epoch-start blocks, attestation production, and leaving the dependent-root logic untouched. Their justified-balances cache is re-keyed by the boundary root with balances captured at the epoch transition, which lands in the same place as the epoch-aware advance in section 4.6.

Differences and their resolution in Prysm:

- Gating: Lodestar gates on `ForkSeq.heze` / `HEZE_FORK_EPOCH`, matching upstream config files which already carry the Heze keys. Prysm now uses the same name (`HezeForkEpoch`, section 4.1) without full Heze fork scaffolding, since the EIP needs only the epoch.
- Lodestar also moves several "finalized slot" bounds that the published EIP text does not list, treating "finalized" as ending at the checkpoint slot. Prysm's handling:
  - `process_pending_deposits` finalized bound at `beacon-chain/core/electra/deposits.go:281`: deliberately left at the epoch start. This one is consensus-critical (mismatched clients compute different states) and the published EIP text does not modify it; match the consensus-specs change when it lands. The site carries a comment pointing here.
  - Gossip "would revert finalized" bounds: adopted. `beacon-chain/sync/validate_beacon_blocks.go:156`, `beacon-chain/verification/blob.go:148`, `beacon-chain/verification/data_column.go:268`, and `beacon-chain/verification/execution_payload_envelope.go:109` now compare against `slots.CheckpointSlot` of the finalized epoch. Post-activation, a block at exactly the finalized epoch's start slot is not necessarily reverted by finality, so the old bound wrongly ignored it (liveness, not safety; IGNORE-level conditions).
  - Peer status validation at `beacon-chain/sync/rpc_status.go:468`: adopted. The finalized-root sanity check now compares against the checkpoint slot, with the "block inside the checkpoint epoch" acceptance kept for pre-activation epochs only. Without this, a finalized child at the epoch start slot (the normal case post-activation) flagged honest peers as invalid. Lodestar's PR does not fix their equivalent; flag it on their PR.
- Sites Lodestar changed with no Prysm analog: checkpoint-state cache re-keying (Prysm keys checkpoint states by the attestation's checkpoint object, which already carries the boundary root), fork choice initialization from synthetic checkpoint nodes (Prysm inserts real anchor blocks), a per-block root cache, and range-sync chain targets (Prysm's initial sync fetches slot ranges and follows parent chains rather than syncing to a root-at-slot target).
- Known pre-existing quirk Heze makes common: `/eth/v1/beacon/states/finalized/root` (`finalizedStateRoot` at `beacon-chain/rpc/lookup/stater.go:402`) returns the checkpoint block's own state root at its slot, while the full-state resolution serves the epoch-aligned state one empty-slot advance later. The same divergence exists today whenever the finalized epoch's first slot is empty; checkpoint sync consumers use the full state and derive the block from its header, so they are unaffected.
- Prysm-only sites with no Lodestar analog: the weak subjectivity window (section 4.7), `IsViableForCheckpoint` positional viability, and the equivocation-record prune bound.

## 8. Alternatives considered

- Gating on state/fork version instead of an epoch parameter: rejected; the EIP changes no containers or fork versions, and an epoch gate (`HEZE_FORK_EPOCH`) composes with the rest of the Heze fork whenever Prysm grows its scaffolding.
- Changing `targetRootForEpoch` at query time instead of the insert-time cache: rejected; the cache is the semantic ("this node's checkpoint"), the query walk is shared with dependent-root derivation, and adjusting at query time would have made `dependentRootForEpoch` double-step.
- Leaving justified balances as-is: rejected; the one-epoch lag would become systematic and diverge from clients that follow `store.checkpoint_states` literally.
