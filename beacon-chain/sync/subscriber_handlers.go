package sync

import (
	"context"
	"fmt"

	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"
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
		s, err := r.db.State(ctx, bytesutil.ToBytes32(as.Attestation_1.Data.BeaconBlockRoot))
		if err != nil {
			return err
		}
		if s == nil {
			return fmt.Errorf("no state found for block root %#x", as.Attestation_1.Data.BeaconBlockRoot)
		}
		if err := r.slashingPool.InsertAttesterSlashing(ctx, s, as); err != nil {
			return errors.Wrap(err, "could not insert attester slashing into pool")
		}
		r.setAttesterSlashingIndicesSeen(as.Attestation_1.AttestingIndices, as.Attestation_2.AttestingIndices)
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
		root, err := stateutil.BlockHeaderRoot(ps.Header_1.Header)
		s, err := r.db.State(ctx, root)
		if err != nil {
			return err
		}
		if s == nil {
			return fmt.Errorf("no state found for block root %#x", root)
		}
		if err := r.slashingPool.InsertProposerSlashing(ctx, s, ps); err != nil {
			return errors.Wrap(err, "could not insert proposer slashing into pool")
		}
		r.setProposerSlashingIndexSeen(ps.Header_1.Header.ProposerIndex)
	}
	return nil
}
