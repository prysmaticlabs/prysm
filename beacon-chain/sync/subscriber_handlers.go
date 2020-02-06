package sync

import (
	"context"

	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
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
	s, err := r.chain.HeadState(ctx)
	if err != nil {
		return err
	}
	if err := r.slashingPool.InsertAttesterSlashing(s, msg.(*ethpb.AttesterSlashing)); err != nil {
		return errors.Wrap(err, "could not insert attester slashing into pool")
	}
	return nil
}

func (r *Service) proposerSlashingSubscriber(ctx context.Context, msg proto.Message) error {
	s, err := r.chain.HeadState(ctx)
	if err != nil {
		return err
	}
	if err := r.slashingPool.InsertProposerSlashing(s, msg.(*ethpb.ProposerSlashing)); err != nil {
		return errors.Wrap(err, "could not insert proposer slashing into pool")
	}
	return nil
}
