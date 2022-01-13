package protoarray

import (
	"sync"

	types "github.com/prysmaticlabs/eth2-types"
)

// ForkChoice defines the overall fork choice store which includes all block nodes, validator's latest votes and balances.
type ForkChoice struct {
	store     *Store
	votes     []Vote // tracks individual validator's last vote.
	votesLock sync.RWMutex
	balances  []uint64 // tracks individual validator's last justified balances.
}

// Store defines the fork choice store which includes block nodes and the last view of checkpoint information.
type Store struct {
	pruneThreshold             uint64              // do not prune tree unless threshold is reached.
	justifiedEpoch             types.Epoch         // latest justified epoch in store.
	finalizedEpoch             types.Epoch         // latest finalized epoch in store.
	finalizedRoot              [32]byte            // latest finalized root in store.
	proposerBoostRoot          [32]byte            // latest block root that was boosted after being received in a timely manner.
	proposerBoostScore         uint64              // proposer boost score for the current boosted root.
	previousProposerBoostRoot  [32]byte            // previous block root that was boosted after being received in a timely manner.
	previousProposerBoostScore uint64              // previous proposer boosted root score.
	nodes                      []*Node             // list of block nodes, each node is a representation of one block.
	nodesIndices               map[[32]byte]uint64 // the root of block node and the nodes index in the list.
	canonicalNodes             map[[32]byte]bool   // the canonical block nodes.
	nodesLock                  sync.RWMutex
	proposerBoostLock          sync.RWMutex
}

// Node defines the individual block which includes its block parent, ancestor and how much weight accounted for it.
// This is used as an array based stateful DAG for efficient fork choice look up.
type Node struct {
	slot           types.Slot  // slot of the block converted to the node.
	root           [32]byte    // root of the block converted to the node.
	parent         uint64      // parent index of this node.
	justifiedEpoch types.Epoch // justifiedEpoch of this node.
	finalizedEpoch types.Epoch // finalizedEpoch of this node.
	weight         uint64      // weight of this node.
	bestChild      uint64      // bestChild index of this node.
	bestDescendant uint64      // bestDescendant of this node.
	graffiti       [32]byte    // graffiti of the block node.
}

// Vote defines an individual validator's vote.
type Vote struct {
	currentRoot [32]byte    // current voting root.
	nextRoot    [32]byte    // next voting root.
	nextEpoch   types.Epoch // epoch of next voting period.
}

// NonExistentNode defines an unknown node which is used for the array based stateful DAG.
const NonExistentNode = ^uint64(0)
