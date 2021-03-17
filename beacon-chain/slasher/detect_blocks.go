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
	ctx context.Context, proposedBlocks []*slashertypes.SignedBlockHeaderWrapper,
) error {
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

	// We check if there are any slashable double proposals in the input list
	// of proposals with respect to our database.
	doubleProposals, err := s.saveSafeProposals(ctx, proposedBlocks)
	if err != nil {
		return err
	}
	s.recordDoubleProposals(doubleProposals)
	return nil
}

// Check for double proposals in our database given a list of incoming block proposals.
// For the proposals that were not slashable, we save them to the database.
func (s *Service) saveSafeProposals(
	ctx context.Context, proposedBlocks []*slashertypes.SignedBlockHeaderWrapper,
) ([]*slashertypes.DoubleBlockProposal, error) {
	ctx, span := trace.StartSpan(ctx, "Slasher.checkDoubleProposalsOnDisk")
	defer span.End()
	safeProposals, doubleProposals, err := s.detectDoubleProposalsOnDisk(ctx, proposedBlocks)
	if err != nil {
		return nil, err
	}
	if err := s.serviceCfg.Database.SaveBlockProposals(ctx, safeProposals); err != nil {
		return nil, err
	}
	return doubleProposals, nil
}

func (s *Service) detectDoubleProposalsOnDisk(
	ctx context.Context, proposedBlocks []*slashertypes.SignedBlockHeaderWrapper,
) (safeProposals []*slashertypes.SignedBlockHeaderWrapper, doubleProposals []*slashertypes.DoubleBlockProposal, err error) {
	ctx, span := trace.StartSpan(ctx, "Slasher.checkDoubleProposalsOnDisk")
	defer span.End()
	doubleProposals, err = s.serviceCfg.Database.CheckDoubleBlockProposals(ctx, proposedBlocks)
	if err != nil {
		return nil, nil, errors.Wrap(err, "could not check for double proposals on disk")
	}
	// We initialize a map of proposers that are safe from slashing.
	safeProposers := make(map[types.ValidatorIndex]*slashertypes.SignedBlockHeaderWrapper, len(proposedBlocks))
	for _, proposal := range proposedBlocks {
		safeProposers[proposal.SignedBeaconBlockHeader.Header.ProposerIndex] = proposal
	}
	for _, doubleProposal := range doubleProposals {
		// If a proposer is found to have committed a slashable offense, we delete
		// them from the safe proposers map.
		delete(safeProposers, doubleProposal.ValidatorIndex)
	}
	// We save all the proposals that are determined "safe" and not-slashable to our database.
	safeProposals = make([]*slashertypes.SignedBlockHeaderWrapper, 0, len(safeProposers))
	for _, proposal := range safeProposers {
		safeProposals = append(safeProposals, proposal)
	}
	return safeProposals, doubleProposals, nil
}

func (s *Service) recordDoubleProposals(doubleProposals []*slashertypes.DoubleBlockProposal) {
	for _, doubleProposal := range doubleProposals {
		doubleProposalsTotal.Inc()
		logDoubleProposal(doubleProposal.PrevBeaconBlockWrapper, doubleProposal.BeaconBlockWrapper)
		s.serviceCfg.ProposerSlashingsFeed.Send(&ethpb.ProposerSlashing{
			Header_1: doubleProposal.PrevBeaconBlockWrapper.SignedBeaconBlockHeader,
			Header_2: doubleProposal.BeaconBlockWrapper.SignedBeaconBlockHeader,
		})
	}
}

func proposalKey(proposal *slashertypes.SignedBlockHeaderWrapper) string {
	header := proposal.SignedBeaconBlockHeader.Header
	return uintToString(uint64(header.Slot)) + ":" + uintToString(uint64(header.ProposerIndex))
}
