package epoch

import (
	"bytes"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

// ValidatorPrecompute track the pre-compute of individual validator info such
// whether validator has performed certain action. (i.e. votes, block inclusion,
// winning root participation)
type ValidatorPrecompute struct {
	// IsSlashed is true if the validator has been slashed.
	IsSlashed bool
	// IsWithdrawableCurrentEpoch is true if the validator can withdraw current epoch.
	IsWithdrawableCurrentEpoch bool
	// IsActiveCurrentEpoch is true if the validator was active current epoch.
	IsActiveCurrentEpoch bool
	// IsActivePrevEpoch is true if the validator was active prev epoch.
	IsActivePrevEpoch bool
	// IsCurrentEpochAttester is true if the validator attested current epoch.
	IsCurrentEpochAttester bool
	// IsCurrentEpochTargetAttester is true if the validator attested current epoch target.
	IsCurrentEpochTargetAttester bool
	// IsPrevEpochAttester is true if the validator attested previous epoch.
	IsPrevEpochAttester bool
	// IsPrevEpochTargetAttester is true if the validator attested previous epoch target.
	IsPrevEpochTargetAttester bool
	// IsHeadAttester is true if the validator attested head.
	IsHeadAttester bool
	// CurrentEpochEffectiveBalance is how much effective balance a validator has current epoch.
	CurrentEpochEffectiveBalance uint64

	// InclusionDistance is the distance between the attestation slot and attestation was included in block.
	InclusionDistance uint64
	// ProposerIndex is the index of proposer at slot where attestation was included.
	ProposerIndex uint64
}

// BalancePrecompute track the different sets of balances during prev and current epochs.
type BalancePrecompute struct {
	// CurrentEpoch is the total effective balance of all active validators during current epoch.
	CurrentEpoch uint64
	// PrevEpoch is the total effective balance of all active validators during prev epoch.
	PrevEpoch uint64
	// CurrentEpochAttesters is the total effective balance of all validators who attested during current epoch.
	CurrentEpochAttesters uint64
	// CurrentEpochTargetAttesters is the total effective balance of all validators who attested
	// for epoch boundary block during current epoch.
	CurrentEpochTargetAttesters uint64
	// PrevEpochAttesters is the total effective balance of all validators who attested during prev epoch.
	PrevEpochAttesters uint64
	// PrevEpochTargetAttesters is the total effective balance of all validators who attested
	// for epoch boundary block during prev epoch.
	PrevEpochTargetAttesters uint64
	// PrevEpochHeadAttesters is the total effective balance of all validators who attested
	// correctly for head block during prev epoch.
	PrevEpochHeadAttesters uint64
}

// NewPrecompute returns new instances of ValidatorPrecompute and BalancePrecompute.
func NewPrecompute(state *pb.BeaconState) ([]*ValidatorPrecompute, *BalancePrecompute) {
	vPrecompute := make([]*ValidatorPrecompute, len(state.Validators))
	bPrecompute := &BalancePrecompute{}

	currentEpoch := helpers.CurrentEpoch(state)
	prevEpoch := helpers.PrevEpoch(state)

	for i, v := range state.Validators {
		withdrawable := currentEpoch > v.WithdrawableEpoch
		p := &ValidatorPrecompute{
			IsSlashed:                    v.Slashed,
			IsWithdrawableCurrentEpoch:   withdrawable,
			CurrentEpochEffectiveBalance: v.EffectiveBalance,
		}
		if helpers.IsActiveValidator(v, currentEpoch) {
			p.IsActiveCurrentEpoch = true
			bPrecompute.CurrentEpoch += v.EffectiveBalance
		}
		if helpers.IsActiveValidator(v, prevEpoch) {
			p.IsActivePrevEpoch = true
			bPrecompute.PrevEpoch += v.EffectiveBalance
		}
		vPrecompute[i] = p
	}
	return vPrecompute, bPrecompute
}

func PrecomputeAttestations(
	state *pb.BeaconState,
	vp []*ValidatorPrecompute,
	bp *BalancePrecompute) ([]*ValidatorPrecompute, *BalancePrecompute, error) {
	currentEpoch := helpers.CurrentEpoch(state)
	prevEpoch := helpers.PrevEpoch(state)

	var currentEpochAttester, currentEpochTargetAttester, prevEpochAttester, prevEpochTargetAttester, headAttester bool

	for _, a := range append(state.PreviousEpochAttestations, state.CurrentEpochAttestations...) {
		if a.Data.Target.Epoch == currentEpoch {
			// Check how validator voted in current epoch.
			currentEpochAttester = true
			votedTarget, err := sameTargetBlockRoot(state, a, currentEpoch)
			if err != nil {
				return nil, nil, err
			}
			if votedTarget {
				currentEpochTargetAttester = true
			}
		} else if a.Data.Target.Epoch == prevEpoch {
			// Check how validator voted in previous epoch.
			prevEpochAttester = true
			votedTarget, err := sameTargetBlockRoot(state, a, prevEpoch)
			if err != nil {
				return nil, nil, err
			}
			if votedTarget {
				prevEpochTargetAttester = true
			}

			// Check if validator voted for canonical blocks.
			votedHead, err := sameHeadBlockRoot(state, a)
			if err != nil {
				return nil, nil, err
			}
			if votedHead {
				headAttester = true
			}
		}

		// Update the precompute fields for each attested validators.
		indices, err := helpers.AttestingIndices(state, a.Data, a.AggregationBits)
		if err != nil {
			return nil, nil, err
		}
		for _, i := range indices {
			if currentEpochAttester {
				vp[i].IsCurrentEpochAttester = true
			}
			if currentEpochTargetAttester {
				vp[i].IsCurrentEpochTargetAttester = true
			}
			if prevEpochAttester {
				vp[i].IsPrevEpochAttester = true
			}
			if prevEpochTargetAttester {
				vp[i].IsPrevEpochTargetAttester = true
			}
			if headAttester {
				vp[i].IsHeadAttester = true
			}
			vp[i].InclusionDistance = a.InclusionDelay
			vp[i].ProposerIndex = a.ProposerIndex
		}
	}

	for _, v := range vp {
		if !v.IsSlashed {
			if v.IsCurrentEpochAttester {
				bp.CurrentEpochAttesters += v.CurrentEpochEffectiveBalance
			}
			if v.IsCurrentEpochTargetAttester {
				bp.CurrentEpochTargetAttesters += v.CurrentEpochEffectiveBalance
			}
			if v.IsPrevEpochAttester {
				bp.PrevEpoch += v.CurrentEpochEffectiveBalance
			}
			if v.IsPrevEpochTargetAttester {
				bp.PrevEpochTargetAttesters += v.CurrentEpochEffectiveBalance
			}
			if v.IsHeadAttester {
				bp.PrevEpochHeadAttesters += v.CurrentEpochEffectiveBalance
			}
		}
	}

	return vp, bp, nil
}

func sameTargetBlockRoot(state *pb.BeaconState, a *pb.PendingAttestation, e uint64) (bool, error) {
	r, err := helpers.BlockRoot(state, e)
	if err != nil {
		return false, err
	}
	if bytes.Equal(a.Data.Target.Root, r) {
		return true, nil
	}
	return false, nil
}

func sameHeadBlockRoot(state *pb.BeaconState, a *pb.PendingAttestation) (bool, error) {
	aSlot, err := helpers.AttestationDataSlot(state, a.Data)
	if err != nil {
		return false, err
	}
	r, err := helpers.BlockRootAtSlot(state, aSlot)
	if err != nil {
		return false, err
	}
	if bytes.Equal(a.Data.BeaconBlockRoot, r) {
		return true, nil
	}
	return false, nil
}
