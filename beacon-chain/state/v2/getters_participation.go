package v2

import (
	"github.com/prysmaticlabs/prysm/beacon-chain/core/time"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/math"
)

// CurrentEpochParticipation corresponding to participation bits on the beacon chain.
func (b *BeaconState) CurrentEpochParticipation() ([]byte, error) {
	if !b.hasInnerState() {
		return nil, ErrNilInnerState
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
		return nil, ErrNilInnerState
	}
	if b.state.PreviousEpochParticipation == nil {
		return nil, nil
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.previousEpochParticipation(), nil
}

// UnrealizedCheckpointBalances returns the total balances: active, target attested in
// current epoch and target attested in previous epoch. This function is used to
// compute the "unrealized justification" that a synced Beacon Block will have.
func (b *BeaconState) UnrealizedCheckpointBalances() (uint64, uint64, uint64, error) {
	if !b.hasInnerState() {
		return 0, 0, 0, ErrNilInnerState
	}
	b.lock.RLock()
	defer b.lock.RUnlock()

	cp := b.state.CurrentEpochParticipation
	pp := b.state.PreviousEpochParticipation
	if cp == nil || pp == nil {
		return 0, 0, 0, ErrNilParticipation
	}

	targetIdx := params.BeaconConfig().TimelyTargetFlagIndex
	activeBalance := uint64(0)
	currentTarget := uint64(0)
	prevTarget := uint64(0)
	currentEpoch := time.CurrentEpoch(b)

	var err error
	for i, v := range b.state.Validators {
		active := v.ActivationEpoch <= currentEpoch && currentEpoch < v.ExitEpoch
		if active && !v.Slashed {
			activeBalance, err = math.Add64(activeBalance, v.EffectiveBalance)
			if err != nil {
				return 0, 0, 0, err
			}
			if ((cp[i] >> targetIdx) & 1) == 1 {
				currentTarget, err = math.Add64(currentTarget, v.EffectiveBalance)
				if err != nil {
					return 0, 0, 0, err
				}
			}
			if ((pp[i] >> targetIdx) & 1) == 1 {
				prevTarget, err = math.Add64(prevTarget, v.EffectiveBalance)
				if err != nil {
					return 0, 0, 0, err
				}
			}
		}
	}
	return activeBalance, prevTarget, currentTarget, nil
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
