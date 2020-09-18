package validator

import (
	"context"
	"sort"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

type proposerAtts []*ethpb.Attestation

// split splits attestation list into two groups: valid and invalid attestations.
// The first group passes the all the required checks for attestation to be considered for proposing.
// And attestations from the second group should be deleted.
func (al proposerAtts) split(ctx context.Context, state *stateTrie.BeaconState) (proposerAtts, proposerAtts) {
	currEpoch := helpers.SlotToEpoch(state.Slot())
	var prevEpoch uint64
	if currEpoch == 0 {
		prevEpoch = 0
	} else {
		prevEpoch = currEpoch - 1
	}
	isValid := func(att *ethpb.Attestation) bool {
		if att == nil || att.Data == nil || att.Data.Target == nil {
			return false
		}
		if att.Data.Target.Epoch != prevEpoch && att.Data.Target.Epoch != currEpoch {
			return false
		}
		if helpers.SlotToEpoch(att.Data.Slot) != att.Data.Target.Epoch {
			return false
		}
		return true
	}
	getExistingAtts := func(att *ethpb.Attestation) []*pbp2p.PendingAttestation {
		if att.Data.Target.Epoch == currEpoch {
			return state.CurrentEpochAttestations()
		} else if att.Data.Target.Epoch == prevEpoch {
			return state.PreviousEpochAttestations()
		}
		return []*pbp2p.PendingAttestation{}
	}

	validAtts := make([]*ethpb.Attestation, 0, len(al))
	invalidAtts := make([]*ethpb.Attestation, 0, len(al))
	for _, att := range al {
		// Short-circuit validation, w/o doing any more processing (and saving pending attestations).
		if isValid(att) {
			// To maximize profit, the validator should attempt to gather aggregate attestations that
			// include singular attestations from the largest number of validators whose signatures
			// from the same epoch have not previously been added on chain.
			//
			// Depending on attestations epoch (current/previous) state attestations are selected,
			// and aggregate bits of those existing attestations are removed from the given one (taking
			// committee and slot numbers into account).
			if att := unmarkSeenValidators(getExistingAtts(att), att); att.AggregationBits.Count() > 0 {
				if _, err := blocks.ProcessAttestation(ctx, state, att); err == nil {
					validAtts = append(validAtts, att)
					continue
				}
			}
		}
		invalidAtts = append(invalidAtts, att)
	}
	return validAtts, invalidAtts
}

func unmarkSeenValidators(existingAtts []*pbp2p.PendingAttestation, originalAtt *ethpb.Attestation) *ethpb.Attestation {
	att := stateTrie.CopyAttestation(originalAtt)
	for _, existingAtt := range existingAtts {
		if existingAtt.Data.Slot == att.Data.Slot && existingAtt.Data.CommitteeIndex == att.Data.CommitteeIndex {
			att.AggregationBits = att.AggregationBits.And(existingAtt.AggregationBits.Not())
		}
	}
	return att
}

func (al proposerAtts) sortByProfitability() proposerAtts {
	if len(al) < 2 {
		return al
	}
	sort.Slice(al, func(i, j int) bool {
		if al[i].Data.Slot == al[j].Data.Slot {
			return al[i].AggregationBits.Count() > al[j].AggregationBits.Count()
		}
		return al[i].Data.Slot > al[j].Data.Slot
	})
	return al
}

func (al proposerAtts) limitToMaxAttestations() proposerAtts {
	if uint64(len(al)) > params.BeaconConfig().MaxAttestations {
		return al[:params.BeaconConfig().MaxAttestations]
	}
	return al
}
