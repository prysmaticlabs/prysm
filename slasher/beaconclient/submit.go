package beaconclient

import (
	"context"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"go.opencensus.io/trace"
)

// submitProposerSlashings subscribes to an event feed for
// slashing objects from the slasher runtime. Upon receiving
// a proposer slashing from the feed, we submit the object to the
// connected beacon node via a client RPC.
func (bs *Service) submitProposerSlashings(ctx context.Context) {
	ctx, span := trace.StartSpan(ctx, "beaconclient.submitProposerSlashing")
	defer span.End()
	item := &ethpb.ProposerSlashing{}
	if _, err := bs.client.SubmitProposerSlashing(ctx, item); err != nil {
		log.Error(err)
	}
}

// submitAttesterSlashings subscribes to an event feed for
// slashing objects from the slasher runtime. Upon receiving an
// attester slashing from the feed, we submit the object to the
// connected beacon node via a client RPC.
func (bs *Service) submitAttesterSlashings(ctx context.Context) {
	ctx, span := trace.StartSpan(ctx, "beaconclient.submitAttesterSlashing")
	defer span.End()
	item := &ethpb.AttesterSlashing{}
	if _, err := bs.client.SubmitAttesterSlashing(ctx, item); err != nil {
		log.Error(err)
	}
}
