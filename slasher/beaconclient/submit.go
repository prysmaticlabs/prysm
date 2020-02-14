package beaconclient

import (
	"context"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
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
	stateSub := bs.proposerSlashingsFeed.Subscribe(ch)
	defer stateSub.Unsubscribe()
	for {
		select {
		case slashing := <-ch:
			if _, err := bs.client.SubmitProposerSlashing(ctx, slashing); err != nil {
				log.Error(err)
			}
		case <-stateSub.Err():
			logrus.Error("Subscriber closed, exiting goroutine")
			return
		case <-ctx.Done():
			logrus.Error("Context canceled")
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
	stateSub := bs.attesterSlashingsFeed.Subscribe(ch)
	defer stateSub.Unsubscribe()
	for {
		select {
		case slashing := <-ch:
			if _, err := bs.client.SubmitAttesterSlashing(ctx, slashing); err != nil {
				log.Error(err)
			}
		case <-stateSub.Err():
			logrus.Error("Subscriber closed, exiting goroutine")
			return
		case <-ctx.Done():
			logrus.Error("Context canceled")
			return
		}
	}
}
