package sync

import (
	"context"

	"github.com/gogo/protobuf/proto"
)

func (s *RegularSync) voluntaryExitSubscriber(ctx context.Context, msg proto.Message) error {
	return s.operations.HandleValidatorExits(ctx, msg)
}

func (s *RegularSync) attesterSlashingSubscriber(ctx context.Context, msg proto.Message) error {
	// TODO(#3259): Requires handlers in operations service to be implemented.
	return nil
}
