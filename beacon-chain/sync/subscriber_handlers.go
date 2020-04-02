package sync

import (
	"context"
	"errors"
	"fmt"

	"github.com/gogo/protobuf/proto"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
)

func (r *Service) voluntaryExitSubscriber(ctx context.Context, msg proto.Message) error {
	ve, ok := msg.(*ethpb.SignedVoluntaryExit)
	if !ok {
		return fmt.Errorf("wrong type, expected: *ethpb.SignedVoluntaryExit got: %T", msg)
	}

	if ve.Exit == nil {
		return errors.New("exit can't be nil")
	}
	r.setExitIndexSeen(ve.Exit.ValidatorIndex)

	s, err := r.chain.HeadState(ctx)
	if err != nil {
		return err
	}
	r.exitPool.InsertVoluntaryExit(ctx, s, ve)
	return nil
}

func (r *Service) attesterSlashingSubscriber(ctx context.Context, msg proto.Message) error {
	as, ok := msg.(*ethpb.AttesterSlashing)
	if !ok {
		return fmt.Errorf("wrong type, expected: *ethpb.AttesterSlashing got: %T", msg)
	}
	// Do some nil checks to prevent easy DoS'ing of this handler.
	if as != nil && as.Attestation_1 != nil && as.Attestation_1.Data != nil {
		r.setAttesterSlashingIndicesSeen(as.Attestation_1.AttestingIndices, as.Attestation_2.AttestingIndices)

		s, err := r.db.State(ctx, bytesutil.ToBytes32(as.Attestation_1.Data.BeaconBlockRoot))
		if err != nil {
			return err
		}
		if s == nil {
			return fmt.Errorf("no state found for block root %#x", as.Attestation_1.Data.BeaconBlockRoot)
		}
		return r.slashingPool.InsertAttesterSlashing(ctx, s, as)
	}
	return nil
}

func (r *Service) proposerSlashingSubscriber(ctx context.Context, msg proto.Message) error {
	ps, ok := msg.(*ethpb.ProposerSlashing)
	if !ok {
		return fmt.Errorf("wrong type, expected: *ethpb.ProposerSlashing got: %T", msg)
	}
	// Do some nil checks to prevent easy DoS'ing of this handler.
	if ps.Header_1 != nil && ps.Header_1.Header != nil {
		r.setProposerSlashingIndexSeen(ps.Header_1.Header.ProposerIndex)

		root, err := ssz.HashTreeRoot(ps.Header_1.Header)
		s, err := r.db.State(ctx, root)
		if err != nil {
			return err
		}
		if s == nil {
			return fmt.Errorf("no state found for block root %#x", root)
		}
		return r.slashingPool.InsertProposerSlashing(ctx, s, ps)
	}
	return nil
}
