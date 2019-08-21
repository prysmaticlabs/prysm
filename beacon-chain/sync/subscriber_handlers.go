package sync

import (
	"context"

	"github.com/gogo/protobuf/proto"
)

func (s *RegularSync) voluntaryExitSubscriber(ctx context.Context, msg proto.Message) error {
	return s.operations.HandleValidatorExits(ctx, msg)
}
