package sync

import (
	"context"
	"fmt"

	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
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
	aSlashing, ok := msg.(*ethpb.AttesterSlashing)
	if !ok {
		return fmt.Errorf("wrong type, expected: *ethpb.AttesterSlashing got: %T", msg)
	}
	// Do some nil checks to prevent easy DoS'ing of this handler.
	aSlashing1IsNil := aSlashing == nil || aSlashing.Attestation_1 == nil || aSlashing.Attestation_1.AttestingIndices == nil
	aSlashing2IsNil := aSlashing == nil || aSlashing.Attestation_2 == nil || aSlashing.Attestation_2.AttestingIndices == nil
	if !aSlashing1IsNil && !aSlashing2IsNil {
		headState, err := r.chain.HeadState(ctx)
		if err != nil {
			return err
		}
		if err := r.slashingPool.InsertAttesterSlashing(ctx, headState, aSlashing); err != nil {
			return errors.Wrap(err, "could not insert attester slashing into pool")
		}
		r.setAttesterSlashingIndicesSeen(aSlashing.Attestation_1.AttestingIndices, aSlashing.Attestation_2.AttestingIndices)
	}
	return nil
}

func (r *Service) proposerSlashingSubscriber(ctx context.Context, msg proto.Message) error {
	pSlashing, ok := msg.(*ethpb.ProposerSlashing)
	if !ok {
		return fmt.Errorf("wrong type, expected: *ethpb.ProposerSlashing got: %T", msg)
	}
	// Do some nil checks to prevent easy DoS'ing of this handler.
	header1IsNil := pSlashing == nil || pSlashing.Header_1 == nil || pSlashing.Header_1.Header == nil
	header2IsNil := pSlashing == nil || pSlashing.Header_2 == nil || pSlashing.Header_2.Header == nil
	if !header1IsNil && !header2IsNil {
		headState, err := r.chain.HeadState(ctx)
		if err != nil {
			return err
		}
		if err := r.slashingPool.InsertProposerSlashing(ctx, headState, pSlashing); err != nil {
			return errors.Wrap(err, "could not insert proposer slashing into pool")
		}
		r.setProposerSlashingIndexSeen(pSlashing.Header_1.Header.ProposerIndex)
	} else {
		return errors.New(fmt.Sprintf("h1: %t, h2: %t", header1IsNil, header2IsNil))
	}
	return nil
}
