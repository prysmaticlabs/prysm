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
		v.IsCurrentEpochAttester, v.IsCurrentEpochTargetAttester, err = attestedCurrentEpoch(state, a)
		if err != nil {
			traceutil.AnnotateError(span, err)
			return nil, nil, errors.Wrap(err, "could not check validator attested current epoch")
		}
		v.IsPrevEpochAttester, v.IsPrevEpochTargetAttester, v.IsPrevEpochHeadAttester, err = attestedPrevEpoch(state, a)
		if err != nil {
			traceutil.AnnotateError(span, err)
			return nil, nil, errors.Wrap(err, "could not check validator attested previous epoch")
		}

		// Get attested indices and update the pre computed fields for each attested validators.
		indices, err := helpers.AttestingIndices(state, a.Data, a.AggregationBits)
		if err != nil {
			return nil, nil, err
		}
		vp = updateValidator(vp, v, indices, a)
	}

	bp = updateBalance(vp, bp)

	return vp, bp, nil
}

// Has attestation `a` attested once in current epoch and epoch boundary block.
func attestedCurrentEpoch(s *pb.BeaconState, a *pb.PendingAttestation) (bool, bool, error) {
	currentEpoch := helpers.CurrentEpoch(s)
	var votedCurrentEpoch, votedTarget bool
	// Did validator vote current epoch.
	if a.Data.Target.Epoch == currentEpoch {
		votedCurrentEpoch = true
		same, err := sameTarget(s, a, currentEpoch)
		if err != nil {
			return false, false, err
		}
		if same {
			votedTarget = true
		}
	}
	return votedCurrentEpoch, votedTarget, nil
}

// Has attestation `a` attested once in previous epoch and epoch boundary block and the same head.
func attestedPrevEpoch(s *pb.BeaconState, a *pb.PendingAttestation) (bool, bool, bool, error) {
	prevEpoch := helpers.PrevEpoch(s)
	var votedPrevEpoch, votedTarget, votedHead bool
	// Did validator vote previous epoch.
	if a.Data.Target.Epoch == prevEpoch {
		votedPrevEpoch = true
		same, err := sameTarget(s, a, prevEpoch)
		if err != nil {
			return false, false, false, errors.Wrap(err, "could not check same target")
		}
		if same {
			votedTarget = true
		}

		same, err = sameHead(s, a)
		if err != nil {
			return false, false, false, errors.Wrap(err, "could not check same head")
		}
		if same {
			votedHead = true
		}
	}
	return votedPrevEpoch, votedTarget, votedHead, nil
}

// Has attestation `a` attested to the same target block in state.
func sameTarget(state *pb.BeaconState, a *pb.PendingAttestation, e uint64) (bool, error) {
	r, err := helpers.BlockRoot(state, e)
	if err != nil {
		return false, err
	}
	if bytes.Equal(a.Data.Target.Root, r) {
		return true, nil
	}
	return false, nil
}

// Has attestation `a` attested to the same block by attestation slot in state.
func sameHead(state *pb.BeaconState, a *pb.PendingAttestation) (bool, error) {
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

// This updates pre computed validator store.
func updateValidator(vp []*Validator, record *Validator, indices []uint64, a *pb.PendingAttestation) []*Validator {
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
		vp[i].InclusionDistance = a.InclusionDelay
		vp[i].ProposerIndex = a.ProposerIndex
	}
	return vp
}

// This updates pre computed balance store.
func updateBalance(vp []*Validator, bp *Balance) *Balance {
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
