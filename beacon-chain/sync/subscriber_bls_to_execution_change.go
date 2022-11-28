package sync

import (
	"context"

	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"google.golang.org/protobuf/proto"
)

func (s *Service) blsToExecutionChangeSubscriber(ctx context.Context, msg proto.Message) error {
	blsMsg, ok := msg.(*ethpb.SignedBLSToExecutionChange)
	if !ok {
		return errors.Errorf("incorrect type of message received, wanted %T but got %T", &ethpb.SignedBLSToExecutionChange{}, msg)
	}
	s.cfg.blsToExecPool.InsertBLSToExecChange(blsMsg)
	return nil
}
