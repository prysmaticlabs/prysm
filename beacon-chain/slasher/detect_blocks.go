package slasher

import (
	"context"
	"fmt"

	slashertypes "github.com/prysmaticlabs/prysm/beacon-chain/slasher/types"
)

// Given a list of blocks, check if they are slashable for the validators involved.
func (s *Service) detectSlashableBlocks(
	ctx context.Context,
	proposedBlocks []*slashertypes.CompactBeaconBlock,
) error {
	// We check if there are any slashable double proposals in the input list
	// of proposals with respect to each other.
	existingProposals := make(map[string][32]byte)
	for i, proposal := range proposedBlocks {
		key := fmt.Sprintf("%d:%d", proposal.Slot, proposal.ProposerIndex)
		existingSigningRoot, ok := existingProposals[key]
		if !ok {
			existingProposals[key] = proposal.SigningRoot
			continue
		}
		if isDoubleProposal(proposedBlocks[i].SigningRoot, existingSigningRoot) {
			logDoubleProposal(proposedBlocks[i], existingSigningRoot)
		}
	}
	// We check if there are any slashable double proposals in the input list
	// of proposals with respect to our database.
	return s.checkDoubleProposalsOnDisk(ctx, proposedBlocks)
}

func (s *Service) checkDoubleProposalsOnDisk(
	ctx context.Context, proposedBlocks []*slashertypes.CompactBeaconBlock,
) error {
	existingProposals, err := s.serviceCfg.Database.ExistingBlockProposals(ctx, proposedBlocks)
	if err != nil {
		return err
	}
	newBlockProposals := make([]*slashertypes.CompactBeaconBlock, 0)
	for i, existing := range existingProposals {
		if existing == nil {
			continue
		}
		if isDoubleProposal(proposedBlocks[i].SigningRoot, existing.ExistingSigningRoot) {
			logDoubleProposal(proposedBlocks[i], existing.ExistingSigningRoot)
		} else {
			// If there is no existing block proposal, we append to a list of confirmed, new proposals.
			newBlockProposals = append(newBlockProposals, proposedBlocks[i])
		}
	}
	return s.serviceCfg.Database.SaveBlockProposals(ctx, newBlockProposals)
}
