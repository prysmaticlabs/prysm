package state_native

import (
	"fmt"

	"github.com/prysmaticlabs/prysm/v4/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/state/state-native/types"
	"github.com/prysmaticlabs/prysm/v4/runtime/version"
)

// Ensure type BeaconState below implements BeaconState interface.
var _ state.BeaconState = (*BeaconState)(nil)

func init() {
	fieldMap = make(map[types.FieldIndex]types.FieldInfo)
	// Initialize the fixed sized arrays.
	fieldMap[types.BlockRoots] = types.FieldInfo{ArrayType: types.BasicArray, ValueType: types.MultiValue}
	fieldMap[types.StateRoots] = types.FieldInfo{ArrayType: types.BasicArray, ValueType: types.MultiValue}
	fieldMap[types.RandaoMixes] = types.FieldInfo{ArrayType: types.BasicArray, ValueType: types.MultiValue}

	// Initialize the composite arrays.
	fieldMap[types.Eth1DataVotes] = types.FieldInfo{ArrayType: types.CompositeArray, ValueType: types.SingleValue}
	fieldMap[types.Validators] = types.FieldInfo{ArrayType: types.CompositeArray, ValueType: types.SingleValue}
	fieldMap[types.PreviousEpochAttestations] = types.FieldInfo{ArrayType: types.CompositeArray, ValueType: types.SingleValue}
	fieldMap[types.CurrentEpochAttestations] = types.FieldInfo{ArrayType: types.CompositeArray, ValueType: types.SingleValue}
	fieldMap[types.Balances] = types.FieldInfo{ArrayType: types.CompressedArray, ValueType: types.MultiValue}
}

// fieldMap keeps track of each field
// to its corresponding data type.
var fieldMap map[types.FieldIndex]types.FieldInfo

func errNotSupported(funcName string, ver int) error {
	return fmt.Errorf("%s is not supported for %s", funcName, version.String(ver))
}
