package sync

import (
	"context"

	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"

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
	as := msg.(*ethpb.AttesterSlashing)
	s, err := r.db.State(ctx, bytesutil.ToBytes32(as.Attestation_1.Data.BeaconBlockRoot))
	if err != nil {
		return err
	}
	r.slashingPool.InsertAttesterSlashing(ctx, s, as)
	return nil
}

func (r *Service) proposerSlashingSubscriber(ctx context.Context, msg proto.Message) error {
	ps := msg.(*ethpb.ProposerSlashing)
	root, err := ssz.HashTreeRoot(ps.Header_1.Header)
	s, err := r.db.State(ctx, root)
	if err != nil {
		return err
	}
	r.slashingPool.InsertProposerSlashing(ctx, s, ps)
	return nil
}
