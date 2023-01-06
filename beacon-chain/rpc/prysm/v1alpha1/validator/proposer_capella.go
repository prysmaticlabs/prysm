package validator

import (
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/interfaces"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/runtime/version"
)

// Sets the bls to exec data for a block.
func (vs *Server) setBlsToExecData(blk interfaces.BeaconBlock, headState state.BeaconState) error {
	if blk.Version() < version.Capella {
		return nil
	}

	changes, err := vs.BLSChangesPool.BLSToExecChangesForInclusion(headState)
	if err != nil {
		if err := blk.Body().SetBLSToExecutionChanges([]*ethpb.SignedBLSToExecutionChange{}); err != nil {
			log.WithError(err).Error("Could not set bls to execution data in block")
		}
		log.WithError(err).Error("Could not get bls to execution changes")
	} else {
		if err := blk.Body().SetBLSToExecutionChanges(changes); err != nil {
			if err := blk.Body().SetBLSToExecutionChanges([]*ethpb.SignedBLSToExecutionChange{}); err != nil {
				log.WithError(err).Error("Could not set bls to execution data in block")
			}
			log.WithError(err).Error("Could not set bls to execution changes")
		}
	}
	return nil
}
