package sync

import (
	"context"
	"fmt"

	"github.com/gogo/protobuf/proto"
	eth "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/feed"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/feed/operation"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
)

func (r *Service) committeeIndexBeaconAttestationSubscriber(ctx context.Context, msg proto.Message) error {
	a, ok := msg.(*eth.Attestation)
	if !ok {
		return fmt.Errorf("message was not type *eth.Attestation, type=%T", msg)
	}

	if exists, _ := r.attPool.HasAggregatedAttestation(a); exists {
		return nil
	}

	// Broadcast the unaggregated attestation on a feed to notify other services in the beacon node
	// of a received unaggregated attestation.
	r.attestationNotifier.OperationFeed().Send(&feed.Event{
		Type: operation.UnaggregatedAttReceived,
		Data: &operation.UnAggregatedAttReceivedData{
			Attestation: a,
		},
	})

	return r.attPool.SaveUnaggregatedAttestation(a)
}

func (r *Service) currentCommitteeIndex() int {
	activeValidatorIndices, err := r.chain.HeadValidatorsIndices(helpers.SlotToEpoch(r.chain.HeadSlot()))
	if err != nil {
		panic(err)
	}
	return int(helpers.SlotCommitteeCount(uint64(len(activeValidatorIndices))))
}
