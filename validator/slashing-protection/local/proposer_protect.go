package local

import (
	"bytes"
	"context"

	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/params"
	slashingprotection "github.com/prysmaticlabs/prysm/validator/slashing-protection"
)

// IsSlashableBlock determines if an incoming block is slashable
// according to local protection. Then, if the block successfully passes checks,
// we update our local proposals history accordingly.
func (s *Service) IsSlashableBlock(
	ctx context.Context, block *ethpb.SignedBeaconBlock, pubKey [48]byte, signingRoot [32]byte,
) (bool, error) {
	if block == nil || block.Block == nil {
		return false, errors.New("received nil block")
	}
	// Check if the block is slashable by local and remote slashing protection
	existingSigningRoot, minSlot, err := s.validatorDB.ProposalHistoryForSlot(ctx, pubKey[:], block.Block.Slot)
	if err != nil {
		return false, errors.Wrap(err, "failed to get proposal history")
	}
	// Dont allow blocks with minimal slot lower then the minimal imported slot to be signed.
	if block.Block.Slot < minSlot {
		slashingprotection.LocalSlashableProposalsTotal.Inc()
		return true, nil
	}
	// Check if we are performing a double block proposal.
	if existingSigningRoot != nil && !bytes.Equal(existingSigningRoot, params.BeaconConfig().ZeroHash[:]) {
		slashingprotection.LocalSlashableProposalsTotal.Inc()
		return true, nil
	}
	// If the block is not slashable, we perform an immediate write to our DB.
	if err := s.validatorDB.SaveProposalHistoryForSlot(ctx, pubKey[:], block.Block.Slot, signingRoot[:]); err != nil {
		return false, errors.Wrap(err, "failed to save updated proposal history")
	}
	return false, nil
}
