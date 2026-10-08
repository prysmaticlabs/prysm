package decoupled

import (
	"time"

	"github.com/OffchainLabs/prysm/v7/consensus-types/primitives"
)

// Vote is an AC vote: validator, slot, head.
type Vote struct {
	Validator primitives.ValidatorIndex
	Slot      primitives.Slot
	Root      [32]byte
}

// Pair identifies a block at an FG height.
type Pair struct {
	Height primitives.Height
	Root   [32]byte
}

// FGSummary contains a block state's claims, not the store's accepted J and F.
type FGSummary struct {
	Target    Pair
	Justified Pair
	Finalized Pair
}

// Node is a store-owned block in the tree rooted at accepted F.
type Node struct {
	Root     [32]byte
	Slot     primitives.Slot
	Parent   *Node
	Children []*Node
	// Votes includes participation for blocks the proposer had not processed.
	Votes     []Vote
	Supported []Vote
	FG        FGSummary
	// processed is the insertion or restart replay time.
	processed time.Time
}

// bucket retains two distinct votes per validator, enough to detect equivocation.
// at preserves first processing times across duplicate receipts.
type bucket struct {
	votes       []Vote
	at          []time.Time
	byValidator map[primitives.ValidatorIndex][]int
}

type lock struct {
	block    *Node
	deadline time.Time
}

// Lock is a lock without tree pointers, for callers outside the store.
type Lock struct {
	Root     [32]byte
	Slot     primitives.Slot
	Deadline time.Time
}

// Store holds Goldilocks fork-choice state. Callers must serialize all access.
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

// supportCounts separates participation from non-equivocating subtree support.
type supportCounts struct {
	equivocators int
	participants int
	subtree      map[*Node]int
}

// view separates observed participation from usable block support.
// Votes for unknown blocks still affect participation and equivocation checks.
type view struct {
	participants []Vote
	supported    []Vote
}

// frame restricts heads to root's subtree and target heights at or above floor.
type frame struct {
	root  *Node
	floor primitives.Height
}

// Block is validated input to InsertNode, which takes ownership of its vote slices.
type Block struct {
	Root      [32]byte
	Parent    [32]byte
	Slot      primitives.Slot
	Votes     []Vote
	Supported []Vote
	FG        FGSummary
}

// DumpNode describes a retained block. Votes and Support count carried votes,
// not fork-choice scores. Viable marks paths to eligible heads; Confirmed marks
// ancestors of public confirmation; Locked marks exact live lock entries.
type DumpNode struct {
	Root, Parent   [32]byte
	Slot           primitives.Slot
	FG             FGSummary
	Votes, Support int
	Processed      time.Time
	Viable         bool
	Confirmed      bool
	Locked         bool
}

// DumpSaved records a confirmation candidate before lock vetoes; zero means none.
type DumpSaved struct {
	Slot primitives.Slot
	Root [32]byte
}

// DumpVoters counts distinct voters for one slot.
type DumpVoters struct {
	Slot  primitives.Slot
	Count int
}

// Dump is a snapshot with roots instead of pointers. Frame and Floor are the FG
// head restrictions. Confirmed is clipped to the head; LatestConfirmed is not.
type Dump struct {
	Slot                                                          primitives.Slot
	Joined                                                        bool
	RestartTime                                                   time.Time
	Head, Confirmed, LatestConfirmed, Justified, Finalized, Frame [32]byte
	JustifiedSlot, FinalizedSlot                                  primitives.Slot
	JustifiedHeight, FinalizedHeight, MaxHeight, Floor            primitives.Height
	CommitteeSize                                                 uint64
	Locks                                                         []Lock
	FastConfirmed                                                 [][32]byte
	Saved                                                         []DumpSaved
	Voters                                                        []DumpVoters
	Nodes                                                         []DumpNode
}
