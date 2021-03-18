package slasher

import (
	"context"

	"github.com/pkg/errors"
	types "github.com/prysmaticlabs/eth2-types"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	slashertypes "github.com/prysmaticlabs/prysm/beacon-chain/slasher/types"
	"go.opencensus.io/trace"
)

// detectProposerSlashings takes in signed block header wrappers and returns a list of proposer slashings detected.
func (s *Service) detectProposerSlashings(
	ctx context.Context,
	proposedBlocks []*slashertypes.SignedBlockHeaderWrapper,
) ([]*ethpb.ProposerSlashing, error) {
	ctx, span := trace.StartSpan(ctx, "Slasher.detectProposerSlashings")
	defer span.End()
	// We check if there are any slashable double proposals in the input list
	// of proposals with respect to each other.
	existingProposals := make(map[string]*slashertypes.SignedBlockHeaderWrapper)
	for i, proposal := range proposedBlocks {
		key := proposalKey(proposal)
		existingProposal, ok := existingProposals[key]
		if !ok {
			existingProposals[key] = proposal
			continue
		}
		if isDoubleProposal(proposedBlocks[i].SigningRoot, existingProposal.SigningRoot) {
			doubleProposalsTotal.Inc()
			slashing := &ethpb.ProposerSlashing{
				Header_1: existingProposal.SignedBeaconBlockHeader,
				Header_2: proposedBlocks[i].SignedBeaconBlockHeader,
			}
			s.serviceCfg.ProposerSlashingsFeed.Send(slashing)
		}
	}

	proposerSlashings, err := s.serviceCfg.Database.CheckDoubleBlockProposals(ctx, proposedBlocks)
	if err != nil {
		return nil, errors.Wrap(err, "could not check for double proposals on disk")
	}
	if err := s.saveSafeProposals(ctx, proposedBlocks, proposerSlashings); err != nil {
		return nil, errors.Wrap(err, "could not save safe proposals")
	}
	return proposerSlashings, nil
}

// Check for double proposals in our database given a list of incoming block proposals.
// For the proposals that were not slashable, we save them to the database.
func (s *Service) saveSafeProposals(
	ctx context.Context,
	proposedBlocks []*slashertypes.SignedBlockHeaderWrapper,
	proposerSlashings []*ethpb.ProposerSlashing,
) error {
	ctx, span := trace.StartSpan(ctx, "Slasher.saveSafeProposals")
	defer span.End()
	safeProposals := safeProposals(proposedBlocks, proposerSlashings)
	return s.serviceCfg.Database.SaveBlockProposals(ctx, safeProposals)
}

func (s *Service) recordDoubleProposals(doubleProposals []*ethpb.ProposerSlashing) {
	for _, doubleProposal := range doubleProposals {
		doubleProposalsTotal.Inc()
		logProposerSlashing(doubleProposal)
		s.serviceCfg.ProposerSlashingsFeed.Send(doubleProposal)
	}
}

func safeProposals(
	proposedBlocks []*slashertypes.SignedBlockHeaderWrapper,
	proposerSlashings []*ethpb.ProposerSlashing,
) []*slashertypes.SignedBlockHeaderWrapper {
	// We initialize a map of proposers that are safe from slashing.
	safeProposers := make(map[types.ValidatorIndex]*slashertypes.SignedBlockHeaderWrapper, len(proposedBlocks))
	for _, proposal := range proposedBlocks {
		safeProposers[proposal.SignedBeaconBlockHeader.Header.ProposerIndex] = proposal
	}
	for _, doubleProposal := range proposerSlashings {
		// If a proposer is found to have committed a slashable offense, we delete
		// them from the safe proposers map.
		delete(safeProposers, doubleProposal.Header_1.Header.ProposerIndex)
	}
	// We save all the proposals that are determined "safe" and not-slashable to our database.
	safeProposals := make([]*slashertypes.SignedBlockHeaderWrapper, 0, len(safeProposers))
	for _, proposal := range safeProposers {
		safeProposals = append(safeProposals, proposal)
	}
	return safeProposals
}

func proposalKey(proposal *slashertypes.SignedBlockHeaderWrapper) string {
	header := proposal.SignedBeaconBlockHeader.Header
	return uintToString(uint64(header.Slot)) + ":" + uintToString(uint64(header.ProposerIndex))
}
