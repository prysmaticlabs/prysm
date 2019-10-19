package precompute

import (
	"bytes"
	"context"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/traceutil"
	"go.opencensus.io/trace"
)

// ProcessAttestations process the attestations in state and update individual validator's pre computes,
// it also tracks and updates epoch attesting balances.
func ProcessAttestations(
	ctx context.Context,
	state *pb.BeaconState,
	vp []*Validator,
	bp *Balance) ([]*Validator, *Balance, error) {
	ctx, span := trace.StartSpan(ctx, "precomputeEpoch.ProcessAttestations")
	defer span.End()

	v := &Validator{}
	var err error
	for _, a := range append(state.PreviousEpochAttestations, state.CurrentEpochAttestations...) {
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

		// Get attested indices and update the pre computed fields for each attested validators.
		indices, err := helpers.AttestingIndices(state, a.Data, a.AggregationBits)
		if err != nil {
			return nil, nil, err
		}
		// Get attestation slot to find lowest inclusion delayed attestation for each attested validators.
		aSlot, err := helpers.AttestationDataSlot(state, a.Data)
		if err != nil {
			return nil, nil, err

		}
		vp = UpdateValidator(vp, v, indices, a, aSlot)
	}

	bp = UpdateBalance(vp, bp)

	return vp, bp, nil
}

// AttestedCurrentEpoch returns true if attestation `a` attested once in current epoch and/or epoch boundary block.
func AttestedCurrentEpoch(s *pb.BeaconState, a *pb.PendingAttestation) (bool, bool, error) {
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
func AttestedPrevEpoch(s *pb.BeaconState, a *pb.PendingAttestation) (bool, bool, bool, error) {
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

		same, err = SameHead(s, a)
		if err != nil {
			return false, false, false, errors.Wrap(err, "could not check same head")
		}
		if same {
			votedHead = true
		}
	}
	return votedPrevEpoch, votedTarget, votedHead, nil
}

// SameTarget returns true if attestation `a` attested to the same target block in state.
func SameTarget(state *pb.BeaconState, a *pb.PendingAttestation, e uint64) (bool, error) {
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
func SameHead(state *pb.BeaconState, a *pb.PendingAttestation) (bool, error) {
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
		}
		if record.IsPrevEpochTargetAttester {
			vp[i].IsPrevEpochTargetAttester = true
		}
		if record.IsPrevEpochHeadAttester {
			vp[i].IsPrevEpochHeadAttester = true
		}

		// Update attestation inclusion info if inclusion slot is lower than before
		if inclusionSlot < vp[i].InclusionSlot {
			vp[i].InclusionSlot = aSlot + a.InclusionDelay
			vp[i].InclusionDistance = a.InclusionDelay
			vp[i].ProposerIndex = a.ProposerIndex
		}
	}
	return vp
}

// UpdateBalance updates pre computed balance store.
func UpdateBalance(vp []*Validator, bp *Balance) *Balance {
	for _, v := range vp {
		if !v.IsSlashed {
			if v.IsCurrentEpochAttester {
				bp.CurrentEpochAttesters += v.CurrentEpochEffectiveBalance
			}
			if v.IsCurrentEpochTargetAttester {
				bp.CurrentEpochTargetAttesters += v.CurrentEpochEffectiveBalance
			}
			if v.IsPrevEpochAttester {
				bp.PrevEpochAttesters += v.CurrentEpochEffectiveBalance
			}
			if v.IsPrevEpochTargetAttester {
				bp.PrevEpochTargetAttesters += v.CurrentEpochEffectiveBalance
			}
			if v.IsPrevEpochHeadAttester {
				bp.PrevEpochHeadAttesters += v.CurrentEpochEffectiveBalance
			}
		}
	}
	return bp
}
