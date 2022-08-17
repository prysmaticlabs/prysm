package protoarray

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
	pruneThreshold                uint64                                  // do not prune tree unless threshold is reached.
	justifiedCheckpoint           *forkchoicetypes.Checkpoint             // latest justified checkpoint in store.
	bestJustifiedCheckpoint       *forkchoicetypes.Checkpoint             // best justified checkpoint in store.
	unrealizedJustifiedCheckpoint *forkchoicetypes.Checkpoint             // best justified checkpoint in store.
	unrealizedFinalizedCheckpoint *forkchoicetypes.Checkpoint             // best justified checkpoint in store.
	prevJustifiedCheckpoint       *forkchoicetypes.Checkpoint             // previous justified checkpoint in store.
	finalizedCheckpoint           *forkchoicetypes.Checkpoint             // latest finalized checkpoint in store.
	proposerBoostRoot             [fieldparams.RootLength]byte            // latest block root that was boosted after being received in a timely manner.
	previousProposerBoostRoot     [fieldparams.RootLength]byte            // previous block root that was boosted after being received in a timely manner.
	previousProposerBoostScore    uint64                                  // previous proposer boosted root score.
	nodes                         []*Node                                 // list of block nodes, each node is a representation of one block.
	nodesIndices                  map[[fieldparams.RootLength]byte]uint64 // the root of block node and the nodes index in the list.
	canonicalNodes                map[[fieldparams.RootLength]byte]bool   // the canonical block nodes.
	payloadIndices                map[[fieldparams.RootLength]byte]uint64 // the payload hash of block node and the index in the list
	slashedIndices                map[types.ValidatorIndex]bool           // The list of equivocating validators
	originRoot                    [fieldparams.RootLength]byte            // The genesis block root
	lastHeadRoot                  [fieldparams.RootLength]byte            // The last cached head block root
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
	parent                   uint64                       // parent index of this node.
	justifiedEpoch           types.Epoch                  // justifiedEpoch of this node.
	unrealizedJustifiedEpoch types.Epoch                  // the epoch that would be justified if the block would be advanced to the next epoch.
	finalizedEpoch           types.Epoch                  // finalizedEpoch of this node.
	unrealizedFinalizedEpoch types.Epoch                  // the epoch that would be finalized if the block would be advanced to the next epoch.
	weight                   uint64                       // weight of this node.
	bestChild                uint64                       // bestChild index of this node.
	bestDescendant           uint64                       // bestDescendant of this node.
	status                   status                       // optimistic status of this node
}

// enum used as optimistic status of a node
type status uint8

const (
	syncing status = iota // the node is optimistic
	valid                 //fully validated node
	invalid               // invalid execution payload
)

// Vote defines an individual validator's vote.
type Vote struct {
	currentRoot [fieldparams.RootLength]byte // current voting root.
	nextRoot    [fieldparams.RootLength]byte // next voting root.
	nextEpoch   types.Epoch                  // epoch of next voting period.
}

// NonExistentNode defines an unknown node which is used for the array based stateful DAG.
const NonExistentNode = ^uint64(0)
