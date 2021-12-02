package v1

import (
	"sync"

	"github.com/pkg/errors"
	eth2types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	customtypes "github.com/prysmaticlabs/prysm/beacon-chain/state/custom-types"
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
	fieldMap[types.Balances] = types.CompressedArray
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
	genesisTime                 uint64                      `ssz-gen:"true"`
	genesisValidatorsRoot       customtypes.Byte32          `ssz-gen:"true" ssz-size:"32"`
	slot                        eth2types.Slot              `ssz-gen:"true"`
	fork                        *ethpb.Fork                 `ssz-gen:"true"`
	latestBlockHeader           *ethpb.BeaconBlockHeader    `ssz-gen:"true"`
	blockRoots                  *customtypes.StateRoots     `ssz-gen:"true" ssz-size:"8192,32"`
	stateRoots                  *customtypes.StateRoots     `ssz-gen:"true" ssz-size:"8192,32"`
	historicalRoots             customtypes.HistoricalRoots `ssz-gen:"true" ssz-size:"?,32" ssz-max:"16777216"`
	eth1Data                    *ethpb.Eth1Data             `ssz-gen:"true"`
	eth1DataVotes               []*ethpb.Eth1Data           `ssz-gen:"true" ssz-max:"2048"`
	eth1DepositIndex            uint64                      `ssz-gen:"true"`
	validators                  []*ethpb.Validator          `ssz-gen:"true" ssz-max:"1099511627776"`
	balances                    []uint64                    `ssz-gen:"true" ssz-max:"1099511627776"`
	randaoMixes                 *customtypes.RandaoMixes    `ssz-gen:"true" ssz-size:"65536,32"`
	slashings                   []uint64                    `ssz-gen:"true" ssz-size:"8192"`
	previousEpochAttestations   []*ethpb.PendingAttestation `ssz-gen:"true" ssz-max:"1099511627776"`
	currentEpochAttestations    []*ethpb.PendingAttestation `ssz-gen:"true" ssz-max:"1099511627776"`
	justificationBits           bitfield.Bitvector4         `ssz-gen:"true" ssz-size:"1"`
	previousJustifiedCheckpoint *ethpb.Checkpoint           `ssz-gen:"true"`
	currentJustifiedCheckpoint  *ethpb.Checkpoint           `ssz-gen:"true"`
	finalizedCheckpoint         *ethpb.Checkpoint           `ssz-gen:"true"`

	lock                  sync.RWMutex
	dirtyFields           map[types.FieldIndex]bool
	dirtyIndices          map[types.FieldIndex][]uint64
	stateFieldLeaves      map[types.FieldIndex]*fieldtrie.FieldTrie
	rebuildTrie           map[types.FieldIndex]bool
	valMapHandler         *stateutil.ValidatorMapHandler
	merkleLayers          [][][]byte
	sharedFieldReferences map[types.FieldIndex]*stateutil.Reference
}

// Field Aliases for values from the types package.
const (
	genesisTime                 = types.GenesisTime
	genesisValidatorRoot        = types.GenesisValidatorRoot
	slot                        = types.Slot
	fork                        = types.Fork
	latestBlockHeader           = types.LatestBlockHeader
	blockRoots                  = types.BlockRoots
	stateRoots                  = types.StateRoots
	historicalRoots             = types.HistoricalRoots
	eth1Data                    = types.Eth1Data
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
