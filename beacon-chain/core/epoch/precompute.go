package epoch

import (
	"bytes"
	"reflect"

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
	// IsProposer is true if the validator was a proposer.
	IsProposer bool
	// CurrentEpochEffectiveBalance is how much effective balance a validator has current epoch.
	CurrentEpochEffectiveBalance uint64

	// InclusionSlot is the earliest slot a validator had an attestation included in prev epoch.
	InclusionSlot uint64
	// InclusionDistance is the distance between the attestation slot and attestation was included in block.
	InclusionDistance uint64
	// ProposerIndex is the index of proposer at slot where attestation was included.
	ProposerIndex uint64

	//  WinningRootCommitteeBalance is the total balance of the crosslink committee.
	WinningRootCommitteeBalance uint64
	// WinningRootAttestingBalance is the total balance of the crosslink committee tht attested for winning root.
	WinningRootAttestingBalance uint64
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

	for i, v := range state.Validators {
		currentEpoch := helpers.CurrentEpoch(state)
		prevEpoch := helpers.PrevEpoch(state)
		withdrawable := currentEpoch > v.WithdrawableEpoch
		precomp := &ValidatorPrecompute{
			IsSlashed:                    v.Slashed,
			IsWithdrawableCurrentEpoch:   withdrawable,
			CurrentEpochEffectiveBalance: v.EffectiveBalance,
		}
		if helpers.IsActiveValidator(v, currentEpoch) {
			precomp.IsActiveCurrentEpoch = true
			bPrecompute.CurrentEpoch += v.EffectiveBalance
		}
		if helpers.IsActiveValidator(v, prevEpoch) {
			precomp.IsActivePrevEpoch = true
			bPrecompute.PrevEpoch += v.EffectiveBalance
		}
		vPrecompute[i] = precomp
	}
	return vPrecompute, bPrecompute
}

func PrecomputeAttestations(state *pb.BeaconState, v []*ValidatorPrecompute) ([]*ValidatorPrecompute, error) {
	currentEpoch := helpers.CurrentEpoch(state)
	prevEpoch := helpers.PrevEpoch(state)
	for _, a := range append(state.PreviousEpochAttestations, state.CurrentEpochAttestations...) {
		p := &ValidatorPrecompute{}
		if a.Data.Target.Epoch == currentEpoch {
			// Check how validator voted in current epoch.
			p.IsCurrentEpochAttester = true
			votedTarget, err := sameTargetBlockRoot(state, a, currentEpoch)
			if err != nil {
				return nil, err
			}
			if votedTarget {
				p.IsCurrentEpochTargetAttester = true
			}
		} else if a.Data.Target.Epoch == prevEpoch {
			// Check how validator voted in previous epoch.
			p.IsPrevEpochAttester = true
			votedTarget, err := sameTargetBlockRoot(state, a, prevEpoch)
			if err != nil {
				return nil, err
			}
			if votedTarget {
				p.IsPrevEpochAttester = true
			}

			// Inclusion slot and distance are only required for prev epoch attesters.
			aSlot, err := helpers.AttestationDataSlot(state, a.Data)
			if err != nil {
				return nil, err
			}
			p.InclusionSlot = aSlot + a.InclusionDelay
			p.InclusionDistance = a.InclusionDelay
			p.ProposerIndex = a.ProposerIndex

			// Check if validator voted for canonical blocks.
			votedHead, err := sameHeadBlockRoot(state, a)
			if err != nil {
				return nil, err
			}
			if votedHead {
				p.IsHeadAttester = true
			}
		}

		// Update the precompute fields for each attested validators.
		indices, err := helpers.AttestingIndices(state, a.Data, a.AggregationBits)
		if err != nil {
			return nil, err
		}
		for _, i := range indices {
			update(v[i], p)
		}

	}
	return v, nil
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

func update(p1 *ValidatorPrecompute, p2 *ValidatorPrecompute) *ValidatorPrecompute {
	entityType := reflect.TypeOf(p1).Elem()
	for i := 0; i < entityType.NumField(); i++ {
		value := entityType.Field(i)
		oldField := reflect.ValueOf(p1).Elem().Field(i)
		newField := reflect.ValueOf(p2).Elem().FieldByName(value.Name)
		oldField.Set(newField)
	}
	return p1
}
