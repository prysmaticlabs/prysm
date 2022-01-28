package v3

// CurrentEpochParticipation corresponding to participation bits on the beacon chain.
func (b *BeaconState) CurrentEpochParticipation() ([]byte, error) {
	if !b.hasInnerState() {
		return nil, nil
	}
	if b.state.CurrentEpochParticipation == nil {
		return nil, nil
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.currentEpochParticipation(), nil
}

// PreviousEpochParticipation corresponding to participation bits on the beacon chain.
func (b *BeaconState) PreviousEpochParticipation() ([]byte, error) {
	if !b.hasInnerState() {
		return nil, nil
	}
	if b.state.PreviousEpochParticipation == nil {
		return nil, nil
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.previousEpochParticipation(), nil
}

// currentEpochParticipation corresponding to participation bits on the beacon chain.
// This assumes that a lock is already held on BeaconState.
func (b *BeaconState) currentEpochParticipation() []byte {
	if !b.hasInnerState() {
		return nil
	}
	tmp := make([]byte, len(b.state.CurrentEpochParticipation))
	copy(tmp, b.state.CurrentEpochParticipation)
	return tmp
}

// previousEpochParticipation corresponding to participation bits on the beacon chain.
// This assumes that a lock is already held on BeaconState.
func (b *BeaconState) previousEpochParticipation() []byte {
	if !b.hasInnerState() {
		return nil
	}
	tmp := make([]byte, len(b.state.PreviousEpochParticipation))
	copy(tmp, b.state.PreviousEpochParticipation)
	return tmp
}
