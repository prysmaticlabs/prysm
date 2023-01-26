package sync

import (
	"github.com/prysmaticlabs/prysm/v3/config/params"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v3/crypto/rand"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/time/slots"
)

// This routine broadcasts known BLS changes at the Capella fork.
func (s *Service) broadcastBLSChanges(currSlot types.Slot) {
	capellaSlotStart, err := slots.EpochStart(params.BeaconConfig().CapellaForkEpoch)
	if err != nil {
		// only possible error is an overflow, so we exit early from the method
		return
	}
	if currSlot != capellaSlotStart {
		return
	}
	changes, err := s.cfg.blsToExecPool.PendingBLSToExecChanges()
	if err != nil {
		log.WithError(err).Error("could not get BLS to execution changes")
	}
	if len(changes) == 0 {
		return
	}
	source := rand.NewGenerator()
	broadcastChanges := make([]*ethpb.SignedBLSToExecutionChange, len(changes))
	for i := 0; i < len(changes); i++ {
		idx := source.Intn(len(changes))
		broadcastChanges[i] = changes[idx]
		changes = append(changes[:idx], changes[idx+1:]...)
	}
	s.cfg.p2p.BroadcastBLSChanges(s.ctx, broadcastChanges)
}
