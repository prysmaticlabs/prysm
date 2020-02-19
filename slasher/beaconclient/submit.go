package beaconclient

import (
	"context"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
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
			if _, err := bs.beaconClient.SubmitAttesterSlashing(ctx, slashing); err != nil {
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
