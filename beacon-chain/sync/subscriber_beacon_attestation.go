package sync

import (
	"context"
	"fmt"

	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	eth "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/cache"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/feed"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/feed/operation"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/sliceutil"
)

func (s *Service) committeeIndexBeaconAttestationSubscriber(ctx context.Context, msg proto.Message) error {
	a, ok := msg.(*eth.Attestation)
	if !ok {
		return fmt.Errorf("message was not type *eth.Attestation, type=%T", msg)
	}

	if a.Data == nil {
		return errors.New("nil attestation")
	}
	s.setSeenCommitteeIndicesEpoch(a.Data.Target.Epoch, a.Data.CommitteeIndex, a.AggregationBits)

	exists, err := s.attPool.HasAggregatedAttestation(a)
	if err != nil {
		return errors.Wrap(err, "failed to determine if attestation pool has this atttestation")
	}
	if exists {
		return nil
	}

	// Broadcast the unaggregated attestation on a feed to notify other services in the beacon node
	// of a received unaggregated attestation.
	s.attestationNotifier.OperationFeed().Send(&feed.Event{
		Type: operation.UnaggregatedAttReceived,
		Data: &operation.UnAggregatedAttReceivedData{
			Attestation: a,
		},
	})

	return s.attPool.SaveUnaggregatedAttestation(a)
}

func (s *Service) persistentSubnetIndices() []uint64 {
	return cache.SubnetIDs.GetAllSubnets()
}

func (s *Service) aggregatorSubnetIndices(currentSlot uint64) []uint64 {
	endEpoch := helpers.SlotToEpoch(currentSlot) + 1
	endSlot := endEpoch * params.BeaconConfig().SlotsPerEpoch
	commIds := []uint64{}
	for i := currentSlot; i <= endSlot; i++ {
		commIds = append(commIds, cache.SubnetIDs.GetAggregatorSubnetIDs(i)...)
	}
	return sliceutil.SetUint64(commIds)
}

func (s *Service) attesterSubnetIndices(currentSlot uint64) []uint64 {
	endEpoch := helpers.SlotToEpoch(currentSlot) + 1
	endSlot := endEpoch * params.BeaconConfig().SlotsPerEpoch
	commIds := []uint64{}
	for i := currentSlot; i <= endSlot; i++ {
		commIds = append(commIds, cache.SubnetIDs.GetAttesterSubnetIDs(i)...)
	}
	return sliceutil.SetUint64(commIds)
}
