package v3

// CurrentEpochParticipation corresponding to participation bits on the beacon chain.
func (b *BeaconState) CurrentEpochParticipation() ([]byte, error) {
	if !b.hasInnerState() {
		return nil, nil
	}
	if b.currentEpochParticipation == nil {
		return nil, nil
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.currentEpochParticipationInternal(), nil
}

// PreviousEpochParticipation corresponding to participation bits on the beacon chain.
func (b *BeaconState) PreviousEpochParticipation() ([]byte, error) {
	if !b.hasInnerState() {
		return nil, nil
	}
	if b.previousEpochParticipation == nil {
		return nil, nil
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.previousEpochParticipationInternal(), nil
}

// currentEpochParticipationInternal corresponding to participation bits on the beacon chain.
// This assumes that a lock is already held on BeaconState.
func (b *BeaconState) currentEpochParticipationInternal() []byte {
	if !b.hasInnerState() {
		return nil
	}
	tmp := make([]byte, len(b.currentEpochParticipation))
	copy(tmp, b.currentEpochParticipation)
	return tmp
}

// previousEpochParticipationInternal corresponding to participation bits on the beacon chain.
// This assumes that a lock is already held on BeaconState.
func (b *BeaconState) previousEpochParticipationInternal() []byte {
	if !b.hasInnerState() {
		return nil
	}
	tmp := make([]byte, len(b.previousEpochParticipation))
	copy(tmp, b.previousEpochParticipation)
	return tmp
}
