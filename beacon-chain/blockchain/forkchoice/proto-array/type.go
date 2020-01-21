package protoarray

import "sync"

// ForkChoice defines the overall fork choice store which includes block nodes, validator's latest votes and balances.
type ForkChoice struct {
	store    *Store
	votes    []Vote   // tracks individual validator's latest vote.
	balances []uint64 // tracks individual validator's effective balances.
}

// Store defines the fork choice object which includes block nodes and the latest head view of checkpoint information.
type Store struct {
	pruneThreshold     uint64              // do not prune tree unless threshold is reached.
	treeFilterRequired bool                // tree filtering as specified in latest eth2 spec.
	justifiedEpoch     uint64              // latest justified epoch in store.
	finalizedEpoch     uint64              // latest finalized epoch in store.
	finalizedRoot      [32]byte            // latest finalized root in store.
	nodes              []*Node             // list of block nodes, each node is a representation of one block.
	nodeIndices        map[[32]byte]uint64 // the root of block node and the nodes index in the list.
	nodeIndicesLock    sync.RWMutex
}

// Node defines the individual block which includes its parent, ancestor and how much weight accounted for it.
type Node struct {
	slot           uint64   // slot of the block converted to the node.
	root           [32]byte // root of the block converted to the node.
	parent         uint64   // the parent index of this node.
	justifiedEpoch uint64   // justified epoch of this node.
	finalizedEpoch uint64   // finalized epoch of this node.
	weight         uint64   // weight of this node.
	bestChild      uint64   // best child index of this node.
	bestDescendant uint64   // head index of this node.
}

// Vote defines individual validator's vote.
type Vote struct {
	currentRoot [32]byte
	nextRoot    [32]byte
	nextEpoch   uint64
}

const nonExistentNode = ^uint64(0)
