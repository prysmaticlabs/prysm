package service

import (
	"fmt"
	"time"

	"github.com/prysmaticlabs/prysm/shared/attestationutil"
	ptypes "github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
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
					log.Infof("Finalized epoch %d", ch.FinalizedEpoch)
				}
				continue
			}
			log.Error("No chain head was returned by beacon chain.")
		case <-s.context.Done():
			err := status.Error(codes.Canceled, "Stream context canceled")
			log.WithError(err)
			return err

		}
	}
}

// attestationFeeder feeds attestations that were received by archive endpoint.
func (s *Service) attestationFeeder() error {
	as, err := s.beaconClient.StreamAttestations(s.context, &ptypes.Empty{})
	if err != nil {
		log.WithError(err).Errorf("failed to retrieve attestation stream")
		return err
	}
	for {
		select {
		default:
			if as == nil {
				err := fmt.Errorf("attestation stream is nil. please check your archiver node status")
				log.WithError(err)
				return err
			}
			at, err := as.Recv()
			if err != nil {
				log.WithError(err)
				return err
			}
			bcs, err := s.beaconClient.ListBeaconCommittees(s.context, &ethpb.ListCommitteesRequest{
				QueryFilter: &ethpb.ListCommitteesRequest_Epoch{
					Epoch: at.Data.Target.Epoch,
				},
			})
			if err != nil {
				log.WithError(err).Errorf("Could not list beacon committees for epoch %d", at.Data.Target.Epoch)
				return err
			}
			err = s.detectAttestation(at, bcs)
			if err != nil {
				continue
			}
			log.Info("detected attestation for target: %d", at.Data.Target)
		case <-s.context.Done():
			err := status.Error(codes.Canceled, "Stream context canceled")
			log.WithError(err)
			return err
		}
	}
}

func (s *Service) slasherOldAttestationFeeder() error {
	ch, err := s.getChainHead()
	if err != nil {
		return err
	}
	errOut := make(chan error)
	startFromEpoch, err := s.getLatestDetectedEpoch(err)
	if err != nil {
		return err
	}
	for ep := startFromEpoch; ep < ch.FinalizedEpoch; ep++ {
		ats, bcs, err := s.getDataForDetection(ep)
		if err != nil || bcs == nil {
			log.Error(err)
			continue
		}
		log.Infof("detecting slashable events on: %v attestations from epoch: %v", len(ats.Attestations), ep)
		for _, attestation := range ats.Attestations {
			err := s.detectAttestation(attestation, bcs)
			if err != nil {
				continue
			}
		}
		s.slasherDb.SetLatestEpochDetected(ep)
		s.getChainHead()
	}
	close(errOut)
	for err := range errOut {
		log.Error(errors.Wrap(err, "error while writing to db in background"))
	}
	return nil
}

func (s *Service) detectAttestation(attestation *ethpb.Attestation, beaconCommitteesAtEpoch *ethpb.BeaconCommittees) error {
	slotCommittees, ok := beaconCommitteesAtEpoch.Committees[attestation.Data.Slot]
	if !ok || slotCommittees == nil {
		err := fmt.Errorf("beacon committees object doesnt contain the attestation slot: %d, number of committees: %d",
			attestation.Data.Slot, len(beaconCommitteesAtEpoch.Committees))
		log.WithError(err)
		return err
	}
	if attestation.Data.CommitteeIndex > uint64(len(slotCommittees.Committees)) {
		err := fmt.Errorf("committee index is out of range in slot wanted: %d, actual: %d", attestation.Data.CommitteeIndex, len(slotCommittees.Committees))
		log.WithError(err)
		return err
	}
	attesterCommittee := slotCommittees.Committees[attestation.Data.CommitteeIndex]
	validatorIndices := attesterCommittee.ValidatorIndices
	ia, err := attestationutil.ConvertToIndexed(s.context, attestation, validatorIndices)
	if err != nil {
		log.WithError(err)
		return err
	}
	sar, err := s.slasher.IsSlashableAttestation(s.context, ia)
	if err != nil {
		log.WithError(err)
		return err
	}
	s.slasherDb.SaveAttesterSlashings(db.Active, sar.AttesterSlashing)
	if len(sar.AttesterSlashing) > 0 {
		for _, as := range sar.AttesterSlashing {
			log.WithField("attesterSlashing", as).Info("detected slashing offence")
		}
	}
	return nil
}

func (s *Service) getDataForDetection(epoch uint64) (*ethpb.ListAttestationsResponse, *ethpb.BeaconCommittees, error) {
	ats, err := s.beaconClient.ListAttestations(s.context, &ethpb.ListAttestationsRequest{
		QueryFilter: &ethpb.ListAttestationsRequest_TargetEpoch{TargetEpoch: epoch},
	})
	if err != nil {
		log.WithError(err).Errorf("Could not list attestations for epoch: %d", epoch)
	}
	bcs, err := s.beaconClient.ListBeaconCommittees(s.context, &ethpb.ListCommitteesRequest{
		QueryFilter: &ethpb.ListCommitteesRequest_Epoch{
			Epoch: epoch,
		},
	})
	if err != nil {
		log.WithError(err).Errorf("Could not list beacon committees for epoch: %d", epoch)
	}
	return ats, bcs, err
}

func (s *Service) getLatestDetectedEpoch(err error) (uint64, error) {
	e, err := s.slasherDb.GetLatestEpochDetected()
	if err != nil {
		log.Error(err)
		s.Stop()
		return 0, err
	}
	return e, nil
}

func (s *Service) getChainHead() (*ethpb.ChainHead, error) {
	if s.beaconClient == nil {
		return nil, fmt.Errorf("can't feed old attestations to slasher. beacon client has not been started")
	}
	ch, err := s.beaconClient.GetChainHead(s.context, &ptypes.Empty{})
	if err != nil {
		log.Error(err)
		return nil, err
	}
	if ch.FinalizedEpoch < 2 {
		log.Info("archive node doesnt have historic data for slasher to proccess. finalized epoch: %d", ch.FinalizedEpoch)
	}
	log.WithField("finalizedEpoch", ch.FinalizedEpoch).Info("current finalized epoch on archive node")
	return ch, nil
}
