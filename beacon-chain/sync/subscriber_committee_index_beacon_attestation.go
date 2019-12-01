package sync

import (
	"context"

	"github.com/gogo/protobuf/proto"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
)

func (r *RegularSync) committeeIndexBeaconAttestationSubscriber(ctx context.Context, msg proto.Message) error {

	return nil
}

func (r *RegularSync) currentCommitteeIndex() int {
	state, err := r.chain.HeadState(context.Background())
	if err != nil {
		panic(err)
	}
	count, err := helpers.CommitteeCountAtSlot(state, helpers.StartSlot(helpers.CurrentEpoch(state)))
	if err != nil {
		panic(err)
	}
	return int(count)
}