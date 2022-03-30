package v0

import (
	v0types "github.com/prysmaticlabs/prysm/beacon-chain/state/state-native/v0/types"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"
)

// SetPreviousParticipationBits for the beacon state. Updates the entire
// list to a new value by overwriting the previous one.
func (b *BeaconState) SetPreviousParticipationBits(val []byte) error {
	b.lock.Lock()
	defer b.lock.Unlock()

	b.sharedFieldReferences[b.fieldIndexesRev[v0types.PreviousEpochParticipationBits]].MinusRef()
	b.sharedFieldReferences[b.fieldIndexesRev[v0types.PreviousEpochParticipationBits]] = stateutil.NewRef(1)

	b.previousEpochParticipation = val
	b.markFieldAsDirty(v0types.PreviousEpochParticipationBits)
	b.rebuildTrie[b.fieldIndexesRev[v0types.PreviousEpochParticipationBits]] = true
	return nil
}

// SetCurrentParticipationBits for the beacon state. Updates the entire
// list to a new value by overwriting the previous one.
func (b *BeaconState) SetCurrentParticipationBits(val []byte) error {
	b.lock.Lock()
	defer b.lock.Unlock()

	b.sharedFieldReferences[b.fieldIndexesRev[v0types.CurrentEpochParticipationBits]].MinusRef()
	b.sharedFieldReferences[b.fieldIndexesRev[v0types.CurrentEpochParticipationBits]] = stateutil.NewRef(1)

	b.currentEpochParticipation = val
	b.markFieldAsDirty(v0types.CurrentEpochParticipationBits)
	b.rebuildTrie[b.fieldIndexesRev[v0types.CurrentEpochParticipationBits]] = true
	return nil
}

// AppendCurrentParticipationBits for the beacon state. Appends the new value
// to the the end of list.
func (b *BeaconState) AppendCurrentParticipationBits(val byte) error {
	b.lock.Lock()
	defer b.lock.Unlock()

	participation := b.currentEpochParticipation
	if b.sharedFieldReferences[b.fieldIndexesRev[v0types.CurrentEpochParticipationBits]].Refs() > 1 {
		// Copy elements in underlying array by reference.
		participation = make([]byte, len(b.currentEpochParticipation))
		copy(participation, b.currentEpochParticipation)
		b.sharedFieldReferences[b.fieldIndexesRev[v0types.CurrentEpochParticipationBits]].MinusRef()
		b.sharedFieldReferences[b.fieldIndexesRev[v0types.CurrentEpochParticipationBits]] = stateutil.NewRef(1)
	}

	b.currentEpochParticipation = append(participation, val)
	b.markFieldAsDirty(v0types.CurrentEpochParticipationBits)
	b.addDirtyIndices(v0types.CurrentEpochParticipationBits, []uint64{uint64(len(b.currentEpochParticipation) - 1)})
	return nil
}

// AppendPreviousParticipationBits for the beacon state. Appends the new value
// to the the end of list.
func (b *BeaconState) AppendPreviousParticipationBits(val byte) error {
	b.lock.Lock()
	defer b.lock.Unlock()

	bits := b.previousEpochParticipation
	if b.sharedFieldReferences[b.fieldIndexesRev[v0types.PreviousEpochParticipationBits]].Refs() > 1 {
		bits = make([]byte, len(b.previousEpochParticipation))
		copy(bits, b.previousEpochParticipation)
		b.sharedFieldReferences[b.fieldIndexesRev[v0types.PreviousEpochParticipationBits]].MinusRef()
		b.sharedFieldReferences[b.fieldIndexesRev[v0types.PreviousEpochParticipationBits]] = stateutil.NewRef(1)
	}

	b.previousEpochParticipation = append(bits, val)
	b.markFieldAsDirty(v0types.PreviousEpochParticipationBits)
	b.addDirtyIndices(v0types.PreviousEpochParticipationBits, []uint64{uint64(len(b.previousEpochParticipation) - 1)})

	return nil
}

// ModifyPreviousParticipationBits modifies the previous participation bitfield via
// the provided mutator function.
func (b *BeaconState) ModifyPreviousParticipationBits(mutator func(val []byte) ([]byte, error)) error {
	b.lock.Lock()

	participation := b.previousEpochParticipation
	if b.sharedFieldReferences[b.fieldIndexesRev[v0types.PreviousEpochParticipationBits]].Refs() > 1 {
		// Copy elements in underlying array by reference.
		participation = make([]byte, len(b.previousEpochParticipation))
		copy(participation, b.previousEpochParticipation)
		b.sharedFieldReferences[b.fieldIndexesRev[v0types.PreviousEpochParticipationBits]].MinusRef()
		b.sharedFieldReferences[b.fieldIndexesRev[v0types.PreviousEpochParticipationBits]] = stateutil.NewRef(1)
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
	b.markFieldAsDirty(v0types.PreviousEpochParticipationBits)
	b.rebuildTrie[b.fieldIndexesRev[v0types.PreviousEpochParticipationBits]] = true
	return nil
}

// ModifyCurrentParticipationBits modifies the current participation bitfield via
// the provided mutator function.
func (b *BeaconState) ModifyCurrentParticipationBits(mutator func(val []byte) ([]byte, error)) error {
	b.lock.Lock()

	participation := b.currentEpochParticipation
	if b.sharedFieldReferences[b.fieldIndexesRev[v0types.CurrentEpochParticipationBits]].Refs() > 1 {
		// Copy elements in underlying array by reference.
		participation = make([]byte, len(b.currentEpochParticipation))
		copy(participation, b.currentEpochParticipation)
		b.sharedFieldReferences[b.fieldIndexesRev[v0types.CurrentEpochParticipationBits]].MinusRef()
		b.sharedFieldReferences[b.fieldIndexesRev[v0types.CurrentEpochParticipationBits]] = stateutil.NewRef(1)
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
	b.markFieldAsDirty(v0types.CurrentEpochParticipationBits)
	b.rebuildTrie[b.fieldIndexesRev[v0types.CurrentEpochParticipationBits]] = true
	return nil
}
