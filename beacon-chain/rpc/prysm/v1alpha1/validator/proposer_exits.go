package validator

import (
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
)

func (vs *Server) getExits(head state.BeaconState, slot primitives.Slot) []*ethpb.SignedVoluntaryExit {
	exits := vs.ExitPool.PendingExits(head, slot, false /*noLimit*/)
	validExits := make([]*ethpb.SignedVoluntaryExit, 0, len(exits))

	for _, exit := range exits {
		val, err := head.ValidatorAtIndexReadOnly(exit.Exit.ValidatorIndex)
		if err != nil {
			log.WithError(err).Warn("Could not retrieve validator index")
			continue
		}
		if err := blocks.VerifyExitAndSignature(val, head.Slot(), head.Fork(), exit, head.GenesisValidatorsRoot()); err != nil {
			log.WithError(err).Warn("Could not verify exit for block inclusion")
			continue
		}
		validExits = append(validExits, exit)
	}

	return validExits
}
