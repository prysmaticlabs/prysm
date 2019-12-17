package sync

import (
	"context"

	"github.com/gogo/protobuf/proto"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
)

// beaconAggregateProofSubscriber forwards the incoming validated aggregated attestation and proof to the
// attestation pool for processing.
func (r *RegularSync) beaconAggregateProofSubscriber(ctx context.Context, msg proto.Message) error {
	return r.attPool.SaveUnaggregatedAttestation(msg.(*ethpb.Attestation))
}
