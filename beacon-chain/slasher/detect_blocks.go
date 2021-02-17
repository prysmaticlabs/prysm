package slasher

import (
	"context"

	types "github.com/prysmaticlabs/eth2-types"
	slashertypes "github.com/prysmaticlabs/prysm/beacon-chain/slasher/types"
)

// Given a list of blocks, check if they are slashable for the validators involved.
func (s *Service) detectSlashableBlocks(
	ctx context.Context,
	proposedBlocks []*slashertypes.CompactBeaconBlock,
) error {
	existingProposals, err := s.serviceCfg.Database.CheckAndUpdateForSlashableProposals(ctx, proposedBlocks)
	if err != nil {
		return err
	}
	for i, existing := range existingProposals {
		if existing == nil {
			continue
		}
		if existing.SigningRoot != proposedBlocks[i].SigningRoot {
			// TODO(#8331): Send over an event feed.
			logSlashingEvent(&slashertypes.Slashing{
				Kind:            slashertypes.DoubleProposal,
				ValidatorIndex:  types.ValidatorIndex(proposedBlocks[i].ProposerIndex),
				SigningRoot:     proposedBlocks[i].SigningRoot,
				PrevSigningRoot: existing.SigningRoot,
				Slot:            proposedBlocks[i].Slot,
			})
		}
	}
	return nil
}
