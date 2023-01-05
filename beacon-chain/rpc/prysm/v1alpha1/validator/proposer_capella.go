package validator

import (
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/interfaces"
	"github.com/prysmaticlabs/prysm/v3/time/slots"
)

// Sets the bls to exec data for a block.
func (vs *Server) setBlsToExecData(blk interfaces.BeaconBlock, headState state.BeaconState) error {
	if slots.ToEpoch(blk.Slot()) < params.BeaconConfig().CapellaForkEpoch {
		return nil
	}

	changes, err := vs.BLSChangesPool.BLSToExecChangesForInclusion(headState)
	if err != nil {
		log.WithError(err).Error("Could not get bls to execution changes")
	} else {
		if err := blk.Body().SetBLSToExecutionChanges(changes); err != nil {
			log.WithError(err).Error("Could not set bls to execution changes")
		}
	}
	return nil
}
