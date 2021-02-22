package slasher

import (
	"context"

	types "github.com/prysmaticlabs/eth2-types"
	slashertypes "github.com/prysmaticlabs/prysm/beacon-chain/slasher/types"
	"go.opencensus.io/trace"
)

// Given a list of blocks, check if they are slashable for the validators involved.
func (s *Service) detectSlashableBlocks(
	ctx context.Context,
	proposedBlocks []*slashertypes.CompactBeaconBlock,
) error {
	ctx, span := trace.StartSpan(ctx, "Slasher.detectSlashableBlocks")
	defer span.End()
	// We check if there are any slashable double proposals in the input list
	// of proposals with respect to each other.
	existingProposals := make(map[string][32]byte)
	for i, proposal := range proposedBlocks {
		key := uintToString(uint64(proposal.Slot)) + ":" + uintToString(uint64(proposal.ProposerIndex))
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

// Check for double proposals in our database given a list of incoming block proposals.
// For the proposals that were not slashable, we save them to the database.
func (s *Service) checkDoubleProposalsOnDisk(
	ctx context.Context, proposedBlocks []*slashertypes.CompactBeaconBlock,
) error {
	ctx, span := trace.StartSpan(ctx, "Slasher.checkDoubleProposalsOnDisk")
	defer span.End()
	doubleProposals, err := s.serviceCfg.Database.CheckDoubleBlockProposals(ctx, proposedBlocks)
	if err != nil {
		return err
	}
	// We initialize a map of proposers that are safe from slashing.
	safeProposers := make(map[types.ValidatorIndex]*slashertypes.CompactBeaconBlock, len(proposedBlocks))
	for _, proposal := range proposedBlocks {
		safeProposers[proposal.ProposerIndex] = proposal
	}
	for i, doubleProposal := range doubleProposals {
		logDoubleProposal(proposedBlocks[i], doubleProposal.ExistingSigningRoot)
		// If a proposer is found to have committed a slashable offense, we delete
		// them from the safe proposers map.
		delete(safeProposers, doubleProposal.ProposerIndex)
	}
	// We save all the proposals that are determined "safe" and not-slashable to our database.
	safeProposals := make([]*slashertypes.CompactBeaconBlock, 0, len(safeProposers))
	for _, proposal := range safeProposers {
		safeProposals = append(safeProposals, proposal)
	}
	return s.serviceCfg.Database.SaveBlockProposals(ctx, safeProposals)
}
