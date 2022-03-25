package v3

import (
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"
)

// SetPreviousParticipationBits for the beacon state. Updates the entire
// list to a new value by overwriting the previous one.
func (b *BeaconState) SetPreviousParticipationBits(val []byte) error {
	b.lock.Lock()
	defer b.lock.Unlock()

	b.sharedFieldReferences[previousEpochParticipationBits].MinusRef()
	b.sharedFieldReferences[previousEpochParticipationBits] = stateutil.NewRef(1)

	b.previousEpochParticipation = val
	b.markFieldAsDirty(previousEpochParticipationBits)
	b.rebuildTrie[previousEpochParticipationBits] = true
	return nil
}

// SetCurrentParticipationBits for the beacon state. Updates the entire
// list to a new value by overwriting the previous one.
func (b *BeaconState) SetCurrentParticipationBits(val []byte) error {
	b.lock.Lock()
	defer b.lock.Unlock()

	b.sharedFieldReferences[currentEpochParticipationBits].MinusRef()
	b.sharedFieldReferences[currentEpochParticipationBits] = stateutil.NewRef(1)

	b.currentEpochParticipation = val
	b.markFieldAsDirty(currentEpochParticipationBits)
	b.rebuildTrie[currentEpochParticipationBits] = true
	return nil
}

// AppendCurrentParticipationBits for the beacon state. Appends the new value
// to the the end of list.
func (b *BeaconState) AppendCurrentParticipationBits(val byte) error {
	b.lock.Lock()
	defer b.lock.Unlock()

	participation := b.currentEpochParticipation
	if b.sharedFieldReferences[currentEpochParticipationBits].Refs() > 1 {
		// Copy elements in underlying array by reference.
		participation = make([]byte, len(b.currentEpochParticipation))
		copy(participation, b.currentEpochParticipation)
		b.sharedFieldReferences[currentEpochParticipationBits].MinusRef()
		b.sharedFieldReferences[currentEpochParticipationBits] = stateutil.NewRef(1)
	}

	b.currentEpochParticipation = append(participation, val)
	b.markFieldAsDirty(currentEpochParticipationBits)
	b.addDirtyIndices(currentEpochParticipationBits, []uint64{uint64(len(b.currentEpochParticipation) - 1)})
	return nil
}

// AppendPreviousParticipationBits for the beacon state. Appends the new value
// to the the end of list.
func (b *BeaconState) AppendPreviousParticipationBits(val byte) error {
	b.lock.Lock()
	defer b.lock.Unlock()

	bits := b.previousEpochParticipation
	if b.sharedFieldReferences[previousEpochParticipationBits].Refs() > 1 {
		bits = make([]byte, len(b.previousEpochParticipation))
		copy(bits, b.previousEpochParticipation)
		b.sharedFieldReferences[previousEpochParticipationBits].MinusRef()
		b.sharedFieldReferences[previousEpochParticipationBits] = stateutil.NewRef(1)
	}

	b.previousEpochParticipation = append(bits, val)
	b.markFieldAsDirty(previousEpochParticipationBits)
	b.addDirtyIndices(previousEpochParticipationBits, []uint64{uint64(len(b.previousEpochParticipation) - 1)})

	return nil
}

// ModifyPreviousParticipationBits modifies the previous participation bitfield via
// the provided mutator function.
func (b *BeaconState) ModifyPreviousParticipationBits(mutator func(val []byte) ([]byte, error)) error {
	b.lock.Lock()

	participation := b.previousEpochParticipation
	if b.sharedFieldReferences[previousEpochParticipationBits].Refs() > 1 {
		// Copy elements in underlying array by reference.
		participation = make([]byte, len(b.previousEpochParticipation))
		copy(participation, b.previousEpochParticipation)
		b.sharedFieldReferences[previousEpochParticipationBits].MinusRef()
		b.sharedFieldReferences[previousEpochParticipationBits] = stateutil.NewRef(1)
	}
	// Lock is released so that mutator can
	// acquire it.
	b.lock.Unlock()

	var err error
	participation, err = mutator(participation)
	if err != nil {
		return err
	}
	b.lock.Lock()
	defer b.lock.Unlock()
	b.previousEpochParticipation = participation
	b.markFieldAsDirty(previousEpochParticipationBits)
	b.rebuildTrie[previousEpochParticipationBits] = true
	return nil
}

// ModifyCurrentParticipationBits modifies the current participation bitfield via
// the provided mutator function.
func (b *BeaconState) ModifyCurrentParticipationBits(mutator func(val []byte) ([]byte, error)) error {
	b.lock.Lock()

	participation := b.currentEpochParticipation
	if b.sharedFieldReferences[currentEpochParticipationBits].Refs() > 1 {
		// Copy elements in underlying array by reference.
		participation = make([]byte, len(b.currentEpochParticipation))
		copy(participation, b.currentEpochParticipation)
		b.sharedFieldReferences[currentEpochParticipationBits].MinusRef()
		b.sharedFieldReferences[currentEpochParticipationBits] = stateutil.NewRef(1)
	}
	// Lock is released so that mutator can
	// acquire it.
	b.lock.Unlock()

	var err error
	participation, err = mutator(participation)
	if err != nil {
		return err
	}
	b.lock.Lock()
	defer b.lock.Unlock()
	b.currentEpochParticipation = participation
	b.markFieldAsDirty(currentEpochParticipationBits)
	b.rebuildTrie[currentEpochParticipationBits] = true
	return nil
}
