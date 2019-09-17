package sync

import (
	"context"

	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
)

// beaconAttestationSubscriber forwards the incoming validated attestation to the blockchain
// service for processing.
func (r *RegularSync) beaconAttestationSubscriber(ctx context.Context, msg interface{}) error {
	if err := r.operations.HandleAttestation(ctx, msg.(*ethpb.Attestation)); err != nil {
		return err
	}

	return r.chain.ReceiveAttestationNoPubsub(ctx, msg.(*ethpb.Attestation))
}
