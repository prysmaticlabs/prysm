package stateutil

import "github.com/prysmaticlabs/prysm/shared/params"

// fieldMap keeps track of each field
// to its corresponding data type.
var fieldMap map[FieldIndex]dataType

// List of current data types the state supports.
const (
	basicArray dataType = iota
	compositeArray
)

// dataType signifies the data type of the field.
type dataType int

func init() {
	fieldMap = make(map[FieldIndex]dataType, params.BeaconConfig().BeaconStateFieldCount)

	// Initialize the fixed sized arrays.
	fieldMap[BlockRoots] = basicArray
	fieldMap[StateRoots] = basicArray
	fieldMap[RandaoMixes] = basicArray

	// Initialize the composite arrays.
	fieldMap[Eth1DataVotes] = compositeArray
	fieldMap[Validators] = compositeArray
	fieldMap[PreviousEpochAttestations] = compositeArray
	fieldMap[CurrentEpochAttestations] = compositeArray
}
