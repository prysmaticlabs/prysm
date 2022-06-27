package state_native

import (
	"bytes"
	"context"
	"errors"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/time"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/attestation"
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
func (b *BeaconState) UnrealizedCheckpointBalances(ctx context.Context) (uint64, uint64, uint64, error) {
	b.lock.RLock()
	defer b.lock.RUnlock()

	var cp, pp []byte

	currentEpoch := time.CurrentEpoch(b)
	if b.version == version.Phase0 {
		targetIdx := params.BeaconConfig().TimelyTargetFlagIndex
		currentRoot, err := helpers.BlockRoot(b, currentEpoch)
		if err != nil {
			return 0, 0, 0, err
		}
		cp := make([]byte, len(b.validators))
		currAtt := b.currentEpochAttestations

		pp := make([]byte, len(b.validators))
		prevEpoch := currentEpoch
		if prevEpoch > 0 {
			prevEpoch--
		}
		prevRoot, err := helpers.BlockRoot(b, prevEpoch)
		if err != nil {
			return 0, 0, 0, err
		}
		prevAtt := b.previousEpochAttestations

		for _, a := range append(prevAtt, currAtt...) {
			if a.InclusionDelay == 0 {
				return 0, 0, 0, errors.New("attestation with inclusion delay of 0")
			}
			currTarget := a.Data.Target.Epoch == currentEpoch && bytes.Equal(a.Data.Target.Root, currentRoot)
			prevTarget := a.Data.Target.Epoch == prevEpoch && bytes.Equal(a.Data.Target.Root, prevRoot)
			if currTarget || prevTarget {
				committee, err := helpers.BeaconCommitteeFromState(ctx, b, a.Data.Slot, a.Data.CommitteeIndex)
				if err != nil {
					return 0, 0, 0, err
				}
				indices, err := attestation.AttestingIndices(a.AggregationBits, committee)
				if err != nil {
					return 0, 0, 0, err
				}
				for _, i := range indices {
					if currTarget {
						cp[i] = (1 << targetIdx)
					}
					if prevTarget {
						pp[i] = (1 << targetIdx)
					}
				}
			}
		}
	} else {

		cp = b.currentEpochParticipation
		pp = b.previousEpochParticipation
		if cp == nil || pp == nil {
			return 0, 0, 0, ErrNilParticipation
		}
	}

	return stateutil.UnrealizedCheckpointBalances(cp, pp, b.validators, currentEpoch)

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
