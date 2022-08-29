package doublylinkedtree

import (
	"sync"

	forkchoicetypes "github.com/prysmaticlabs/prysm/v3/beacon-chain/forkchoice/types"
	fieldparams "github.com/prysmaticlabs/prysm/v3/config/fieldparams"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
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
	justifiedCheckpoint           *forkchoicetypes.Checkpoint            // latest justified epoch in store.
	bestJustifiedCheckpoint       *forkchoicetypes.Checkpoint            // best justified checkpoint in store.
	unrealizedJustifiedCheckpoint *forkchoicetypes.Checkpoint            // best unrealized justified checkpoint in store.
	unrealizedFinalizedCheckpoint *forkchoicetypes.Checkpoint            // best unrealized finalized checkpoint in store.
	prevJustifiedCheckpoint       *forkchoicetypes.Checkpoint            // previous justified checkpoint in store.
	finalizedCheckpoint           *forkchoicetypes.Checkpoint            // latest finalized epoch in store.
	pruneThreshold                uint64                                 // do not prune tree unless threshold is reached.
	proposerBoostRoot             [fieldparams.RootLength]byte           // latest block root that was boosted after being received in a timely manner.
	previousProposerBoostRoot     [fieldparams.RootLength]byte           // previous block root that was boosted after being received in a timely manner.
	previousProposerBoostScore    uint64                                 // previous proposer boosted root score.
	treeRootNode                  *Node                                  // the root node of the store tree.
	headNode                      *Node                                  // last head Node
	nodeByRoot                    map[[fieldparams.RootLength]byte]*Node // nodes indexed by roots.
	nodeByPayload                 map[[fieldparams.RootLength]byte]*Node // nodes indexed by payload Hash
	slashedIndices                map[types.ValidatorIndex]bool          // the list of equivocating validator indices
	originRoot                    [fieldparams.RootLength]byte           // The genesis block root
	nodesLock                     sync.RWMutex
	proposerBoostLock             sync.RWMutex
	checkpointsLock               sync.RWMutex
	genesisTime                   uint64
	highestReceivedSlot           types.Slot                            // The highest received slot in the chain.
	receivedBlocksLastEpoch       [fieldparams.SlotsPerEpoch]types.Slot // Using `highestReceivedSlot`. The slot of blocks received in the last epoch.
	allTipsAreInvalid             bool                                  // tracks if all tips are not viable for head
}

// Node defines the individual block which includes its block parent, ancestor and how much weight accounted for it.
// This is used as an array based stateful DAG for efficient fork choice look up.
type Node struct {
	slot                     types.Slot                   // slot of the block converted to the node.
	root                     [fieldparams.RootLength]byte // root of the block converted to the node.
	payloadHash              [fieldparams.RootLength]byte // payloadHash of the block converted to the node.
	parent                   *Node                        // parent index of this node.
	children                 []*Node                      // the list of direct children of this Node
	justifiedEpoch           types.Epoch                  // justifiedEpoch of this node.
	unrealizedJustifiedEpoch types.Epoch                  // the epoch that would be justified if the block would be advanced to the next epoch.
	finalizedEpoch           types.Epoch                  // finalizedEpoch of this node.
	unrealizedFinalizedEpoch types.Epoch                  // the epoch that would be finalized if the block would be advanced to the next epoch.
	balance                  uint64                       // the balance that voted for this node directly
	weight                   uint64                       // weight of this node: the total balance including children
	bestDescendant           *Node                        // bestDescendant node of this node.
	optimistic               bool                         // whether the block has been fully validated or not
	timestamp                uint64                       // The timestamp when the node was inserted.
}

// Vote defines an individual validator's vote.
type Vote struct {
	currentRoot [fieldparams.RootLength]byte // current voting root.
	nextRoot    [fieldparams.RootLength]byte // next voting root.
	nextEpoch   types.Epoch                  // epoch of next voting period.
}
