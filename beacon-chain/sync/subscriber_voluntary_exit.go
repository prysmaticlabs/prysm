package sync

import (
	"context"

	"github.com/gogo/protobuf/proto"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
)

func (s *RegularSync) voluntaryExitSubscriber(ctx context.Context, msg proto.Message) error {
	exit := msg.(*ethpb.VoluntaryExit)

	return s.operations.HandleValidatorExits(ctx, exit)
}
