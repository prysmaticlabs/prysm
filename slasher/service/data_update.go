package service

import (
	"context"
	"fmt"
	"time"

	ptypes "github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/attestationutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/sliceutil"
	"github.com/prysmaticlabs/prysm/slasher/db"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// historicalAttestationFeeder starts performing slashing detection
// on all historical attestations made until the current head.
// The latest epoch is updated after each iteration in case the long
// process is interrupted.
func (s *Service) historicalAttestationFeeder() error {
	startFromEpoch, err := s.getLatestDetectedEpoch()
	if err != nil {
		return errors.Wrap(err, "failed to latest detected epoch")
	}
	ch, err := s.getChainHead()
	if err != nil {
		return errors.Wrap(err, "failed to get chain head")
	}

	for epoch := startFromEpoch; epoch < ch.FinalizedEpoch; epoch++ {
		atts, bCommittees, err := s.attsAndCommitteesForEpoch(epoch)
		if err != nil || bCommittees == nil {
			log.Error(err)
			continue
		}
		log.Infof("Checking %v attestations from epoch %v for slashable events", len(atts), epoch)
		for _, attestation := range atts {
			idxAtt, err := convertToIndexed(s.context, attestation, bCommittees)
			if err = s.detectSlashings(idxAtt); err != nil {
				log.Error(err)
				continue
			}
		}
		if err := s.slasherDb.SetLatestEpochDetected(epoch); err != nil {
			log.Error(err)
			continue
		}
	}
	return nil
}

// attestationFeeder feeds attestations that were received by archive endpoint.
func (s *Service) attestationFeeder() error {
	as, err := s.beaconClient.StreamAttestations(s.context, &ptypes.Empty{})
	if err != nil {
		return errors.Wrap(err, "failed to retrieve attestation stream")
	}
	for {
		select {
		default:
			if as == nil {
				return fmt.Errorf("attestation stream is nil. please check your archiver node status")
			}
			att, err := as.Recv()
			if err != nil {
				return err
			}
			bCommittees, err := s.getCommittees(att)
			if err != nil {
				err = errors.Wrapf(err, "could not list beacon committees for epoch %d", att.Data.Target.Epoch)
				log.WithError(err)
				continue
			}
			idxAtt, err := convertToIndexed(s.context, att, bCommittees)
			if err = s.detectSlashings(idxAtt); err != nil {
				log.Error(err)
				continue
			}
			log.Infof("detected attestation for target: %d", att.Data.Target.Epoch)
		case <-s.context.Done():
			return status.Error(codes.Canceled, "Stream context canceled")
		}
	}
}

// finalizedChangeUpdater this is a stub for the coming PRs #3133.
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

func (s *Service) detectSlashings(idxAtt *ethpb.IndexedAttestation) error {
	attSlashingResp, err := s.slasher.IsSlashableAttestation(s.context, idxAtt)
	if err != nil {
		return errors.Wrap(err, "failed to check attestation")
	}

	if len(attSlashingResp.AttesterSlashing) > 0 {
		if err := s.slasherDb.SaveAttesterSlashings(db.Active, attSlashingResp.AttesterSlashing); err != nil {
			return errors.Wrap(err, "failed to save attester slashings")
		}
		for _, as := range attSlashingResp.AttesterSlashing {
			slashableIndices := sliceutil.IntersectionUint64(as.Attestation_1.AttestingIndices, as.Attestation_2.AttestingIndices)
			log.WithFields(logrus.Fields{
				"target1":          as.Attestation_1.Data.Target.Epoch,
				"source1":          as.Attestation_1.Data.Target.Epoch,
				"target2":          as.Attestation_2.Data.Target.Epoch,
				"source2":          as.Attestation_2.Data.Target.Epoch,
				"slashableIndices": slashableIndices,
			}).Info("Detected slashing offence")
		}
	}
	return nil
}

func (s *Service) getCommittees(at *ethpb.Attestation) (*ethpb.BeaconCommittees, error) {
	epoch := at.Data.Target.Epoch
	committees, err := committeesCache.Get(s.context, epoch)
	if err != nil {
		return nil, err
	}
	if committees != nil {
		return committees, nil
	}
	committeeReq := &ethpb.ListCommitteesRequest{
		QueryFilter: &ethpb.ListCommitteesRequest_Epoch{
			Epoch: epoch,
		},
	}
	bCommittees, err := s.beaconClient.ListBeaconCommittees(s.context, committeeReq)
	if err != nil {
		log.WithError(err).Errorf("Could not list beacon committees for epoch %d", at.Data.Target.Epoch)
		return nil, err
	}
	committeesCache.Put(s.context, epoch, bCommittees)
	return bCommittees, nil
}

func convertToIndexed(ctx context.Context, att *ethpb.Attestation, bCommittee *ethpb.BeaconCommittees) (*ethpb.IndexedAttestation, error) {
	slotCommittees, ok := bCommittee.Committees[att.Data.Slot]
	if !ok || slotCommittees == nil {
		return nil, fmt.Errorf(
			"could not get commitees for att slot: %d, number of committees: %d",
			att.Data.Slot,
			len(bCommittee.Committees),
		)
	}
	if att.Data.CommitteeIndex > uint64(len(slotCommittees.Committees)) {
		return nil, fmt.Errorf(
			"committee index is out of range in slot wanted: %d, actual: %d",
			att.Data.CommitteeIndex,
			len(slotCommittees.Committees),
		)
	}
	attCommittee := slotCommittees.Committees[att.Data.CommitteeIndex]
	validatorIndices := attCommittee.ValidatorIndices
	idxAtt, err := attestationutil.ConvertToIndexed(ctx, att, validatorIndices)
	if err != nil {
		return nil, err
	}
	return idxAtt, nil
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
		return nil, err
	}
	if ch.FinalizedEpoch < 2 {
		log.Info("archive node does not have historic data for slasher to process")
	}
	log.WithField("finalizedEpoch", ch.FinalizedEpoch).Info("current finalized epoch on archive node")
	return ch, nil
}
