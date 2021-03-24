package stateV0

import (
	"sync"

	"github.com/pkg/errors"
	iface "github.com/prysmaticlabs/prysm/beacon-chain/state/interface"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// Ensure type BeaconState below implements BeaconState interface.
var _ iface.BeaconState = (*BeaconState)(nil)

func init() {
	fieldMap = make(map[fieldIndex]dataType, params.BeaconConfig().BeaconStateFieldCount)

	// Initialize the fixed sized arrays.
	fieldMap[blockRoots] = basicArray
	fieldMap[stateRoots] = basicArray
	fieldMap[randaoMixes] = basicArray

	// Initialize the composite arrays.
	fieldMap[eth1DataVotes] = compositeArray
	fieldMap[validators] = compositeArray
	fieldMap[previousEpochAttestations] = compositeArray
	fieldMap[currentEpochAttestations] = compositeArray
}

type fieldIndex int

// dataType signifies the data type of the field.
type dataType int

// Below we define a set of useful enum values for the field
// indices of the beacon state. For example, genesisTime is the
// 0th field of the beacon state. This is helpful when we are
// updating the Merkle branches up the trie representation
// of the beacon state.
const (
	genesisTime fieldIndex = iota
	genesisValidatorRoot
	slot
	fork
	latestBlockHeader
	blockRoots
	stateRoots
	historicalRoots
	eth1Data
	eth1DataVotes
	eth1DepositIndex
	validators
	balances
	randaoMixes
	slashings
	previousEpochAttestations
	currentEpochAttestations
	justificationBits
	previousJustifiedCheckpoint
	currentJustifiedCheckpoint
	finalizedCheckpoint
)

// List of current data types the state supports.
const (
	basicArray dataType = iota
	compositeArray
)

// fieldMap keeps track of each field
// to its corresponding data type.
var fieldMap map[fieldIndex]dataType

// ErrNilInnerState returns when the inner state is nil and no copy set or get
// operations can be performed on state.
var ErrNilInnerState = errors.New("nil inner state")

// BeaconState defines a struct containing utilities for the eth2 chain state, defining
// getters and setters for its respective values and helpful functions such as HashTreeRoot().
type BeaconState struct {
	state                 *pbp2p.BeaconState
	lock                  sync.RWMutex
	dirtyFields           map[fieldIndex]interface{}
	dirtyIndices          map[fieldIndex][]uint64
	stateFieldLeaves      map[fieldIndex]*FieldTrie
	rebuildTrie           map[fieldIndex]bool
	valMapHandler         *stateutil.ValidatorMapHandler
	merkleLayers          [][][]byte
	sharedFieldReferences map[fieldIndex]*stateutil.Reference
}

// String returns the name of the field index.
func (f fieldIndex) String() string {
	switch f {
	case genesisTime:
		return "genesisTime"
	case genesisValidatorRoot:
		return "genesisValidatorRoot"
	case slot:
		return "slot"
	case fork:
		return "fork"
	case latestBlockHeader:
		return "latestBlockHeader"
	case blockRoots:
		return "blockRoots"
	case stateRoots:
		return "stateRoots"
	case historicalRoots:
		return "historicalRoots"
	case eth1Data:
		return "eth1Data"
	case eth1DataVotes:
		return "eth1DataVotes"
	case eth1DepositIndex:
		return "eth1DepositIndex"
	case validators:
		return "validators"
	case balances:
		return "balances"
	case randaoMixes:
		return "randaoMixes"
	case slashings:
		return "slashings"
	case previousEpochAttestations:
		return "previousEpochAttestations"
	case currentEpochAttestations:
		return "currentEpochAttestations"
	case justificationBits:
		return "justificationBits"
	case previousJustifiedCheckpoint:
		return "previousJustifiedCheckpoint"
	case currentJustifiedCheckpoint:
		return "currentJustifiedCheckpoint"
	case finalizedCheckpoint:
		return "finalizedCheckpoint"
	default:
		return ""
	}
}
