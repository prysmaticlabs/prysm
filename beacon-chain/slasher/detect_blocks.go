package slasher

import (
	"context"

	"github.com/pkg/errors"
	slashertypes "github.com/prysmaticlabs/prysm/v4/beacon-chain/slasher/types"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	"go.opencensus.io/trace"
)

// detectProposerSlashings takes in signed block header wrappers and returns a list of proposer slashings detected.
func (s *Service) detectProposerSlashings(
	ctx context.Context,
	incomingProposals []*slashertypes.SignedBlockHeaderWrapper,
) ([]*ethpb.ProposerSlashing, error) {
	ctx, span := trace.StartSpan(ctx, "slasher.detectProposerSlashings")
	defer span.End()

	// internalSlashings will contain any slashable double proposals in the input list
	// of proposals with respect to each other.
	internalSlashings := []*ethpb.ProposerSlashing{}

	existingProposals := make(map[string]*slashertypes.SignedBlockHeaderWrapper)

	// We check if there are any slashable double proposals in the input list
	// of proposals with respect to each other.
	for _, incomingProposal := range incomingProposals {
		key := proposalKey(incomingProposal)
		existingProposal, ok := existingProposals[key]

		// If we have not seen this proposal before, we add it to our map of existing proposals
		// and we continue to the next proposal.
		if !ok {
			existingProposals[key] = incomingProposal
			continue
		}

		// If we have seen this proposal before, we check if it is a double proposal.
		if isDoubleProposal(incomingProposal.SigningRoot, existingProposal.SigningRoot) {
			doubleProposalsTotal.Inc()

			slashing := &ethpb.ProposerSlashing{
				Header_1: existingProposal.SignedBeaconBlockHeader,
				Header_2: incomingProposal.SignedBeaconBlockHeader,
			}

			internalSlashings = append(internalSlashings, slashing)
		}
	}

	// We check if there are any slashable double proposals in the input list
	// of proposals with respect to the slasher database.
	databaseSlashings, err := s.serviceCfg.Database.CheckDoubleBlockProposals(ctx, incomingProposals)
	if err != nil {
		return nil, errors.Wrap(err, "could not check for double proposals on disk")
	}

	// We save the safe proposals (with respect to the database) to our database.
	// If some proposals in incomingProposals are slashable with respect to each other,
	// we (arbitrarily) save the last one to the database.
	if err := s.saveSafeProposals(ctx, incomingProposals, databaseSlashings); err != nil {
		return nil, errors.Wrap(err, "could not save safe proposals")
	}

	// totalSlashings contain all slashings we have detected.
	totalSlashings := append(internalSlashings, databaseSlashings...)
	return totalSlashings, nil
}

// Check for double proposals in our database given a list of incoming block proposals.
// For the proposals that were not slashable with respect to the database,
// we save them to the database.
// For the proposals that are slashable with respect to the content of proposals themselves,
// we (arbitrarily) save the last one to the dtatabase.
func (s *Service) saveSafeProposals(
	ctx context.Context,
	proposals []*slashertypes.SignedBlockHeaderWrapper,
	proposerSlashings []*ethpb.ProposerSlashing,
) error {
	ctx, span := trace.StartSpan(ctx, "slasher.saveSafeProposals")
	defer span.End()

	filteredProposals := filterSafeProposals(proposals, proposerSlashings)
	return s.serviceCfg.Database.SaveBlockProposals(ctx, filteredProposals)
}

// filterSafeProposals, given a list of proposals and a list of proposer slashings,
// filters out proposals for which the proposer index is found in proposer slashings.
func filterSafeProposals(
	proposals []*slashertypes.SignedBlockHeaderWrapper,
	proposerSlashings []*ethpb.ProposerSlashing,
) []*slashertypes.SignedBlockHeaderWrapper {
	// We initialize a map of proposers that are safe from slashing.
	safeProposers := make(map[primitives.ValidatorIndex]*slashertypes.SignedBlockHeaderWrapper, len(proposals))

	for _, proposal := range proposals {
		safeProposers[proposal.SignedBeaconBlockHeader.Header.ProposerIndex] = proposal
	}

	for _, proposerSlashing := range proposerSlashings {
		// If a proposer is found to have committed a slashable offense, we delete
		// them from the safe proposers map.
		delete(safeProposers, proposerSlashing.Header_1.Header.ProposerIndex)
	}

	// We save all the proposals that are determined "safe" and not-slashable to our database.
	safeProposals := make([]*slashertypes.SignedBlockHeaderWrapper, 0, len(safeProposers))
	for _, proposal := range safeProposers {
		safeProposals = append(safeProposals, proposal)
	}

	return safeProposals
}

// proposalKey build a key which is a combination of the slot and the proposer index.
// If a validator proposes several blocks for the same slot, then several (potentially slashable)
// proposals will correspond to the same key.
func proposalKey(proposal *slashertypes.SignedBlockHeaderWrapper) string {
	header := proposal.SignedBeaconBlockHeader.Header

	slotKey := uintToString(uint64(header.Slot))
	proposerIndexKey := uintToString(uint64(header.ProposerIndex))

	return slotKey + ":" + proposerIndexKey
}
