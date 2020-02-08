package service

import (
	"fmt"
	"time"

	ptypes "github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/attestationutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/slasher/db"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// finalizedChangeUpdater this is a stub for the coming PRs #3133
// Store validator index to public key map Validate attestation signature.
func (s *Service) finalizedChangeUpdater() error {
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
					log.Infof("finalized epoch %d", ch.FinalizedEpoch)
				}
				continue
			}
			log.Error("no chain head was returned by beacon chain.")
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
			att, err := as.Recv()
			if err != nil {
				log.WithError(err)
				return err
			}
			committeeReq := &ethpb.ListCommitteesRequest{
				QueryFilter: &ethpb.ListCommitteesRequest_Epoch{
					Epoch: att.Data.Target.Epoch,
				},
			}
			bCommittees, err := s.beaconClient.ListBeaconCommittees(s.context, committeeReq)
			if err != nil {
				log.WithError(err).Errorf("Could not list beacon committees for epoch %d", att.Data.Target.Epoch)
				return err
			}
			err = s.detectSlashingsFromAtt(att, bCommittees)
			if err != nil {
				continue
			}
			log.Info("detected attestation for target: %d", att.Data.Target)
		case <-s.context.Done():
			err := status.Error(codes.Canceled, "Stream context canceled")
			log.WithError(err)
			return err
		}
	}
}

// historicalAttestationFeeder a function to kick start slashing detection
// after all the included attestations in the canonical chain have been
// slashing detected. latest epoch is being updated after each iteration
// in case it changed during the detection process.
func (s *Service) historicalAttestationFeeder() error {
	ch, err := s.getChainHead()
	if err != nil {
		return err
	}
	errOut := make(chan error)
	startFromEpoch, err := s.getLatestDetectedEpoch()
	if err != nil {
		return err
	}

	for epoch := startFromEpoch; epoch < ch.FinalizedEpoch; epoch++ {
		atts, bCommittees, err := s.attsAndCommitteesForEpoch(epoch)
		if err != nil || bCommittees == nil {
			log.Error(err)
			continue
		}
		log.Infof("Detecting slashable events on: %v attestations from epoch: %v", len(atts), epoch)
		for _, attestation := range atts {
			if err := s.detectSlashingsFromAtt(attestation, bCommittees); err != nil {
				log.Error(err)
				continue
			}
		}
		if err := s.slasherDb.SetLatestEpochDetected(epoch); err != nil {
			log.Error(err)
			continue
		}
	}
	close(errOut)
	for err := range errOut {
		log.Error(errors.Wrap(err, "error while writing to db in background"))
	}
	return nil
}

func (s *Service) detectSlashingsFromAtt(attestation *ethpb.Attestation, beaconCommittee *ethpb.BeaconCommittees) error {
	slotCommittees, ok := beaconCommittee.Committees[attestation.Data.Slot]
	if !ok || slotCommittees == nil {
		err := fmt.Errorf("beacon committees object doesnt contain the attestation slot: %d, number of committees: %d",
			attestation.Data.Slot, len(beaconCommittee.Committees))
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
	attSlashingResp, err := s.slasher.IsSlashableAttestation(s.context, ia)
	if err != nil {
		log.WithError(err)
		return err
	}
	if err := s.slasherDb.SaveAttesterSlashings(db.Active, attSlashingResp.AttesterSlashing); err != nil {
		log.WithError(err)
		return err
	}
	if len(attSlashingResp.AttesterSlashing) > 0 {
		for _, as := range attSlashingResp.AttesterSlashing {
			log.WithField("attesterSlashing", as).Info("detected slashing offence")
		}
	}
	return nil
}

func (s *Service) attsAndCommitteesForEpoch(epoch uint64) ([]*ethpb.Attestation, *ethpb.BeaconCommittees, error) {
	attResp, err := s.beaconClient.ListAttestations(s.context, &ethpb.ListAttestationsRequest{
		QueryFilter: &ethpb.ListAttestationsRequest_TargetEpoch{TargetEpoch: epoch},
	})
	if err != nil {
		log.WithError(err).Errorf("Could not list attestations for epoch: %d", epoch)
	}
	bCommittees, err := s.beaconClient.ListBeaconCommittees(s.context, &ethpb.ListCommitteesRequest{
		QueryFilter: &ethpb.ListCommitteesRequest_Epoch{
			Epoch: epoch,
		},
	})
	if err != nil {
		log.WithError(err).Errorf("Could not list beacon committees for epoch: %d", epoch)
	}
	return attResp.Attestations, bCommittees, err
}

func (s *Service) getLatestDetectedEpoch() (uint64, error) {
	e, err := s.slasherDb.GetLatestEpochDetected()
	if err != nil {
		log.Error(err)
		return 0, err
	}
	return e, nil
}

func (s *Service) getChainHead() (*ethpb.ChainHead, error) {
	if s.beaconClient == nil {
		return nil, errors.New("cannot feed old attestations to slasher, beacon client has not been started")
	}
	ch, err := s.beaconClient.GetChainHead(s.context, &ptypes.Empty{})
	if err != nil {
		log.Error(err)
		return nil, err
	}
	if ch.FinalizedEpoch < 2 {
		log.Info("archive node does not have historic data for slasher to process")
	}
	log.WithField("finalizedEpoch", ch.FinalizedEpoch).Info("current finalized epoch on archive node")
	return ch, nil
}
