package v3

import (
	stateTypes "github.com/prysmaticlabs/prysm/beacon-chain/state/types"
)

func (b *BeaconState) markFieldAsDirty(field stateTypes.FieldIndex) {
	b.dirtyFields[field] = true
}
