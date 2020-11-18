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

var (
	ErrSlashableBlock       = errors.New("attempted a double proposal, block rejected by local slashing protection")
	ErrRemoteSlashableBlock = errors.New("attempted a double proposal, block rejected by remote slashing protection")
)

// IsSlashableBlock checks if signed beacon block is slashable against
// a validator's slashing protection history and against a remote slashing protector if enabled.
func (s *Service) IsSlashableBlock(
	ctx context.Context, block *ethpb.SignedBeaconBlock, pubKey [48]byte, domain *ethpb.DomainResponse,
) error {
	if block == nil || block.Block == nil {
		return errors.New("received nil block")
	}
	metricsKey := fmt.Sprintf("%#x", pubKey)
	existingSigningRoot, err := s.validatorDB.ProposalHistoryForSlot(ctx, pubKey[:], block.Block.Slot)
	if err != nil {
		return errors.Wrap(err, "failed to get proposal history")
	}
	// Check if the block is slashable by local and remote slashing protection
	if s.remoteProtector != nil && s.remoteProtector.IsSlashableBlock(ctx, block) {
		remoteSlashableProposalsTotal.WithLabelValues(metricsKey).Inc()
		return ErrRemoteSlashableBlock
	}
	if !bytes.Equal(existingSigningRoot, params.BeaconConfig().ZeroHash[:]) {
		localSlashableProposalsTotal.WithLabelValues(metricsKey).Inc()
		return ErrSlashableBlock
	}
	signingRoot, err := helpers.ComputeSigningRoot(block.Block, domain.SignatureDomain)
	if err != nil {
		return errors.Wrap(err, "failed to compute signing root for block")
	}
	// If the block is not slashable, we perform an immediate write to our DB.
	if err := s.validatorDB.SaveProposalHistoryForSlot(ctx, pubKey[:], block.Block.Slot, signingRoot[:]); err != nil {
		return errors.Wrap(err, "failed to save updated proposal history")
	}
	return nil
}
