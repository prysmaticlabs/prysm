package blockchain

import (
	"github.com/prysmaticlabs/prysm/v3/async/event"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/feed"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	"github.com/prysmaticlabs/prysm/v3/time/slots"
)

// This routine broadcasts all known BLS changes at the Capella fork.
func (s *Service) spawnBroadcastBLSChangesRoutine(stateFeed *event.Feed) {
	// Wait for state to be initialized.
	stateChannel := make(chan *feed.Event, 1)
	stateSub := stateFeed.Subscribe(stateChannel)
	go func() {
		select {
		case <-s.ctx.Done():
			stateSub.Unsubscribe()
			return
		case <-stateChannel:
			stateSub.Unsubscribe()
			break
		}

		st := slots.NewSlotTicker(s.genesisTime, params.BeaconConfig().SecondsPerSlot)
		for {
			select {
			case <-s.ctx.Done():
				return
			case <-st.C():
				if slots.ToEpoch(s.CurrentSlot()) >= params.BeaconConfig().CapellaForkEpoch {
					changes, err := s.cfg.BLSToExecPool.PendingBLSToExecChanges()
					if err != nil {
						log.WithError(err).Error("Could not get BLS to execution changes.")
						return
					}
					for _, ch := range changes {
						if err := s.cfg.P2p.Broadcast(s.ctx, ch); err != nil {
							log.WithError(err).Error("Could not broadcast BLS to execution changes.")
							return
						}
					}
					return
				}
			}
		}
	}()
}
