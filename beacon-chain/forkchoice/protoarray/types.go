package protoarray

import "sync"

// ForkChoice defines the overall fork choice store which includes all block nodes, validator's latest votes and balances.
type ForkChoice struct {
	store    *Store
	votes    []Vote   // tracks individual validator's last vote.
	balances []uint64 // tracks individual validator's last justified balances.
}

// Store defines the fork choice store which includes block nodes and the last view of checkpoint information.
type Store struct {
	PruneThreshold  uint64              // do not prune tree unless threshold is reached.
	JustifiedEpoch  uint64              // latest justified epoch in store.
	FinalizedEpoch  uint64              // latest finalized epoch in store.
	finalizedRoot   [32]byte            // latest finalized root in store.
	Nodes           []*Node             // list of block nodes, each node is a representation of one block.
	NodeIndices     map[[32]byte]uint64 // the root of block node and the Nodes index in the list.
	nodeIndicesLock sync.RWMutex
}

// Node defines the individual block which includes its block parent, ancestor and how much weight accounted for it.
// This is used as an array based stateful DAG for efficient fork choice look up.
type Node struct {
	Slot           uint64   // Slot of the block converted to the node.
	Root           [32]byte // Root of the block converted to the node.
	Parent         uint64   // Parent index of this node.
	JustifiedEpoch uint64   // JustifiedEpoch of this node.
	FinalizedEpoch uint64   // FinalizedEpoch of this node.
	Weight         uint64   // Weight of this node.
	BestChild      uint64   // BestChild index of this node.
	BestDescendant uint64   // BestDescendant of this node.
	Graffiti       [32]byte // Graffiti of the block node.
}

// Vote defines an individual validator's vote.
type Vote struct {
	currentRoot [32]byte // current voting root.
	nextRoot    [32]byte // next voting root.
	nextEpoch   uint64   // epoch of next voting period.
}

// NonExistentNode defines an unknown node which is used for the array based stateful DAG.
const NonExistentNode = ^uint64(0)
