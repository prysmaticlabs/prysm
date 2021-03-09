package simulator

import (
	"math"

	"github.com/sirupsen/logrus"

	types "github.com/prysmaticlabs/eth2-types"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/rand"
)

func generateAttestationsForSlot(
	simParams *Parameters, slot types.Slot,
) ([]*ethpb.IndexedAttestation, []*ethpb.AttesterSlashing) {
	attestations := make([]*ethpb.IndexedAttestation, 0)
	slashings := make([]*ethpb.AttesterSlashing, 0)
	currentEpoch := helpers.SlotToEpoch(slot)

	committeesPerSlot := helpers.SlotCommitteeCount(simParams.NumValidators)
	valsPerCommittee := simParams.NumValidators / (committeesPerSlot * uint64(params.BeaconConfig().SlotsPerEpoch))
	valsPerSlot := committeesPerSlot * valsPerCommittee

	var sourceEpoch types.Epoch = 0
	if currentEpoch != 0 {
		sourceEpoch = currentEpoch - 1
	}

	var slashedIndices []uint64
	startIdx := valsPerSlot * uint64(slot%params.BeaconConfig().SlotsPerEpoch)
	endIdx := startIdx + valsPerCommittee
	for c := types.CommitteeIndex(0); uint64(c) < committeesPerSlot; c++ {
		attData := &ethpb.AttestationData{
			Slot:            slot,
			CommitteeIndex:  c,
			BeaconBlockRoot: bytesutil.PadTo([]byte("block"), 32),
			Source: &ethpb.Checkpoint{
				Epoch: sourceEpoch,
				Root:  bytesutil.PadTo([]byte("source"), 32),
			},
			Target: &ethpb.Checkpoint{
				Epoch: currentEpoch,
				Root:  bytesutil.PadTo([]byte("target"), 32),
			},
		}

		valsPerAttestation := uint64(math.Floor(simParams.AggregationPercent * float64(valsPerCommittee)))
		for i := startIdx; i < endIdx; i += valsPerAttestation {
			attEndIdx := i + valsPerAttestation
			if attEndIdx >= endIdx {
				attEndIdx = endIdx
			}
			indices := make([]uint64, 0, valsPerAttestation)
			for idx := i; idx < attEndIdx; idx++ {
				indices = append(indices, idx)
			}
			att := &ethpb.IndexedAttestation{
				AttestingIndices: indices,
				Data:             attData,
				Signature:        params.BeaconConfig().EmptySignature[:],
			}
			attestations = append(attestations, att)
			if rand.NewGenerator().Float64() < simParams.AttesterSlashingProbab {
				slashableAtt := makeSlashableFromAtt(att, []uint64{indices[0]})
				slashedIndices = append(slashedIndices, slashableAtt.AttestingIndices...)
				slashings = append(slashings, &ethpb.AttesterSlashing{
					Attestation_1: att,
					Attestation_2: slashableAtt,
				})
				attestations = append(attestations, slashableAtt)
			}
		}
		startIdx += valsPerCommittee
		endIdx += valsPerCommittee
	}
	if len(slashedIndices) > 0 {
		log.WithFields(logrus.Fields{
			"amount":  len(slashedIndices),
			"indices": slashedIndices,
		}).Infof("Slashable attestation made")
	}
	return attestations, slashings
}

func makeSlashableFromAtt(att *ethpb.IndexedAttestation, indices []uint64) *ethpb.IndexedAttestation {
	if att.Data.Source.Epoch <= 2 {
		return makeDoubleVoteFromAtt(att, indices)
	}
	attData := &ethpb.AttestationData{
		Slot:            att.Data.Slot,
		CommitteeIndex:  att.Data.CommitteeIndex,
		BeaconBlockRoot: att.Data.BeaconBlockRoot,
		Source: &ethpb.Checkpoint{
			Epoch: att.Data.Source.Epoch - 3,
			Root:  att.Data.Source.Root,
		},
		Target: &ethpb.Checkpoint{
			Epoch: att.Data.Target.Epoch,
			Root:  att.Data.Target.Root,
		},
	}
	return &ethpb.IndexedAttestation{
		AttestingIndices: indices,
		Data:             attData,
		Signature:        params.BeaconConfig().EmptySignature[:],
	}
}

func makeDoubleVoteFromAtt(att *ethpb.IndexedAttestation, indices []uint64) *ethpb.IndexedAttestation {
	attData := &ethpb.AttestationData{
		Slot:            att.Data.Slot,
		CommitteeIndex:  att.Data.CommitteeIndex,
		BeaconBlockRoot: bytesutil.PadTo([]byte("slash me"), 32),
		Source: &ethpb.Checkpoint{
			Epoch: att.Data.Source.Epoch,
			Root:  att.Data.Source.Root,
		},
		Target: &ethpb.Checkpoint{
			Epoch: att.Data.Target.Epoch,
			Root:  att.Data.Target.Root,
		},
	}
	return &ethpb.IndexedAttestation{
		AttestingIndices: indices,
		Data:             attData,
		Signature:        params.BeaconConfig().EmptySignature[:],
	}
}
