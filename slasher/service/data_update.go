package service

import (
	"context"
	"fmt"
	"sort"
	"time"

	ptypes "github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/sliceutil"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// finalisedChangeUpdater this is a stub for the comming PRs #3133
// Store validator index to public key map Validate attestation signature.
func (s *Service) finalisedChangeUpdater() error {
	secondsPerSlot := params.BeaconConfig().SecondsPerSlot
	d := time.Duration(secondsPerSlot) * time.Second
	tick := time.Tick(d)
	var finalizedEpoch uint64
	for {
		select {
		case <-tick:
			ch, err := s.beaconClient.GetChainHead(s.context, &ptypes.Empty{})
			if err != nil {
				log.Error(err)
				continue
			}
			if ch != nil {
				if ch.FinalizedEpoch > finalizedEpoch {
					finalizedEpoch = ch.FinalizedEpoch
					log.Infof("Finalized epoch %d", ch.FinalizedEpoch)
				}
				continue
			}
			log.Error("No chain head was returned by beacon chain.")
		case <-s.context.Done():
			return status.Error(codes.Canceled, "Stream context canceled")

		}
	}
}

func (s *Service) slasherOldAtetstationFeeder() error {
	ch, err := s.beaconClient.GetChainHead(s.context, &ptypes.Empty{})
	if err != nil {
		log.Error(err)
	}
	if ch.FinalizedEpoch < 2 {
		return fmt.Errorf("archive node doesnt have historic data for slasher to proccess. finalized epoch: %d", ch.FinalizedEpoch)
	}
	for i := uint64(0); i < ch.FinalizedEpoch; i++ {
		ats, err := s.beaconClient.ListAttestations(s.context, &ethpb.ListAttestationsRequest{
			QueryFilter: &ethpb.ListAttestationsRequest_TargetEpoch{TargetEpoch: i},
		})
		if err != nil {
			log.Error(err)
		}
		//bcs, err := s.beaconClient.ListBeaconCommittees(s.context, &ethpb.ListCommitteesRequest{
		//	QueryFilter: &ethpb.ListCommitteesRequest_Epoch{
		//		Epoch: i,
		//	},
		//})
		//if err != nil {
		//	log.Error(err)
		//}
		log.Infof("detecting slashable events on: %v attestations from epoch: %v", len(ats.Attestations), i)
		for _, attestation := range ats.Attestations {
			//e := helpers.SlotToEpoch(attestation.Data.Slot)
			e := attestation.Data.Slot / 8

			bcs, err := s.beaconClient.ListBeaconCommittees(s.context, &ethpb.ListCommitteesRequest{
				QueryFilter: &ethpb.ListCommitteesRequest_Epoch{
					Epoch: e,
				},
			})
			scs, ok := bcs.Committees[attestation.Data.Slot]
			if !ok {
				var keys []uint64
				for k := range bcs.Committees {
					keys = append(keys, k)
				}
				log.Errorf("committees doesnt contain the attestation slot: %d, actual first slot: %v", attestation.Data.Slot, keys)
				continue
			}
			if attestation.Data.CommitteeIndex > uint64(len(scs.Committees)) {
				log.Errorf("committee index is out of range in slot wanted: %v, actual", attestation.Data.CommitteeIndex, len(scs.Committees))
				continue
			}
			sc := scs.Committees[attestation.Data.CommitteeIndex]
			c := sc.ValidatorIndices
			ia, err := ConvertToIndexed(s.context, attestation, c)
			if err != nil {
				log.Error(err)
				continue
			}
			sar, err := s.slasher.IsSlashableAttestation(s.context, ia)
			if err != nil {
				log.Error(err)
				continue
			}
			if len(sar.AttesterSlashing) > 0 {
				log.Infof("slashing response: %v", sar.AttesterSlashing)
			}
		}
	}
	return nil
}

// ConvertToIndexed converts attestation to (almost) indexed-verifiable form.
//
// Note about spec pseudocode definition. The state was used by get_attesting_indices to determine
// the attestation committee. Now that we provide this as an argument, we no longer need to provide
// a state.
//
// Spec pseudocode definition:
//   def get_indexed_attestation(state: BeaconState, attestation: Attestation) -> IndexedAttestation:
//    """
//    Return the indexed attestation corresponding to ``attestation``.
//    """
//    attesting_indices = get_attesting_indices(state, attestation.data, attestation.aggregation_bits)
//    custody_bit_1_indices = get_attesting_indices(state, attestation.data, attestation.custody_bits)
//    assert custody_bit_1_indices.issubset(attesting_indices)
//    custody_bit_0_indices = attesting_indices.difference(custody_bit_1_indices)
//
//    return IndexedAttestation(
//        custody_bit_0_indices=sorted(custody_bit_0_indices),
//        custody_bit_1_indices=sorted(custody_bit_1_indices),
//        data=attestation.data,
//        signature=attestation.signature,
//    )
func ConvertToIndexed(ctx context.Context, attestation *ethpb.Attestation, committee []uint64) (*ethpb.IndexedAttestation, error) {
	attIndices, err := helpers.AttestingIndices(attestation.AggregationBits, committee)
	if err != nil {
		return nil, errors.Wrap(err, "could not get attesting indices")
	}

	cb1i, err := helpers.AttestingIndices(attestation.CustodyBits, committee)
	if err != nil {
		return nil, err
	}
	if !sliceutil.SubsetUint64(cb1i, attIndices) {
		return nil, fmt.Errorf("%v is not a subset of %v", cb1i, attIndices)
	}
	cb1Map := make(map[uint64]bool)
	for _, idx := range cb1i {
		cb1Map[idx] = true
	}
	cb0i := []uint64{}
	for _, idx := range attIndices {
		if !cb1Map[idx] {
			cb0i = append(cb0i, idx)
		}
	}
	sort.Slice(cb0i, func(i, j int) bool {
		return cb0i[i] < cb0i[j]
	})

	sort.Slice(cb1i, func(i, j int) bool {
		return cb1i[i] < cb1i[j]
	})
	inAtt := &ethpb.IndexedAttestation{
		Data:                attestation.Data,
		Signature:           attestation.Signature,
		CustodyBit_0Indices: cb0i,
		CustodyBit_1Indices: cb1i,
	}
	return inAtt, nil
}
