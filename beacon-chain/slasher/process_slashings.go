package slasher

import (
	"context"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	slashertypes "github.com/prysmaticlabs/prysm/beacon-chain/slasher/types"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
)

func (s *Service) processSlashings(ctx context.Context, slashings []*slashertypes.Slashing) {
	for _, sl := range slashings {
		if sl.Kind == slashertypes.NotSlashable {
			continue
		}
		if isAttesterSlashingKind(sl.Kind) {
			if err := s.verifyAttSignature(ctx, sl.Attestation); err != nil {
				continue
			}
			if err := s.verifyAttSignature(ctx, sl.PrevAttestation); err != nil {
				continue
			}
		}
		if isProposerSlashingKind(sl.Kind) {
			if err := s.verifyBlockSignature(ctx, sl.BeaconBlock); err != nil {
				continue
			}
			if err := s.verifyBlockSignature(ctx, sl.PrevBeaconBlock); err != nil {
				continue
			}
		}
		// Log the slashing event.
		logSlashingEvent(sl)

		// Submit to the slashing operations pool.
		// TODO: Implement.
	}
}

func (s *Service) verifyBlockSignature(ctx context.Context) error {
	parentState, err := s.serviceCfg.StateByRoot(ctx, bytesutil.ToBytes32(blk.Block.ParentRoot))
	if err != nil {
		return err
	}
	return blocks.VerifyBlockSignature(parentState, blk)
}

func (s *Service) verifyAttSignature(ctx context.Context, att *ethpb.IndexedAttestation) error {
	preState, err := s.serviceCfg.StateFetcher.AttestationPreState(ctx, att.Data.Target)
	if err != nil {
		return err
	}
	return blocks.VerifyIndexedAttestation(ctx, preState, att)
}

func isProposerSlashingKind(kind slashertypes.SlashingKind) bool {
	return kind == slashertypes.DoubleProposal
}

func isAttesterSlashingKind(kind slashertypes.SlashingKind) bool {
	return kind == slashertypes.SurroundedVote || kind == slashertypes.SurroundingVote || kind == slashertypes.DoubleVote
}
