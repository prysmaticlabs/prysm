package sync

import (
	"context"

	"github.com/gogo/protobuf/proto"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
)

func (r *Service) voluntaryExitSubscriber(ctx context.Context, msg proto.Message) error {
	s, err := r.chain.HeadState(ctx)
	if err != nil {
		return err
	}
	r.exitPool.InsertVoluntaryExit(ctx, s, msg.(*ethpb.SignedVoluntaryExit))
	return nil
}

func (r *Service) attesterSlashingSubscriber(ctx context.Context, msg proto.Message) error {
	// TODO(#3259): Requires handlers in operations service to be implemented.
	return nil
}

func (r *Service) proposerSlashingSubscriber(ctx context.Context, msg proto.Message) error {
	// TODO(#3259): Requires handlers in operations service to be implemented.
	return nil
}
