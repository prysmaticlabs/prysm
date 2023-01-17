package sync

import (
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v3/time/slots"
)

// This routine broadcasts all known BLS changes at the Capella fork.
func (s *Service) broadcastBLSChanges(currSlot types.Slot) error {
	capellaSlotStart, err := slots.EpochStart(params.BeaconConfig().CapellaForkEpoch)
	if err != nil {
		// only possible error is an overflow, so we exit early from the method
		return nil
	}
	if currSlot == capellaSlotStart {
		changes, err := s.cfg.blsToExecPool.PendingBLSToExecChanges()
		if err != nil {
			return errors.Wrap(err, "could not get BLS to execution changes")
		}
		for _, ch := range changes {
			if err := s.cfg.p2p.Broadcast(s.ctx, ch); err != nil {
				return errors.Wrap(err, "could not broadcast BLS to execution changes.")
			}
		}
	}
	return nil
}
