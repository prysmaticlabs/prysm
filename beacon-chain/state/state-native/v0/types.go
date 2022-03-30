package v0

import (
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	v0types "github.com/prysmaticlabs/prysm/beacon-chain/state/state-native/v0/types"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/types"
	"github.com/prysmaticlabs/prysm/config/params"
)

// Ensure type BeaconState below implements BeaconState interface.
var _ state.BeaconState = (*BeaconState)(nil)

func init() {
	fieldMap = make(map[v0types.FieldIndex]types.DataType, params.BeaconConfig().BeaconStateFieldCount)
	// Initialize the fixed sized arrays.
	fieldMap[v0types.BlockRoots] = types.BasicArray
	fieldMap[v0types.StateRoots] = types.BasicArray
	fieldMap[v0types.RandaoMixes] = types.BasicArray

	// Initialize the composite arrays.
	fieldMap[v0types.Eth1DataVotes] = types.CompositeArray
	fieldMap[v0types.Validators] = types.CompositeArray
	fieldMap[v0types.PreviousEpochAttestations] = types.CompositeArray
	fieldMap[v0types.CurrentEpochAttestations] = types.CompositeArray
	fieldMap[v0types.Balances] = types.CompressedArray
}

// fieldMap keeps track of each field
// to its corresponding data type.
var fieldMap map[v0types.FieldIndex]types.DataType
