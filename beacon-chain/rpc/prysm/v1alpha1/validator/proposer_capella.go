package validator

import (
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/interfaces"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/runtime/version"
)

// Sets the bls to exec data for a block.
func (vs *Server) setBlsToExecData(blk interfaces.BeaconBlock, headState state.BeaconState) {
	if blk.Version() < version.Capella {
		return
	}
	if err := blk.Body().SetBLSToExecutionChanges([]*ethpb.SignedBLSToExecutionChange{}); err != nil {
		log.WithError(err).Error("Could not set bls to execution data in block")
		return
	}
	changes, err := vs.BLSChangesPool.BLSToExecChangesForInclusion(headState)
	if err != nil {
		log.WithError(err).Error("Could not get bls to execution changes")
		return
	} else {
		if err := blk.Body().SetBLSToExecutionChanges(changes); err != nil {
			log.WithError(err).Error("Could not set bls to execution changes")
			return
		}
	}
}
