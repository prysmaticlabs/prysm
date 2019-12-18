package sync

import (
	"context"
	"fmt"

	"github.com/gogo/protobuf/proto"
	eth "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
)

func (r *Service) committeeIndexBeaconAttestationSubscriber(ctx context.Context, msg proto.Message) error {
	a, ok := msg.(*eth.Attestation)
	if !ok {
		return fmt.Errorf("message was not type *eth.Attestation, type=%T", msg)
	}
	return r.attPool.SaveUnaggregatedAttestation(a)
}

func (r *Service) currentCommitteeIndex() int {
	state, err := r.chain.HeadState(context.Background())
	if err != nil {
		panic(err)
	}
	if state == nil {
		return 0
	}
	count, err := helpers.CommitteeCountAtSlot(state, helpers.StartSlot(helpers.CurrentEpoch(state)))
	if err != nil {
		panic(err)
	}
	return int(count)
}
