package validator

import (
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
)

func (vs *Server) getExits(head state.BeaconState, slot primitives.Slot) []*ethpb.SignedVoluntaryExit {
	exits, err := vs.ExitPool.ExitsForInclusion(head, slot)
	if err != nil {
		log.WithError(err).Error("Could not get exits")
		return []*ethpb.SignedVoluntaryExit{}
	}
	return exits
}
