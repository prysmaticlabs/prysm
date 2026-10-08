package decoupled

import (
	"time"

	"github.com/OffchainLabs/prysm/v7/consensus-types/primitives"
)

// Spec: GoldfishVote, one validator's AvailableChainAttestationData.
type Vote struct {
	Validator     primitives.ValidatorIndex
	Slot          primitives.Slot
	Root          [32]byte
	PayloadStatus bool
}

// Spec: HeightPair.
type Pair struct {
	Height primitives.Height
	Root   [32]byte
}

// The post-state pairs the fork choice reads, target_pair, justified_pair and finalized_pair.
type FGSummary struct {
	Target    Pair
	Justified Pair
	Finalized Pair
}

type Node struct {
	Root       [32]byte
	Slot       primitives.Slot
	Parent     *Node
	Children   []*Node
	BlockVotes []Vote
	FG         FGSummary
	processed  time.Time
}

type poolEntry struct {
	vote Vote
	at   time.Time
}

// Spec: Store Σ without the SG fields. TODO: stabilization, the walk anchor is the FG root and the stable record is F until the committee-lock design lands.
type Store struct {
	genesis         time.Time
	now             func() time.Time
	root            *Node
	nodes           map[[32]byte]*Node
	pool            map[primitives.Slot][]poolEntry
	finalized       *Node
	justified       *Node
	hJ              primitives.Height
	hMax            primitives.Height
	liveConfirmed   *Node
	latestConfirmed *Node
}
