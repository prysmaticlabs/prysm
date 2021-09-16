package beaconclient

import (
	"context"
	"strings"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/container/slice"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
)

// subscribeDetectedProposerSlashings subscribes to an event feed for
// slashing objects from the slasher runtime. Upon receiving
// a proposer slashing from the feed, we submit the object to the
// connected beacon node via a client RPC.
func (s *Service) subscribeDetectedProposerSlashings(ctx context.Context, ch chan *ethpb.ProposerSlashing) {
	ctx, span := trace.StartSpan(ctx, "beaconclient.submitProposerSlashing")
	defer span.End()
	sub := s.cfg.ProposerSlashingsFeed.Subscribe(ch)
	defer sub.Unsubscribe()
	for {
		select {
		case slashing := <-ch:
			if _, err := s.cfg.BeaconClient.SubmitProposerSlashing(ctx, slashing); err != nil {
				log.Error(err)
			}
		case <-sub.Err():
			log.Error("Subscriber closed, exiting goroutine")
			return
		case <-ctx.Done():
			log.Error("Context canceled")
			return
		}
	}
}

// subscribeDetectedAttesterSlashings subscribes to an event feed for
// slashing objects from the slasher runtime. Upon receiving an
// attester slashing from the feed, we submit the object to the
// connected beacon node via a client RPC.
func (s *Service) subscribeDetectedAttesterSlashings(ctx context.Context, ch chan *ethpb.AttesterSlashing) {
	ctx, span := trace.StartSpan(ctx, "beaconclient.submitAttesterSlashing")
	defer span.End()
	sub := s.cfg.AttesterSlashingsFeed.Subscribe(ch)
	defer sub.Unsubscribe()
	for {
		select {
		case slashing := <-ch:
			if slashing != nil && slashing.Attestation_1 != nil && slashing.Attestation_2 != nil {
				slashableIndices := slice.IntersectionUint64(slashing.Attestation_1.AttestingIndices, slashing.Attestation_2.AttestingIndices)
				_, err := s.cfg.BeaconClient.SubmitAttesterSlashing(ctx, slashing)
				if err == nil {
					log.WithFields(logrus.Fields{
						"sourceEpoch": slashing.Attestation_1.Data.Source.Epoch,
						"targetEpoch": slashing.Attestation_1.Data.Target.Epoch,
						"indices":     slashableIndices,
					}).Info("Found a valid attester slashing! Submitting to beacon node")
				} else if strings.Contains(err.Error(), helpers.ErrSigFailedToVerify.Error()) {
					log.WithError(err).Errorf("Could not submit attester slashing with indices %v", slashableIndices)
				} else if !strings.Contains(err.Error(), "could not slash") {
					log.WithError(err).Errorf("Could not slash validators with indices %v", slashableIndices)
				}
			}
		case <-sub.Err():
			log.Error("Subscriber closed, exiting goroutine")
			return
		case <-ctx.Done():
			log.Error("Context canceled")
			return
		}
	}
}
