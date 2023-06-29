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
	FieldMap = make(map[types.FieldIndex]types.FieldInfo)
	// Initialize the fixed sized arrays.
	FieldMap[types.BlockRoots] = types.FieldInfo{ArrayType: types.BasicArray, ValueType: types.SingleValue}
	FieldMap[types.StateRoots] = types.FieldInfo{ArrayType: types.BasicArray, ValueType: types.SingleValue}
	FieldMap[types.RandaoMixes] = types.FieldInfo{ArrayType: types.BasicArray, ValueType: types.SingleValue}

	// Initialize the composite arrays.
	FieldMap[types.Eth1DataVotes] = types.FieldInfo{ArrayType: types.CompositeArray, ValueType: types.SingleValue}
	FieldMap[types.Validators] = types.FieldInfo{ArrayType: types.CompositeArray, ValueType: types.SingleValue}
	FieldMap[types.PreviousEpochAttestations] = types.FieldInfo{ArrayType: types.CompositeArray, ValueType: types.SingleValue}
	FieldMap[types.CurrentEpochAttestations] = types.FieldInfo{ArrayType: types.CompositeArray, ValueType: types.SingleValue}
	FieldMap[types.Balances] = types.FieldInfo{ArrayType: types.CompressedArray, ValueType: types.SingleValue}
}

// FieldMap keeps track of each field
// to its corresponding data type.
var FieldMap map[types.FieldIndex]types.FieldInfo

func errNotSupported(funcName string, ver int) error {
	return fmt.Errorf("%s is not supported for %s", funcName, version.String(ver))
}
