package sync

import (
	"context"

	"github.com/gogo/protobuf/proto"
)

func (r *RegularSync) voluntaryExitSubscriber(ctx context.Context, msg proto.Message) error {
	return r.operations.HandleValidatorExits(ctx, msg)
}

func (r *RegularSync) attesterSlashingSubscriber(ctx context.Context, msg proto.Message) error {
	// TODO(#3259): Requires handlers in operations service to be implemented.
	return nil
}

func (r *RegularSync) proposerSlashingSubscriber(ctx context.Context, msg proto.Message) error {
	// TODO(#3259): Requires handlers in operations service to be implemented.
	return nil
}
