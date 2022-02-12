package protoarray

import (
	"sync"

	types "github.com/prysmaticlabs/eth2-types"
	fieldparams "github.com/prysmaticlabs/prysm/config/fieldparams"
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
	justifiedEpoch             types.Epoch                            // latest justified epoch in store.
	finalizedEpoch             types.Epoch                            // latest finalized epoch in store.
	finalizedRoot              [fieldparams.RootLength]byte           // latest finalized root in store.
	pruneThreshold             uint64                                 // do not prune tree unless threshold is reached.
	proposerBoostRoot          [fieldparams.RootLength]byte           // latest block root that was boosted after being received in a timely manner.
	previousProposerBoostRoot  [fieldparams.RootLength]byte           // previous block root that was boosted after being received in a timely manner.
	previousProposerBoostScore uint64                                 // previous proposer boosted root score.
	treeRoot                   *Node                                  // the root node of the store tree.
	nodeByRoot                 map[[fieldparams.RootLength]byte]*Node // nodes indexed by roots.
	canonicalNodes             map[[fieldparams.RootLength]byte]bool  // the canonical block nodes.
	nodesLock                  sync.RWMutex
	proposerBoostLock          sync.RWMutex
}

// Node defines the individual block which includes its block parent, ancestor and how much weight accounted for it.
// This is used as an array based stateful DAG for efficient fork choice look up.
type Node struct {
	slot           types.Slot                   // slot of the block converted to the node.
	root           [fieldparams.RootLength]byte // root of the block converted to the node.
	parent         *Node                        // parent index of this node.
	children       []*Node                      // the list of direct children of this Node
	justifiedEpoch types.Epoch                  // justifiedEpoch of this node.
	finalizedEpoch types.Epoch                  // finalizedEpoch of this node.
	balance        uint64                       // the balance that voted for this node directly
	weight         uint64                       // weight of this node: the total balance including children
	bestDescendant *Node                        // bestDescendant node of this node.
	optimistic     bool                         // whether the block has been fully validated or not
}

// Vote defines an individual validator's vote.
type Vote struct {
	currentRoot [fieldparams.RootLength]byte // current voting root.
	nextRoot    [fieldparams.RootLength]byte // next voting root.
	nextEpoch   types.Epoch                  // epoch of next voting period.
}
