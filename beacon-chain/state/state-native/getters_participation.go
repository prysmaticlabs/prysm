package state_native

import (
	"github.com/prysmaticlabs/prysm/beacon-chain/core/time"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/math"
	"github.com/prysmaticlabs/prysm/runtime/version"
)

// CurrentEpochParticipation corresponding to participation bits on the beacon chain.
func (b *BeaconState) CurrentEpochParticipation() ([]byte, error) {
	if b.version == version.Phase0 {
		return nil, errNotSupported("CurrentEpochParticipation", b.version)
	}

	if b.currentEpochParticipation == nil {
		return nil, nil
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.currentEpochParticipationVal(), nil
}

// PreviousEpochParticipation corresponding to participation bits on the beacon chain.
func (b *BeaconState) PreviousEpochParticipation() ([]byte, error) {
	if b.version == version.Phase0 {
		return nil, errNotSupported("PreviousEpochParticipation", b.version)
	}

	if b.previousEpochParticipation == nil {
		return nil, nil
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.previousEpochParticipationVal(), nil
}

// UnrealizedCheckpointBalances returns the total balances: active, target attested in
// current epoch and target attested in previous epoch. This function is used to
// compute the "unrealized justification" that a synced Beacon Block will have.
func (b *BeaconState) UnrealizedCheckpointBalances() (uint64, uint64, uint64, error) {
	b.lock.RLock()
	defer b.lock.RUnlock()

	cp := b.currentEpochParticipation
	pp := b.previousEpochParticipation
	if cp == nil || pp == nil {
		return 0, 0, 0, ErrNilParticipation
	}

	targetIdx := params.BeaconConfig().TimelyTargetFlagIndex
	activeBalance := uint64(0)
	currentTarget := uint64(0)
	prevTarget := uint64(0)
	currentEpoch := time.CurrentEpoch(b)

	var err error
	for i, v := range b.validators {
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

// currentEpochParticipationVal corresponding to participation bits on the beacon chain.
// This assumes that a lock is already held on BeaconState.
func (b *BeaconState) currentEpochParticipationVal() []byte {
	tmp := make([]byte, len(b.currentEpochParticipation))
	copy(tmp, b.currentEpochParticipation)
	return tmp
}

// previousEpochParticipationVal corresponding to participation bits on the beacon chain.
// This assumes that a lock is already held on BeaconState.
func (b *BeaconState) previousEpochParticipationVal() []byte {
	tmp := make([]byte, len(b.previousEpochParticipation))
	copy(tmp, b.previousEpochParticipation)
	return tmp
}
