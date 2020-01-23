package service

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	ptypes "github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/slasher/db"
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
	if s.beaconClient == nil {
		return fmt.Errorf("can't feed old attestations to slasher. beacon client has not been started")
	}
	ch, err := s.beaconClient.GetChainHead(s.context, &ptypes.Empty{})
	if err != nil {
		log.Error(err)
		return err
	}
	if ch.FinalizedEpoch < 2 {
		log.Info("archive node doesnt have historic data for slasher to proccess. finalized epoch: %d", ch.FinalizedEpoch)
	}
	log.Infof("Finalized epoch is currently: %d", ch.FinalizedEpoch)
	errOut := make(chan error)
	var errorWg sync.WaitGroup
	e, err := s.slasherDb.GetLatestEpochDetected()
	if err != nil {
		log.Error(err)
		s.Stop()
		return err
	}
	for i := e; i < ch.FinalizedEpoch; i++ {
		ats, err := s.beaconClient.ListAttestations(s.context, &ethpb.ListAttestationsRequest{
			QueryFilter: &ethpb.ListAttestationsRequest_TargetEpoch{TargetEpoch: i},
		})
		if err != nil {
			log.Error(err)
		}
		bcs, err := s.beaconClient.ListBeaconCommittees(s.context, &ethpb.ListCommitteesRequest{
			QueryFilter: &ethpb.ListCommitteesRequest_Epoch{
				Epoch: i,
			},
		})
		if err != nil || bcs == nil {
			log.Error(err)
			continue
		}
		log.Infof("detecting slashable events on: %v attestations from epoch: %v", len(ats.Attestations), i)
		for _, attestation := range ats.Attestations {
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
			s.slasherDb.SaveAttesterSlashings(db.Active, sar.AttesterSlashing)
			if len(sar.AttesterSlashing) > 0 {
				log.Infof("slashing response: %v", sar.AttesterSlashing)
			}
		}
		s.slasherDb.SetLatestEpochDetected(i)
		ch, err = s.beaconClient.GetChainHead(s.context, &ptypes.Empty{})
		if err != nil {
			log.Error(err)
			s.Stop()
		}
	}
	errorWg.Wait()
	close(errOut)
	for err := range errOut {
		log.Error(errors.Wrap(err, "error while writing to db in background"))
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
	if attestation.Data == nil {
		return nil, fmt.Errorf("cant hash nil data in indexed attestation")
	}
	attIndices, err := helpers.AttestingIndices(attestation.AggregationBits, committee)
	if err != nil {
		return nil, errors.Wrap(err, "could not get attesting indices")
	}
	cb0i := []uint64{}
	for _, idx := range attIndices {
		cb0i = append(cb0i, idx)
	}
	sort.Slice(cb0i, func(i, j int) bool {
		return cb0i[i] < cb0i[j]
	})

	inAtt := &ethpb.IndexedAttestation{
		Data:             attestation.Data,
		Signature:        attestation.Signature,
		AttestingIndices: cb0i,
	}
	return inAtt, nil

}
