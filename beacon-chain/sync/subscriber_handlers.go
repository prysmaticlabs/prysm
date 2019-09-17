package sync

import (
	"context"
	)

func (r *RegularSync) voluntaryExitSubscriber(ctx context.Context, msg interface{}) error {
	return r.operations.HandleValidatorExits(ctx, msg)
}

func (r *RegularSync) attesterSlashingSubscriber(ctx context.Context, msg interface{}) error {
	// TODO(#3259): Requires handlers in operations service to be implemented.
	return nil
}

func (r *RegularSync) proposerSlashingSubscriber(ctx context.Context, msg interface{}) error {
	// TODO(#3259): Requires handlers in operations service to be implemented.
	return nil
}
