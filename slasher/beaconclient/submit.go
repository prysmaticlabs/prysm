package beaconclient

import (
	"context"
	"strings"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/sliceutil"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
)

// subscribeDetectedProposerSlashings subscribes to an event feed for
// slashing objects from the slasher runtime. Upon receiving
// a proposer slashing from the feed, we submit the object to the
// connected beacon node via a client RPC.
func (bs *Service) subscribeDetectedProposerSlashings(ctx context.Context, ch chan *ethpb.ProposerSlashing) {
	ctx, span := trace.StartSpan(ctx, "beaconclient.submitProposerSlashing")
	defer span.End()
	sub := bs.proposerSlashingsFeed.Subscribe(ch)
	defer sub.Unsubscribe()
	for {
		select {
		case slashing := <-ch:
			if _, err := bs.beaconClient.SubmitProposerSlashing(ctx, slashing); err != nil {
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
func (bs *Service) subscribeDetectedAttesterSlashings(ctx context.Context, ch chan *ethpb.AttesterSlashing) {
	ctx, span := trace.StartSpan(ctx, "beaconclient.submitAttesterSlashing")
	defer span.End()
	sub := bs.attesterSlashingsFeed.Subscribe(ch)
	defer sub.Unsubscribe()
	for {
		select {
		case slashing := <-ch:
			_, err := bs.beaconClient.SubmitAttesterSlashing(ctx, slashing)
			if err == nil {
				slash := slashing
				if slash != nil && slash.Attestation_1 != nil && slash.Attestation_2 != nil {
					slashableIndices := sliceutil.IntersectionUint64(slash.Attestation_1.AttestingIndices, slash.Attestation_2.AttestingIndices)
					log.WithFields(logrus.Fields{
						"sourceEpoch": slash.Attestation_1.Data.Source.Epoch,
						"targetEpoch": slash.Attestation_1.Data.Target.Epoch,
						"indices":     slashableIndices,
					}).Info("Found a valid attester slashing! Submitting to beacon node")
				}
			} else if strings.Contains(err.Error(), helpers.ErrSigFailedToVerify.Error()) {
				log.WithError(err).Error("Could not submit attester slashing")
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
