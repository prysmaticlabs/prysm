package precompute

import (
	"bytes"
	"context"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/attestationutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/traceutil"
	"go.opencensus.io/trace"
)

// Balances stores balances such as prev/current total validator balances, attested balances and more.
// It's used for metrics reporting.
var Balances *Balance

// ProcessAttestations process the attestations in state and update individual validator's pre computes,
// it also tracks and updates epoch attesting balances.
func ProcessAttestations(
	ctx context.Context,
	state *stateTrie.BeaconState,
	vp []*Validator,
	pBal *Balance,
) ([]*Validator, *Balance, error) {
	ctx, span := trace.StartSpan(ctx, "precomputeEpoch.ProcessAttestations")
	defer span.End()

	v := &Validator{}
	var err error

	for _, a := range append(state.PreviousEpochAttestations(), state.CurrentEpochAttestations()...) {
		if a.InclusionDelay == 0 {
			return nil, nil, errors.New("attestation with inclusion delay of 0")
		}
		v.IsCurrentEpochAttester, v.IsCurrentEpochTargetAttester, err = AttestedCurrentEpoch(state, a)
		if err != nil {
			traceutil.AnnotateError(span, err)
			return nil, nil, errors.Wrap(err, "could not check validator attested current epoch")
		}
		v.IsPrevEpochAttester, v.IsPrevEpochTargetAttester, v.IsPrevEpochHeadAttester, err = AttestedPrevEpoch(state, a)
		if err != nil {
			traceutil.AnnotateError(span, err)
			return nil, nil, errors.Wrap(err, "could not check validator attested previous epoch")
		}

		committee, err := helpers.BeaconCommitteeFromState(state, a.Data.Slot, a.Data.CommitteeIndex)
		if err != nil {
			return nil, nil, err
		}
		indices := attestationutil.AttestingIndices(a.AggregationBits, committee)
		vp = UpdateValidator(vp, v, indices, a, a.Data.Slot)
	}

	pBal = UpdateBalance(vp, pBal)
	Balances = pBal

	return vp, pBal, nil
}

// AttestedCurrentEpoch returns true if attestation `a` attested once in current epoch and/or epoch boundary block.
func AttestedCurrentEpoch(s *stateTrie.BeaconState, a *pb.PendingAttestation) (bool, bool, error) {
	currentEpoch := helpers.CurrentEpoch(s)
	var votedCurrentEpoch, votedTarget bool
	// Did validator vote current epoch.
	if a.Data.Target.Epoch == currentEpoch {
		votedCurrentEpoch = true
		same, err := SameTarget(s, a, currentEpoch)
		if err != nil {
			return false, false, err
		}
		if same {
			votedTarget = true
		}
	}
	return votedCurrentEpoch, votedTarget, nil
}

// AttestedPrevEpoch returns true if attestation `a` attested once in previous epoch and epoch boundary block and/or the same head.
func AttestedPrevEpoch(s *stateTrie.BeaconState, a *pb.PendingAttestation) (bool, bool, bool, error) {
	prevEpoch := helpers.PrevEpoch(s)
	var votedPrevEpoch, votedTarget, votedHead bool
	// Did validator vote previous epoch.
	if a.Data.Target.Epoch == prevEpoch {
		votedPrevEpoch = true
		same, err := SameTarget(s, a, prevEpoch)
		if err != nil {
			return false, false, false, errors.Wrap(err, "could not check same target")
		}
		if same {
			votedTarget = true
		}

		if votedTarget {
			same, err = SameHead(s, a)
			if err != nil {
				return false, false, false, errors.Wrap(err, "could not check same head")
			}
			if same {
				votedHead = true
			}
		}
	}
	return votedPrevEpoch, votedTarget, votedHead, nil
}

// SameTarget returns true if attestation `a` attested to the same target block in state.
func SameTarget(state *stateTrie.BeaconState, a *pb.PendingAttestation, e uint64) (bool, error) {
	r, err := helpers.BlockRoot(state, e)
	if err != nil {
		return false, err
	}
	if bytes.Equal(a.Data.Target.Root, r) {
		return true, nil
	}
	return false, nil
}

// SameHead returns true if attestation `a` attested to the same block by attestation slot in state.
func SameHead(state *stateTrie.BeaconState, a *pb.PendingAttestation) (bool, error) {
	r, err := helpers.BlockRootAtSlot(state, a.Data.Slot)
	if err != nil {
		return false, err
	}
	if bytes.Equal(a.Data.BeaconBlockRoot, r) {
		return true, nil
	}
	return false, nil
}

// UpdateValidator updates pre computed validator store.
func UpdateValidator(vp []*Validator, record *Validator, indices []uint64, a *pb.PendingAttestation, aSlot uint64) []*Validator {
	inclusionSlot := aSlot + a.InclusionDelay

	for _, i := range indices {
		if record.IsCurrentEpochAttester {
			vp[i].IsCurrentEpochAttester = true
		}
		if record.IsCurrentEpochTargetAttester {
			vp[i].IsCurrentEpochTargetAttester = true
		}
		if record.IsPrevEpochAttester {
			vp[i].IsPrevEpochAttester = true
			// Update attestation inclusion info if inclusion slot is lower than before
			if inclusionSlot < vp[i].InclusionSlot {
				vp[i].InclusionSlot = aSlot + a.InclusionDelay
				vp[i].InclusionDistance = a.InclusionDelay
				vp[i].ProposerIndex = a.ProposerIndex
			}
		}
		if record.IsPrevEpochTargetAttester {
			vp[i].IsPrevEpochTargetAttester = true
		}
		if record.IsPrevEpochHeadAttester {
			vp[i].IsPrevEpochHeadAttester = true
		}
	}
	return vp
}

// UpdateBalance updates pre computed balance store.
func UpdateBalance(vp []*Validator, bBal *Balance) *Balance {
	for _, v := range vp {
		if !v.IsSlashed {
			if v.IsCurrentEpochAttester {
				bBal.CurrentEpochAttested += v.CurrentEpochEffectiveBalance
			}
			if v.IsCurrentEpochTargetAttester {
				bBal.CurrentEpochTargetAttested += v.CurrentEpochEffectiveBalance
			}
			if v.IsPrevEpochAttester {
				bBal.PrevEpochAttested += v.CurrentEpochEffectiveBalance
			}
			if v.IsPrevEpochTargetAttester {
				bBal.PrevEpochTargetAttested += v.CurrentEpochEffectiveBalance
			}
			if v.IsPrevEpochHeadAttester {
				bBal.PrevEpochHeadAttested += v.CurrentEpochEffectiveBalance
			}
		}
	}

	return EnsureBalancesLowerBound(bBal)
}

// EnsureBalancesLowerBound ensures all the balances such as active current epoch, active previous epoch and more
// have EffectiveBalanceIncrement(1 eth) as a lower bound.
func EnsureBalancesLowerBound(bBal *Balance) *Balance {
	ebi := params.BeaconConfig().EffectiveBalanceIncrement
	if ebi > bBal.ActiveCurrentEpoch {
		bBal.ActiveCurrentEpoch = ebi
	}
	if ebi > bBal.ActivePrevEpoch {
		bBal.ActivePrevEpoch = ebi
	}
	if ebi > bBal.CurrentEpochAttested {
		bBal.CurrentEpochAttested = ebi
	}
	if ebi > bBal.CurrentEpochTargetAttested {
		bBal.CurrentEpochTargetAttested = ebi
	}
	if ebi > bBal.PrevEpochAttested {
		bBal.PrevEpochAttested = ebi
	}
	if ebi > bBal.PrevEpochTargetAttested {
		bBal.PrevEpochTargetAttested = ebi
	}
	if ebi > bBal.PrevEpochHeadAttested {
		bBal.PrevEpochHeadAttested = ebi
	}
	return bBal
}
