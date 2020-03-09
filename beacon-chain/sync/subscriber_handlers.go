package sync

import (
	"context"
	"fmt"
	"reflect"

	"github.com/gogo/protobuf/proto"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
)

func (r *Service) voluntaryExitSubscriber(ctx context.Context, msg proto.Message) error {
	s, err := r.chain.HeadState(ctx)
	if err != nil {
		return err
	}
	va, ok := msg.(*ethpb.SignedVoluntaryExit)
	if !ok {
		return fmt.Errorf("wrong type fo susbscriber. expected: *ethpb.SignedVoluntaryExit got: %s", reflect.TypeOf(msg).String())
	}
	r.exitPool.InsertVoluntaryExit(ctx, s, va)
	return nil
}

func (r *Service) attesterSlashingSubscriber(ctx context.Context, msg proto.Message) error {
	s, err := r.chain.HeadState(ctx)
	if err != nil {
		return err
	}
	as, ok := msg.(*ethpb.AttesterSlashing)
	if !ok {
		return fmt.Errorf("wrong type fo susbscriber. expected: *ethpb.AttesterSlashing got: %s", reflect.TypeOf(msg).String())
	}
	r.slashingPool.InsertAttesterSlashing(s, as)
	return nil
}

func (r *Service) proposerSlashingSubscriber(ctx context.Context, msg proto.Message) error {
	s, err := r.chain.HeadState(ctx)
	if err != nil {
		return err
	}
	ps, ok := msg.(*ethpb.ProposerSlashing)
	if !ok {
		return fmt.Errorf("wrong type fo susbscriber. expected: *ethpb.ProposerSlashing got: %s", reflect.TypeOf(msg).String())
	}
	r.slashingPool.InsertProposerSlashing(s, ps)
	return nil
}
