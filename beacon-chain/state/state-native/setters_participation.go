package state_native

import (
	nativetypes "github.com/prysmaticlabs/prysm/v3/beacon-chain/state/state-native/types"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/state/stateutil"
	"github.com/prysmaticlabs/prysm/v3/runtime/version"
)

// SetPreviousParticipationBits for the beacon state. Updates the entire
// list to a new value by overwriting the previous one.
func (b *BeaconState) SetPreviousParticipationBits(val []byte) error {
	b.lock.Lock()
	defer b.lock.Unlock()

	if b.version == version.Phase0 {
		return errNotSupported("SetPreviousParticipationBits", b.version)
	}

	b.sharedFieldReferences[nativetypes.PreviousEpochParticipationBits].MinusRef()
	b.sharedFieldReferences[nativetypes.PreviousEpochParticipationBits] = stateutil.NewRef(1)

	b.previousEpochParticipation = val
	b.markFieldAsDirty(nativetypes.PreviousEpochParticipationBits)
	b.rebuildTrie[nativetypes.PreviousEpochParticipationBits] = true
	return nil
}

// SetCurrentParticipationBits for the beacon state. Updates the entire
// list to a new value by overwriting the previous one.
func (b *BeaconState) SetCurrentParticipationBits(val []byte) error {
	b.lock.Lock()
	defer b.lock.Unlock()

	if b.version == version.Phase0 {
		return errNotSupported("SetCurrentParticipationBits", b.version)
	}

	b.sharedFieldReferences[nativetypes.CurrentEpochParticipationBits].MinusRef()
	b.sharedFieldReferences[nativetypes.CurrentEpochParticipationBits] = stateutil.NewRef(1)

	b.currentEpochParticipation = val
	b.markFieldAsDirty(nativetypes.CurrentEpochParticipationBits)
	b.rebuildTrie[nativetypes.CurrentEpochParticipationBits] = true
	return nil
}

// AppendCurrentParticipationBits for the beacon state. Appends the new value
// to the the end of list.
func (b *BeaconState) AppendCurrentParticipationBits(val byte) error {
	b.lock.Lock()
	defer b.lock.Unlock()

	if b.version == version.Phase0 {
		return errNotSupported("AppendCurrentParticipationBits", b.version)
	}

	participation := b.currentEpochParticipation
	if b.sharedFieldReferences[nativetypes.CurrentEpochParticipationBits].Refs() > 1 {
		// Copy elements in underlying array by reference.
		participation = make([]byte, len(b.currentEpochParticipation))
		copy(participation, b.currentEpochParticipation)
		b.sharedFieldReferences[nativetypes.CurrentEpochParticipationBits].MinusRef()
		b.sharedFieldReferences[nativetypes.CurrentEpochParticipationBits] = stateutil.NewRef(1)
	}

	b.currentEpochParticipation = append(participation, val)
	b.markFieldAsDirty(nativetypes.CurrentEpochParticipationBits)
	b.addDirtyIndices(nativetypes.CurrentEpochParticipationBits, []uint64{uint64(len(b.currentEpochParticipation) - 1)})
	return nil
}

// AppendPreviousParticipationBits for the beacon state. Appends the new value
// to the the end of list.
func (b *BeaconState) AppendPreviousParticipationBits(val byte) error {
	b.lock.Lock()
	defer b.lock.Unlock()

	if b.version == version.Phase0 {
		return errNotSupported("AppendPreviousParticipationBits", b.version)
	}

	bits := b.previousEpochParticipation
	if b.sharedFieldReferences[nativetypes.PreviousEpochParticipationBits].Refs() > 1 {
		bits = make([]byte, len(b.previousEpochParticipation))
		copy(bits, b.previousEpochParticipation)
		b.sharedFieldReferences[nativetypes.PreviousEpochParticipationBits].MinusRef()
		b.sharedFieldReferences[nativetypes.PreviousEpochParticipationBits] = stateutil.NewRef(1)
	}

	b.previousEpochParticipation = append(bits, val)
	b.markFieldAsDirty(nativetypes.PreviousEpochParticipationBits)
	b.addDirtyIndices(nativetypes.PreviousEpochParticipationBits, []uint64{uint64(len(b.previousEpochParticipation) - 1)})

	return nil
}

// ModifyPreviousParticipationBits modifies the previous participation bitfield via
// the provided mutator function.
func (b *BeaconState) ModifyPreviousParticipationBits(mutator func(val []byte) ([]byte, error)) error {
	b.lock.Lock()

	if b.version == version.Phase0 {
		b.lock.Unlock()
		return errNotSupported("ModifyPreviousParticipationBits", b.version)
	}

	participation := b.previousEpochParticipation
	if b.sharedFieldReferences[nativetypes.PreviousEpochParticipationBits].Refs() > 1 {
		// Copy elements in underlying array by reference.
		participation = make([]byte, len(b.previousEpochParticipation))
		copy(participation, b.previousEpochParticipation)
		b.sharedFieldReferences[nativetypes.PreviousEpochParticipationBits].MinusRef()
		b.sharedFieldReferences[nativetypes.PreviousEpochParticipationBits] = stateutil.NewRef(1)
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
	b.markFieldAsDirty(nativetypes.PreviousEpochParticipationBits)
	b.rebuildTrie[nativetypes.PreviousEpochParticipationBits] = true
	return nil
}

// ModifyCurrentParticipationBits modifies the current participation bitfield via
// the provided mutator function.
func (b *BeaconState) ModifyCurrentParticipationBits(mutator func(val []byte) ([]byte, error)) error {
	b.lock.Lock()

	if b.version == version.Phase0 {
		b.lock.Unlock()
		return errNotSupported("ModifyCurrentParticipationBits", b.version)
	}

	participation := b.currentEpochParticipation
	if b.sharedFieldReferences[nativetypes.CurrentEpochParticipationBits].Refs() > 1 {
		// Copy elements in underlying array by reference.
		participation = make([]byte, len(b.currentEpochParticipation))
		copy(participation, b.currentEpochParticipation)
		b.sharedFieldReferences[nativetypes.CurrentEpochParticipationBits].MinusRef()
		b.sharedFieldReferences[nativetypes.CurrentEpochParticipationBits] = stateutil.NewRef(1)
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
	b.markFieldAsDirty(nativetypes.CurrentEpochParticipationBits)
	b.rebuildTrie[nativetypes.CurrentEpochParticipationBits] = true
	return nil
}
