package v1

import (
	"sync"

	"github.com/pkg/errors"
	eth2types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/fieldtrie"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/types"
	"github.com/prysmaticlabs/prysm/config/params"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
)

// Ensure type BeaconState below implements BeaconState interface.
var _ state.BeaconState = (*BeaconState)(nil)

func init() {
	fieldMap = make(map[types.FieldIndex]types.DataType, params.BeaconConfig().BeaconStateFieldCount)

	// Initialize the fixed sized arrays.
	fieldMap[types.BlockRoots] = types.BasicArray
	fieldMap[types.StateRoots] = types.BasicArray
	fieldMap[types.RandaoMixes] = types.BasicArray

	// Initialize the composite arrays.
	fieldMap[types.Eth1DataVotes] = types.CompositeArray
	fieldMap[types.Validators] = types.CompositeArray
	fieldMap[types.PreviousEpochAttestations] = types.CompositeArray
	fieldMap[types.CurrentEpochAttestations] = types.CompositeArray
}

// fieldMap keeps track of each field
// to its corresponding data type.
var fieldMap map[types.FieldIndex]types.DataType

// ErrNilInnerState returns when the inner state is nil and no copy set or get
// operations can be performed on state.
var ErrNilInnerState = errors.New("nil inner state")

// BeaconState defines a struct containing utilities for the Ethereum Beacon Chain state, defining
// getters and setters for its respective values and helpful functions such as HashTreeRoot().
type BeaconState struct {
	state                               *ethpb.BeaconState
	genesisTimeInternal                 uint64
	genesisValidatorsRootInternal       []byte
	slotInternal                        eth2types.Slot
	forkInternal                        *types.Fork
	latestBlockHeaderInternal           *types.BeaconBlockHeader
	blockRootsInternal                  [][]byte
	stateRootsInternal                  [][]byte
	historicalRootsInternal             [][]byte
	eth1DataInternal                    *types.Eth1Data
	eth1DataVotesInternal               []*types.Eth1Data
	eth1DepositIndexInternal            uint64
	validatorsInternal                  []*types.Validator
	balancesInternal                    []uint64
	randaoMixesInternal                 [][]byte
	slashingsInternal                   []uint64
	previousEpochAttestationsInternal   []*pendingAttestation
	currentEpochAttestationsInternal    []*pendingAttestation
	justificationBitsInternal           bitfield.Bitvector4
	previousJustifiedCheckpointInternal *types.Checkpoint
	currentJustifiedCheckpointInternal  *types.Checkpoint
	finalizedCheckpointInternal         *types.Checkpoint

	lock                  sync.RWMutex
	dirtyFields           map[types.FieldIndex]bool
	dirtyIndices          map[types.FieldIndex][]uint64
	stateFieldLeaves      map[types.FieldIndex]*fieldtrie.FieldTrie
	rebuildTrie           map[types.FieldIndex]bool
	valMapHandler         *stateutil.ValidatorMapHandler
	merkleLayers          [][][]byte
	sharedFieldReferences map[types.FieldIndex]*stateutil.Reference
}

type pendingAttestation struct {
	aggregationBits bitfield.Bitlist
	data            attestationData
	inclusionDelay  eth2types.Slot
	proposerIndex   eth2types.ValidatorIndex
}

type attestationData struct {
	slot            eth2types.Slot
	committeeIndex  eth2types.CommitteeIndex
	beaconBlockRoot []byte
	source          types.Checkpoint
	target          types.Checkpoint
}

// Field Aliases for values from the types package.
const (
	genesisTime                 = types.GenesisTime
	genesisValidatorRoot        = types.GenesisValidatorRoot
	slot                        = types.Slot
	fork                        = types.ChainFork
	latestBlockHeader           = types.LatestBlockHeader
	blockRoots                  = types.BlockRoots
	stateRoots                  = types.StateRoots
	historicalRoots             = types.HistoricalRoots
	eth1Data                    = types.LatestEth1Data
	eth1DataVotes               = types.Eth1DataVotes
	eth1DepositIndex            = types.Eth1DepositIndex
	validators                  = types.Validators
	balances                    = types.Balances
	randaoMixes                 = types.RandaoMixes
	slashings                   = types.Slashings
	previousEpochAttestations   = types.PreviousEpochAttestations
	currentEpochAttestations    = types.CurrentEpochAttestations
	justificationBits           = types.JustificationBits
	previousJustifiedCheckpoint = types.PreviousJustifiedCheckpoint
	currentJustifiedCheckpoint  = types.CurrentJustifiedCheckpoint
	finalizedCheckpoint         = types.FinalizedCheckpoint
)
