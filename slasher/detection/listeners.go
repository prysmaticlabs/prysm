/*
Package detection defines a service that reacts to incoming blocks/attestations
by running slashing detection for double proposals, double votes, and surround votes
according to the eth2 specification. As soon as slashing objects are found, they are
sent over a feed for the beaconclient service to submit to a beacon node via gRPC.
*/
package detection

import (
	"context"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"go.opencensus.io/trace"
)

// detectIncomingBlocks subscribes to an event feed for
// block objects from a notifier interface. Upon receiving
// a signed beacon block from the feed, we run proposer slashing
// detection on the block.
func (ds *Service) detectIncomingBlocks(ctx context.Context, ch chan *ethpb.SignedBeaconBlock) {
	ctx, span := trace.StartSpan(ctx, "detection.detectIncomingBlocks")
	defer span.End()
	sub := ds.notifier.BlockFeed().Subscribe(ch)
	defer sub.Unsubscribe()
	for {
		select {
		case <-ch:
			log.Debug("Running detection on block...")
			// TODO(#4836): Run detection function for proposer slashings.
		case <-sub.Err():
			log.Error("Subscriber closed, exiting goroutine")
			return
		case <-ctx.Done():
			log.Error("Context canceled")
			return
		}
	}
}

// detectIncomingAttestations subscribes to an event feed for
// attestation objects from a notifier interface. Upon receiving
// an attestation from the feed, we run surround vote and double vote
// detection on the attestation.
func (ds *Service) detectIncomingAttestations(ctx context.Context, ch chan *ethpb.IndexedAttestation) {
	ctx, span := trace.StartSpan(ctx, "detection.detectIncomingAttestations")
	defer span.End()
	sub := ds.notifier.AttestationFeed().Subscribe(ch)
	defer sub.Unsubscribe()
	for {
		select {
		case indexedAtt := <-ch:
			log.Debug("Running detection on attestation...")
			slashings, err := ds.detectAttesterSlashings(ctx, indexedAtt)
			if err != nil {
				log.WithError(err).Error("Could not detect attester slashings")
				continue
			}
			ds.submitAttesterSlashings(ctx, slashings)
		case <-sub.Err():
			log.Error("Subscriber closed, exiting goroutine")
			return
		case <-ctx.Done():
			log.Error("Context canceled")
			return
		}
	}
}
