package state_native

import (
	"fmt"

	"github.com/prysmaticlabs/prysm/v4/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/state/state-native/types"
	"github.com/prysmaticlabs/prysm/v4/runtime/version"
)

// Ensure type BeaconState below implements BeaconState interface.
var _ state.BeaconState = (*BeaconState)(nil)

// initialization for tests
func init() {
	FieldMap = make(map[types.FieldIndex]types.DataType)
	// Initialize the fixed sized arrays.
	FieldMap[types.BlockRoots] = types.BasicArray
	FieldMap[types.StateRoots] = types.BasicArray
	FieldMap[types.RandaoMixes] = types.BasicArray

	// Initialize the composite arrays.
	FieldMap[types.Eth1DataVotes] = types.CompositeArray
	FieldMap[types.Validators] = types.CompositeArray
	FieldMap[types.PreviousEpochAttestations] = types.CompositeArray
	FieldMap[types.CurrentEpochAttestations] = types.CompositeArray
	FieldMap[types.Balances] = types.CompressedArray
}

// FieldMap keeps track of each field
// to its corresponding data type.
var FieldMap map[types.FieldIndex]types.DataType

func errNotSupported(funcName string, ver int) error {
	return fmt.Errorf("%s is not supported for %s", funcName, version.String(ver))
}
