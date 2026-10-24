# Decoupled fork choice implementation

This write-up explains the structures and algorithms in `beacon-chain/forkchoice/decoupled`, how Prysm calls them, and their memory and CPU costs. Optimization ideas are identified separately from the existing implementation. TBD: profiling.

The package implements Goldilocks fork choice. Available-chain (AC) votes supply branch support, while finality-gadget (FG) restricts eligible branches. Local receipt times determine which votes and blocks are available to each operation.

## The two main structures

There are two types that look like the fork choice entry point: `ForkChoice` and `Store`.

### ForkChoice connects the algorithm to Prysm

[`ForkChoice`](../beacon-chain/forkchoice/decoupled/forkchoice.go) implements the shared Prysm interface. Its important fields are:

```go
legacy          *doublylinkedtree.ForkChoice
store           atomic.Pointer[Store]
exec            map[[32]byte]*payload
head, confirmed [32]byte
highestSlot     primitives.Slot
highestRoot     [32]byte
```

It also embeds an `RWMutex` and holds a database ancestor lookup callback. Before activation, methods delegate to `legacy` fork choice. After activation, methods use `store`. `Activate` loads the stored chain and execution metadata before publishing the new store pointer, then replaces the legacy tree with an empty one to release its votes and balances.

The atomic pointer lets callers check whether activation has happened without locking. Reading or changing the store's contents still requires the mutex. There is no separate lock per map or node.

`exec` stores execution information that the decoupled algorithm does not need: block hash, parent hash, builder index, gas limit, and whether the payload has arrived. This keeps execution compatibility out of the core tree operations. The cost is a second root-indexed map and a separate payload allocation per block in the tree.

`head` and `confirmed` are the roots published by the blockchain service. Calling the algorithm's `Head` method can advance time and recompute a different result. A cached getter does neither.

### Store owns the algorithm state

[`Store`](../beacon-chain/forkchoice/decoupled/types.go) holds the tree, votes, and timing state:

```go
type Store struct {
    committeeSize   uint64
    genesis         time.Time
    restartTime     time.Time
    ticked          time.Time
    nodes           map[[32]byte]*Node
    finalized       *Node
    justified       *Node
    hJ              primitives.Height
    hF              primitives.Height
    hMax            primitives.Height
    pool            map[primitives.Slot]*bucket
    confirmations   map[primitives.Slot]*Node
    locks           []lock
    fastConfirmed   []*Node
    latestConfirmed *Node
    duty            duty
}
```

| Fields | What they represent |
| --- | --- |
| `nodes` | Root-to-node lookup for the tree |
| `finalized`, `justified`, `hF`, `hJ`, `hMax` | Accepted FG state and the max target height used for filtering |
| `pool` | Vote observations indexed by slot |
| `confirmations` | Per-slot candidates before lock vetoes |
| `locks` | Blocks with expiration deadlines |
| `fastConfirmed` | Latest evaluation's potential FG sources |
| `latestConfirmed` | The underlying confirmation record |
| `duty` | Recorded AC voting head and FG source/head |
| `genesis`, `restartTime`, `ticked`, `committeeSize` | Timing and threshold inputs |

The caller passes `now` to operations that advance time. This makes event ordering explicit and allows the algorithm to be exercised with controlled timestamps.

## Node and the block tree

Each block has one [`Node`](../beacon-chain/forkchoice/decoupled/types.go):

```go
type Node struct {
    Root      [32]byte
    Slot      primitives.Slot
    Parent    *Node
    Children  []*Node
    Votes     []Vote
    Supported []Vote
    FG        FGSummary
    processed time.Time
}
```

The root map provides direct lookup by block root. Parent pointers support ancestor checks, common-ancestor searches, support propagation, and lock retreat. Child slices support GHOST traversal and subtree pruning. Keeping both directions avoids finding children by scanning the entire map.

Each node is allocated separately, and its parent and children are linked through pointers. Removing a branch does not change the pointers to surviving nodes. However, nodes can be scattered across memory, so walking the tree can require loading data from many different locations. An alternative is to store nodes together in a slice and link them by index. This could reduce allocations and improve CPU cache usage, but moving nodes to fill deleted entries would require updating their indices wherever they are referenced. Whether that change is worthwhile needs profiling.

`Votes` contains all AC votes carried by the block. `Supported` contains the subset the proposer marked as supported. The distinction matters when current proposals supplement a voter's frozen local view.

`blockVotes` expands aggregate bits into individual `Vote` values. Supported values are appended into a second slice. Insertion reuses both slices, but votes can be duplicated across the lists and stored blocks.

`processed` records when this block became available to the local algorithm.

## Vote buckets and temporary views

A vote stores the validator index, slot, and block root:

```go
type Vote struct {
    Validator primitives.ValidatorIndex
    Slot      primitives.Slot
    Root      [32]byte
}
```

```go
type bucket struct {
    votes       []Vote
    at          []time.Time
    byValidator map[primitives.ValidatorIndex][]int
}
```

At most two distinct votes are stored per validator per slot. One counts toward participation. Two indicate equivocation. Duplicate receipt does not replace the original timestamp.

`Store.pool` maps each slot to a `bucket` and keeps the current and previous slots. Each node also keeps its own vote lists.

Head selection collects votes in a local `view`:

```go
type view struct {
    participants []Vote
    supported    []Vote
}
```

`countSupport` takes that view and returns the counts in a `supportCounts`:

```go
type supportCounts struct {
    equivocators int
    participants int
    subtree      map[*Node]int
}
```

`subtree[n]` counts non-equivocating votes for node `n` and its descendants. These two structs are built during selection and are not fields of `Store`.

This work allocates slices and maps on each calculation. `voterView` uses maps to remove duplicate votes, and `viableTree` builds a map of which nodes can lead to an eligible head.

## The callers of the algorithms

The main callers are in [`blockchain/decoupled.go`](../beacon-chain/blockchain/decoupled.go).

| Trigger | Call path | Result |
| --- | --- | --- |
| Validated block import | Blockchain insertion → `ForkChoice.InsertNode` or `ForkChoice.InsertChain` → `add` → store `InsertNode` | Tree and FG state updated, carried votes recorded |
| AC gossip or validator submission | Sync subscriber or validator RPC → `ReceiveAvailableAttestation` → `ProcessVote` | Vote fields and processing time recorded |
| Scheduled tick | `spawnDecoupledTicker` → `tickDecoupled` → `OnTick` | Due phases processed and duties recorded |
| Head update | `UpdateHead` → `publishDecoupledHead` → `Head` and `Confirmed` | Head and confirmed roots published, execution client notified when needed |
| Block production | `DecoupledProposal` → `Proposal` | Parent and vote lists returned together and cached for the slot |
| AC voting duty | Validator RPC → `DecoupledVote` → `Vote` | Previously recorded AC head returned |
| FG duty | Validator RPC → `DecoupledFinalityData` → `FGInputs` | Previously recorded source and head returned |
| Initial sync or resync completion | `onDecoupledSyncComplete` → `Restart` | Fresh observation period begins |
| Execution invalidation | `ForkChoice.SetOptimisticToInvalid` → `Remove` | Requested subtree removed |
| Debug RPC or tick event | `DecoupledForkChoiceDump` → `Dump` | Detached snapshot returned |

Block insertion, `ReceiveAvailableAttestation`, `tickDecoupled`, `UpdateHead`, `DecoupledProposal`, `DecoupledVote`, `DecoupledFinalityData`, and `DecoupledForkChoiceDump` acquire the same `ForkChoice` write lock.

## Block insertion and finality updates

`ForkChoice` converts the block and post-state into a `Block` input containing roots, slot, expanded votes, and an FG summary. The core's [`InsertNode`](../beacon-chain/forkchoice/decoupled/insert.go) does the following:

1. Runs due ticks before incorporating the block.
2. Ignores duplicate roots and rejects an unknown parent.
3. Allocates a node, links it to its parent, and adds it to the root map.
4. Updates max height and the accepted justified pair.
5. Accepts finalization from the post-state only when the node is still in the tree, is an ancestor of the accepted justified node, and has a viable descendant.
6. Records the block's carried votes at the processing time.

Justification compares `(height, root)` pairs lexicographically. Finalization moves the tree root and prunes branches. Keeping the compact FG summary on each node avoids loading beacon state during finalization.

The basic map insertion is inexpensive. Total insertion cost can include aggregate expansion, due ticks, ancestry checks, descendant scans, and pruning.

## Vote ingestion and time advancement

[`ProcessVote`](../beacon-chain/forkchoice/decoupled/votes.go) calls `OnTick` first, then records each supplied validator's vote. `addVote` rejects votes outside the current or previous slot and preserves the first processing time. A vote for an unknown root is stored because it still counts toward participation and can reveal equivocation.

When support needs a timestamp, `processedAt` computes:

```text
max(first vote processing time, referenced block processing time)
```

Both nodes and buckets record timestamps. A vote counts only after both the vote and its block are processed, and only if the block is no newer than the vote's slot.

[`OnTick`](../beacon-chain/forkchoice/decoupled/schedule.go) processes due phases at their scheduled instants. Let `Δ` be one quarter of a slot:

| Time in slot s | What happens |
| --- | --- |
| `Δ` | Use previous-slot votes to update locks, then record the AC voting head |
| `2Δ` | Evaluate the previous slot's confirmation and record FG inputs on FG attestation duty slots |
| `3Δ` | Cutoff for the local view used by the following slot's voter |
| `5Δ` | Use slot s votes to move locks back when at least half the committee voted for conflicting branches |
| `6Δ` | Close slot s participation and equivocation accounting and evaluate its confirmation |

Confirmation support for slot s must be available before its `2Δ`, even though confirmation is evaluated at `6Δ`. For the next slot's voter view, Prysm checks when each previous-slot vote and its block were processed. A vote counts toward participation only if received before `3Δ`. It counts toward support only if both the vote and its block were processed before `3Δ`.

`OnTick` runs scheduled phases before processing input received at the same instant. A delayed tick processes missed phases in order, without repeating completed phases.

`New` leaves `ticked` at zero, so the first `OnTick` loops through every slot since genesis. Starting near `restartTime` could reduce startup work. Scheduled steps must still run in order, and proposals and AC votes must wait until the node has observed the entire previous slot.

## Head selection

[`selectHead`](../beacon-chain/forkchoice/decoupled/head.go) selects the FG root and minimum target height, counts support, selects an anchor, and runs GHOST. Each caller selects which blocks and votes to include.

### Frame and viable paths

A `frame` contains the root pointer and minimum target height used to filter possible heads. Normally the root is finalized `F` and the floor is `max(hMax - 1, 0)`. When `hMax == hJ + 1`, the root becomes justified `J`, unless a competing branch outside `J` also reaches `hMax`. That case uses `F` with floor `hMax`.

[`viableTree`](../beacon-chain/forkchoice/decoupled/filter.go) recursively marks paths under the frame root leading to heads that meet the floor. It builds a `map[*Node]bool`, which makes subsequent eligibility checks simple lookups. The tradeoff is an allocation and a traversal of the relevant tree on each build.

### Support and GHOST

[`countSupport`](../beacon-chain/forkchoice/decoupled/ghost.go) counts participants and equivocators. Each supported vote from a validator who did not equivocate adds one to the block's support and to each ancestor up to the finalized root.

For head selection:

```text
score(block) = equivocator count + non-equivocating subtree support
majority(block) = 2 * score(block) > distinct participant count
```

Head selection adds one to every block's score for each equivocating validator. Confirmation counts these validators as participants but adds nothing to block scores.

GHOST follows the highest-scoring eligible child, breaking ties by greater root. Eligibility normally requires a majority, with exceptions for current-slot proposals, pre-fork blocks, and children whose parent is below the height floor.

The anchor is the deepest live lock, backed up to a viable ancestor. Without such a lock it is the frame root. Anchor selection uses the full tree even when the caller supplies a restricted voter view.

Each vote walks up to `D` ancestors, so `V` votes can cost `O(V * D)`. We could first count supported votes per block, excluding equivocators, then add child totals to parents in one pass.

### Different views

| Output | How it is selected |
| --- | --- |
| `Head` | Uses votes from the previous slot before `2Δ` of the current slot, then votes from the current slot. |
| `Proposal` | Uses votes from the previous slot to select the parent and returns the same vote lists for inclusion in the block. |
| Recorded AC vote | Combines the previous slot's local view at `3Δ` with votes included in blocks from the current slot. |
| FG source and head | Selects the deepest eligible source that passed fast confirmation. At the FG duty's `2Δ`, selects the attestation head using current-slot votes recorded in `pool`. |

At `Δ`, `voteHead` uses `voterView(s)` for votes and `voterBlocks(s)` for blocks when selecting the AC voting head.

[`voterView(s)`](../beacon-chain/forkchoice/decoupled/views.go) combines:

- Votes from `pool[s-1]` recorded before the previous slot's `3Δ`. They add support only if their blocks were also processed before then.
- Votes included in blocks from slot `s`. `Votes` adds participation. `Supported` adds support for known blocks, without the `3Δ` cutoff.

It removes duplicate votes but keeps conflicting votes to detect equivocation.

[`voterBlocks(s)`](../beacon-chain/forkchoice/decoupled/views.go) includes blocks processed before the cutoff, plus blocks from slot `s` and their ancestors.

Each function scans `st.nodes`. `voterView` allocates vote slices and maps to remove duplicates. `voterBlocks` uses the `late` map to avoid walking shared ancestors again.

## Confirmation, locks, and recorded duties

[`updateConfirmation`](../beacon-chain/forkchoice/decoupled/confirmation.go) is called by `OnTick` at `2Δ` to evaluate the previous slot. It counts participation before that vote slot's `6Δ`, but accepts support only when the vote and block were both available before `2Δ`. An equivocator counts as one participant but adds no support to any block.

Confirmation runs GHOST from `F`, using majority support without the head-selection height floor. `confirmations[slot]` records the candidate, or nil if no block has a majority. A conflicting live lock stops the update, but the recorded candidate still counts toward later slow-lock checks.

Fast locks need support from at least half the committee. For slow locks, `streakAncestor` finds the common ancestor of the last `lockStreak()` candidates. A missing or removed candidate breaks the streak.

The parent walk checks each block `n` from the candidate to `F`, with `streak` set to that common ancestor, `slow = false`, and `locked = nil`:

```go
fast := 2*uint64(c.score(n)) >= st.committeeSize
if fast {
    st.fastConfirmed = append(st.fastConfirmed, n)
}
slow = slow || n == streak
if locked == nil && (fast || slow) {
    locked = n
}
```

Only the first block passing either test gets a lock.

Only blocks passing `fast` enter `fastConfirmed`. `source` returns the first one still in the tree and on a branch allowed by FG.

Each [`lock`](../beacon-chain/forkchoice/decoupled/types.go) records:

```go
type lock struct {
    block    *Node
    deadline time.Time
}
```

`vetoed` checks for a live lock on another branch. `anchor` scans for the deepest live lock. Ancestors and descendants do not conflict.

`unlock` moves each live lock toward `F` while at least half the committee voted for conflicting branches. Equivocators are excluded.

`Confirmed` returns the common ancestor of `latestConfirmed` and the current head, using `F` if the recorded block was removed. It leaves `latestConfirmed` unchanged.

`duty` records the AC vote head and FG source/head at their scheduled times. `Vote` and `FGInputs` process due phases, then return those nodes if still in the tree. The blockchain service caches the proposal's parent and vote lists for the slot.

## Memory footprint and object lifetime

Tree nodes, vote lists, vote buckets, and the `ForkChoice.exec` map stay in memory between calls. Each head selection also allocates temporary slices and maps.

These sizes are calculated from the Go structs on a 64-bit system. They exclude slice contents, map entries, objects reached through pointers, and extra space allocated by Go.

| Value | Inline size | Additional storage |
| --- | --- | --- |
| `Vote` | 48 bytes | No referenced storage. Copies appear in several slices. |
| `FGSummary` | 120 bytes | Three inline `(height, root)` pairs |
| `Node` | 264 bytes | Storage for the elements of `Children`, `Votes`, and `Supported` |
| `bucket` | 56 bytes | Vote/time arrays, map storage, and per-validator index slices |
| `lock` | 32 bytes | References an existing node |

Vote slices use about `48 * (cap(Votes) + cap(Supported))` bytes per node, including unused capacity. Each bucket also records a timestamp per vote and an index slice per validator.

Storage options include two fixed positions and a count per validator, indices into `Votes` for `Supported`, and aggregates stored without expanding each vote.

[`prune`](../beacon-chain/forkchoice/decoupled/tree.go) detaches the new finalized root and rebuilds the map from its subtree. Both maps and a traversal slice exist during the rebuild. Sizing the new map from the old entry count can waste space.

Removed nodes stay in memory while locks, duties, confirmation history, or cached pointers reference them. Parent and child pointers can keep whole branches alive. `inTree` excludes removed nodes but does not clear these references.

Without finalization, blocks and their vote lists accumulate. `pool` still holds only two slots of votes.

## Repeated work and lock contention

These calls repeat head selection:

- `publishDecoupledHead` selects the head twice: once through `Head(now)` and again through `Confirmed(now)`.
- `tickDecoupled` calls `Head` and `Confirmed` at `2Δ` to update metrics. The ticker then calls `UpdateHead`, repeating head selection.
- `Dump` selects the head twice to fill the snapshot's `Confirmed` and `Head` fields. It also calls `viableTree` again to fill each node's `Viable` field.
- `notifyDecoupledTick` creates a full snapshot, including node data the event never uses.

Caching across calls must account for changes to votes, the tree, locks, and time.

These calls hold the write lock, including block/state reads in `publishDecoupledHead`. Votes get timestamps after acquiring that lock, so waiting can push them past a cutoff.

Before applying a result computed outside the lock, check that the store has not changed. If votes are timestamped on arrival, process them before later scheduled steps without revising recorded duties.

## Incomplete work

- Complete optimistic status, PTC, FULL/EMPTY selection, and latest-valid-hash invalidation.
- Add committee-size updates and restore locks and confirmation after restart.
- Reuse head selection within each call and omit unused node data from tick snapshots.
- Profile `Store` separately from vote expansion, database access, and lock waits in `ForkChoice` and the blockchain service.
- Measure memory with competing branches, repeated votes, and delayed finalization. Check how much pruning frees.
- Profile `countSupport`, confirmation, and unlocking as vote count and tree depth grow.
- Measure the first tick on an old chain separately from later ticks.
