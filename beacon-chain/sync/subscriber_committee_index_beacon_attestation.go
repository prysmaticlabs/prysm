package sync

import (
	"context"
	"fmt"

	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	eth "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
)

func (r *Service) committeeIndexBeaconAttestationSubscriber(ctx context.Context, msg proto.Message) error {
	a, ok := msg.(*eth.Attestation)
	if !ok {
		return fmt.Errorf("message was not type *eth.Attestation, type=%T", msg)
	}

	if exists, _  := r.attPool.HasAggregatedAttestation(a); exists {
		return nil
	}

	if !featureconfig.Get().DisableStrictAttestationPubsubVerification && !r.chain.IsValidAttestation(ctx, a) {
		return errors.New("invalid attestation")
	}

	return r.attPool.SaveUnaggregatedAttestation(a)
}

func (r *Service) currentCommitteeIndex() int {
	activeValidatorIndices, err := r.chain.HeadValidatorsIndices(helpers.SlotToEpoch(r.chain.HeadSlot()))
	if err != nil {
		panic(err)
	}
	return int(helpers.SlotCommitteeCount(uint64(len(activeValidatorIndices))))
}
