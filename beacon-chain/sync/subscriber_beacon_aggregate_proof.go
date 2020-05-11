package sync

import (
	"context"
	"errors"
	"fmt"

	"github.com/gogo/protobuf/proto"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/feed"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/feed/operation"
)

// beaconAggregateProofSubscriber forwards the incoming validated aggregated attestation and proof to the
// attestation pool for processing.
func (r *Service) beaconAggregateProofSubscriber(ctx context.Context, msg proto.Message) error {
	a, ok := msg.(*ethpb.SignedAggregateAttestationAndProof)
	if !ok {
		return fmt.Errorf("message was not type *eth.SignedAggregateAttestationAndProof, type=%T", msg)
	}

	if a.Message.Aggregate == nil || a.Message.Aggregate.Data == nil {
		return errors.New("nil aggregate")
	}
	r.setAggregatorIndexEpochSeen(a.Message.Aggregate.Data.Target.Epoch, a.Message.AggregatorIndex)

	// Broadcast the aggregated attestation on a feed to notify other services in the beacon node
	// of a received aggregated attestation.
	r.attestationNotifier.OperationFeed().Send(&feed.Event{
		Type: operation.AggregatedAttReceived,
		Data: &operation.AggregatedAttReceivedData{
			Attestation: a.Message,
		},
	})

	return r.attPool.SaveAggregatedAttestation(a.Message.Aggregate)
}
