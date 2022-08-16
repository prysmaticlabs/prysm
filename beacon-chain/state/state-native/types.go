package state_native

import (
	"fmt"

	"github.com/prysmaticlabs/prysm/v3/beacon-chain/state"
	nativetypes "github.com/prysmaticlabs/prysm/v3/beacon-chain/state/state-native/types"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/state/types"
	"github.com/prysmaticlabs/prysm/v3/runtime/version"
)

// Ensure type BeaconState below implements BeaconState interface.
var _ state.BeaconState = (*BeaconState)(nil)

func init() {
	fieldMap = make(map[nativetypes.FieldIndex]types.DataType)
	// Initialize the fixed sized arrays.
	fieldMap[nativetypes.BlockRoots] = types.BasicArray
	fieldMap[nativetypes.StateRoots] = types.BasicArray
	fieldMap[nativetypes.RandaoMixes] = types.BasicArray

	// Initialize the composite arrays.
	fieldMap[nativetypes.Eth1DataVotes] = types.CompositeArray
	fieldMap[nativetypes.Validators] = types.CompositeArray
	fieldMap[nativetypes.PreviousEpochAttestations] = types.CompositeArray
	fieldMap[nativetypes.CurrentEpochAttestations] = types.CompositeArray
	fieldMap[nativetypes.Balances] = types.CompressedArray
}

// fieldMap keeps track of each field
// to its corresponding data type.
var fieldMap map[nativetypes.FieldIndex]types.DataType

func errNotSupported(funcName string, ver int) error {
	return fmt.Errorf("%s is not supported for %s", funcName, version.String(ver))
}
