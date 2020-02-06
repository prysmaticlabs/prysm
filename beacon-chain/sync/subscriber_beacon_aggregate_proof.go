package sync

import (
	"context"
	"errors"
	"fmt"

	"github.com/gogo/protobuf/proto"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
)

// beaconAggregateProofSubscriber forwards the incoming validated aggregated attestation and proof to the
// attestation pool for processing.
func (r *Service) beaconAggregateProofSubscriber(ctx context.Context, msg proto.Message) error {
	a, ok := msg.(*ethpb.AggregateAttestationAndProof)
	if !ok {
		return fmt.Errorf("message was not type *eth.AggregateAttestationAndProof, type=%T", msg)
	}

	if !featureconfig.Get().DisableStrictAttestationPubsubVerification && !r.chain.IsValidAttestation(ctx, a.Aggregate) {
		return errors.New("invalid attestation")
	}

	return r.attPool.SaveAggregatedAttestation(a.Aggregate)
}
