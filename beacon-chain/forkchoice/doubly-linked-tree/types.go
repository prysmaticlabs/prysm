package doublylinkedtree

import (
	"sync"

	"github.com/prysmaticlabs/prysm/v3/beacon-chain/forkchoice"
	forkchoicetypes "github.com/prysmaticlabs/prysm/v3/beacon-chain/forkchoice/types"
	fieldparams "github.com/prysmaticlabs/prysm/v3/config/fieldparams"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
)

// ForkChoice defines the overall fork choice store which includes all block nodes, validator's latest votes and balances.
type ForkChoice struct {
	store          *Store
	balancesByRoot forkchoice.BalancesByRooter // handler to obtain balances for the state with a given root
	votes          []Vote                      // tracks individual validator's last vote.
	balances       []uint64                    // tracks individual validator's last justified balances.
	votesLock      sync.RWMutex
}

// Store defines the fork choice store which includes block nodes and the last view of checkpoint information.
type Store struct {
	headNode                      *Node                                  // last head Node
	unrealizedJustifiedCheckpoint *forkchoicetypes.Checkpoint            // best unrealized justified checkpoint in store.
	justifiedCheckpoint           *forkchoicetypes.Checkpoint            // latest justified epoch in store.
	unrealizedFinalizedCheckpoint *forkchoicetypes.Checkpoint            // best unrealized finalized checkpoint in store.
	prevJustifiedCheckpoint       *forkchoicetypes.Checkpoint            // previous justified checkpoint in store.
	finalizedCheckpoint           *forkchoicetypes.Checkpoint            // latest finalized epoch in store.
	nodeByRoot                    map[[fieldparams.RootLength]byte]*Node // nodes indexed by roots.
	highestReceivedNode           *Node
	slashedIndices                map[types.ValidatorIndex]bool          // the list of equivocating validator indices
	nodeByPayload                 map[[fieldparams.RootLength]byte]*Node // nodes indexed by payload Hash
	bestJustifiedCheckpoint       *forkchoicetypes.Checkpoint            // best justified checkpoint in store.
	treeRootNode                  *Node                                  // the root node of the store tree.
	receivedBlocksLastEpoch       [fieldparams.SlotsPerEpoch]types.Slot  // Using `highestReceivedSlot`. The slot of blocks received in the last epoch.
	committeeBalance              uint64                                 // tracks the total active validator balance divided by slots per epoch. Requires a lock on nodes to read/write
	previousProposerBoostScore    uint64                                 // previous proposer boosted root score.
	genesisTime                   uint64                                 // The highest slot node.
	proposerBoostLock             sync.RWMutex
	checkpointsLock               sync.RWMutex
	nodesLock                     sync.RWMutex
	originRoot                    [fieldparams.RootLength]byte // The genesis block root
	previousProposerBoostRoot     [fieldparams.RootLength]byte // previous block root that was boosted after being received in a timely manner.
	proposerBoostRoot             [fieldparams.RootLength]byte // latest block root that was boosted after being received in a timely manner.
	allTipsAreInvalid             bool                         // tracks if all tips are not viable for head
}

// Node defines the individual block which includes its block parent, ancestor and how much weight accounted for it.
// This is used as an array based stateful DAG for efficient fork choice look up.
type Node struct {
	parent                   *Node                        // parent index of this node.
	bestDescendant           *Node                        // bestDescendant node of this node.
	children                 []*Node                      // the list of direct children of this Node
	unrealizedJustifiedEpoch types.Epoch                  // the epoch that would be justified if the block would be advanced to the next epoch.
	justifiedEpoch           types.Epoch                  // justifiedEpoch of this node.
	slot                     types.Slot                   // slot of the block converted to the node.
	finalizedEpoch           types.Epoch                  // finalizedEpoch of this node.
	unrealizedFinalizedEpoch types.Epoch                  // the epoch that would be finalized if the block would be advanced to the next epoch.
	balance                  uint64                       // the balance that voted for this node directly
	weight                   uint64                       // weight of this node: the total balance including children
	timestamp                uint64                       // The timestamp when the node was inserted.
	payloadHash              [fieldparams.RootLength]byte // payloadHash of the block converted to the node.
	root                     [fieldparams.RootLength]byte // root of the block converted to the node.
	optimistic               bool                         // whether the block has been fully validated or not
}

// Vote defines an individual validator's vote.
type Vote struct {
	currentRoot [fieldparams.RootLength]byte // current voting root.
	nextRoot    [fieldparams.RootLength]byte // next voting root.
	nextEpoch   types.Epoch                  // epoch of next voting period.
}
