package v1

import (
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/types"
	"github.com/prysmaticlabs/prysm/config/params"
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
