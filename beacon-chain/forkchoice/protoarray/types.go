package protoarray

import (
	"sync"

	types "github.com/prysmaticlabs/eth2-types"
	fieldparams "github.com/prysmaticlabs/prysm/config/fieldparams"
)

// ForkChoice defines the overall fork choice store which includes all block nodes, validator's latest votes and balances.
type ForkChoice struct {
	store      *Store
	votes      []Vote // tracks individual validator's last vote.
	votesLock  sync.RWMutex
	balances   []uint64 // tracks individual validator's last justified balances.
	syncedTips *optimisticStore
}

// Store defines the fork choice store which includes block nodes and the last view of checkpoint information.
type Store struct {
	pruneThreshold             uint64                                  // do not prune tree unless threshold is reached.
	justifiedEpoch             types.Epoch                             // latest justified epoch in store.
	finalizedEpoch             types.Epoch                             // latest finalized epoch in store.
	finalizedRoot              [fieldparams.RootLength]byte            // latest finalized root in store.
	proposerBoostRoot          [fieldparams.RootLength]byte            // latest block root that was boosted after being received in a timely manner.
	previousProposerBoostRoot  [fieldparams.RootLength]byte            // previous block root that was boosted after being received in a timely manner.
	previousProposerBoostScore uint64                                  // previous proposer boosted root score.
	nodes                      []*Node                                 // list of block nodes, each node is a representation of one block.
	nodesIndices               map[[fieldparams.RootLength]byte]uint64 // the root of block node and the nodes index in the list.
	canonicalNodes             map[[fieldparams.RootLength]byte]bool   // the canonical block nodes.
	nodesLock                  sync.RWMutex
	proposerBoostLock          sync.RWMutex
}

// Node defines the individual block which includes its block parent, ancestor and how much weight accounted for it.
// This is used as an array based stateful DAG for efficient fork choice look up.
type Node struct {
	slot           types.Slot                   // slot of the block converted to the node.
	root           [fieldparams.RootLength]byte // root of the block converted to the node.
	payloadHash    [fieldparams.RootLength]byte // payloadHash of the block converted to the node.
	parent         uint64                       // parent index of this node.
	justifiedEpoch types.Epoch                  // justifiedEpoch of this node.
	finalizedEpoch types.Epoch                  // finalizedEpoch of this node.
	weight         uint64                       // weight of this node.
	bestChild      uint64                       // bestChild index of this node.
	bestDescendant uint64                       // bestDescendant of this node.
	graffiti       [fieldparams.RootLength]byte // graffiti of the block node.
}

// optimisticStore defines a structure that tracks the tips of the fully
// validated blocks tree.
type optimisticStore struct {
	validatedTips map[[32]byte]types.Slot
	sync.RWMutex
}

// Vote defines an individual validator's vote.
type Vote struct {
	currentRoot [fieldparams.RootLength]byte // current voting root.
	nextRoot    [fieldparams.RootLength]byte // next voting root.
	nextEpoch   types.Epoch                  // epoch of next voting period.
}

// NonExistentNode defines an unknown node which is used for the array based stateful DAG.
const NonExistentNode = ^uint64(0)
