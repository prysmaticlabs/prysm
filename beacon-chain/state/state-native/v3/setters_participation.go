package v3

import (
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"
)

// SetPreviousParticipationBits for the beacon state. Updates the entire
// list to a new value by overwriting the previous one.
func (b *BeaconState) SetPreviousParticipationBits(val []byte) error {
	if !b.hasInnerState() {
		return ErrNilInnerState
	}
	b.lock.Lock()
	defer b.lock.Unlock()

	b.sharedFieldReferences[previousEpochParticipationBits].MinusRef()
	b.sharedFieldReferences[previousEpochParticipationBits] = stateutil.NewRef(1)

	b.state.PreviousEpochParticipation = val
	b.markFieldAsDirty(previousEpochParticipationBits)
	b.rebuildTrie[previousEpochParticipationBits] = true
	return nil
}

// SetCurrentParticipationBits for the beacon state. Updates the entire
// list to a new value by overwriting the previous one.
func (b *BeaconState) SetCurrentParticipationBits(val []byte) error {
	if !b.hasInnerState() {
		return ErrNilInnerState
	}
	b.lock.Lock()
	defer b.lock.Unlock()

	b.sharedFieldReferences[currentEpochParticipationBits].MinusRef()
	b.sharedFieldReferences[currentEpochParticipationBits] = stateutil.NewRef(1)

	b.state.CurrentEpochParticipation = val
	b.markFieldAsDirty(currentEpochParticipationBits)
	b.rebuildTrie[currentEpochParticipationBits] = true
	return nil
}

// AppendCurrentParticipationBits for the beacon state. Appends the new value
// to the the end of list.
func (b *BeaconState) AppendCurrentParticipationBits(val byte) error {
	if !b.hasInnerState() {
		return ErrNilInnerState
	}
	b.lock.Lock()
	defer b.lock.Unlock()

	participation := b.state.CurrentEpochParticipation
	if b.sharedFieldReferences[currentEpochParticipationBits].Refs() > 1 {
		// Copy elements in underlying array by reference.
		participation = make([]byte, len(b.state.CurrentEpochParticipation))
		copy(participation, b.state.CurrentEpochParticipation)
		b.sharedFieldReferences[currentEpochParticipationBits].MinusRef()
		b.sharedFieldReferences[currentEpochParticipationBits] = stateutil.NewRef(1)
	}

	b.state.CurrentEpochParticipation = append(participation, val)
	b.markFieldAsDirty(currentEpochParticipationBits)
	b.addDirtyIndices(currentEpochParticipationBits, []uint64{uint64(len(b.state.CurrentEpochParticipation) - 1)})
	return nil
}

// AppendPreviousParticipationBits for the beacon state. Appends the new value
// to the the end of list.
func (b *BeaconState) AppendPreviousParticipationBits(val byte) error {
	if !b.hasInnerState() {
		return ErrNilInnerState
	}
	b.lock.Lock()
	defer b.lock.Unlock()

	bits := b.state.PreviousEpochParticipation
	if b.sharedFieldReferences[previousEpochParticipationBits].Refs() > 1 {
		bits = make([]byte, len(b.state.PreviousEpochParticipation))
		copy(bits, b.state.PreviousEpochParticipation)
		b.sharedFieldReferences[previousEpochParticipationBits].MinusRef()
		b.sharedFieldReferences[previousEpochParticipationBits] = stateutil.NewRef(1)
	}

	b.state.PreviousEpochParticipation = append(bits, val)
	b.markFieldAsDirty(previousEpochParticipationBits)
	b.addDirtyIndices(previousEpochParticipationBits, []uint64{uint64(len(b.state.PreviousEpochParticipation) - 1)})

	return nil
}
