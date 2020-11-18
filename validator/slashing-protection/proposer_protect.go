package slashingprotection

import (
	"bytes"
	"context"
	"fmt"

	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// IsSlashableBlock checks if signed beacon block is slashable against
// a validator's slashing protection history and against a remote slashing protector if enabled.
func (s *Service) IsSlashableBlock(
	ctx context.Context, block *ethpb.SignedBeaconBlock, pubKey [48]byte, domain *ethpb.DomainResponse,
) (bool, error) {
	if block == nil || block.Block == nil {
		return false, errors.New("received nil block")
	}
	metricsKey := fmt.Sprintf("%#x", pubKey)
	existingSigningRoot, err := s.validatorDB.ProposalHistoryForSlot(ctx, pubKey[:], block.Block.Slot)
	if err != nil {
		return false, errors.Wrap(err, "failed to get proposal history")
	}
	// Check if the block is slashable by local and remote slashing protection
	if s.remoteProtector != nil {
		slashable, err := s.remoteProtector.IsSlashableBlock(ctx, block, pubKey, domain)
		if err != nil {
			return false, errors.Wrap(err, "failed to get proposal history")
		}
		remoteSlashableProposalsTotal.WithLabelValues(metricsKey).Inc()
		return slashable, nil
	}
	if !bytes.Equal(existingSigningRoot, params.BeaconConfig().ZeroHash[:]) {
		localSlashableProposalsTotal.WithLabelValues(metricsKey).Inc()
		return true, nil
	}
	signingRoot, err := helpers.ComputeSigningRoot(block.Block, domain.SignatureDomain)
	if err != nil {
		return false, errors.Wrap(err, "failed to compute signing root for block")
	}
	// If the block is not slashable, we perform an immediate write to our DB.
	if err := s.validatorDB.SaveProposalHistoryForSlot(ctx, pubKey[:], block.Block.Slot, signingRoot[:]); err != nil {
		return false, errors.Wrap(err, "failed to save updated proposal history")
	}
	return false, nil
}
