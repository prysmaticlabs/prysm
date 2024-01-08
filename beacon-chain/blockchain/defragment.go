package blockchain

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prysmaticlabs/prysm/v4/config/params"
	"github.com/prysmaticlabs/prysm/v4/time"
	"github.com/prysmaticlabs/prysm/v4/time/slots"
)

var stateDefragmentationTime = promauto.NewSummary(prometheus.SummaryOpts{
	Name: "head_state_defragmentation_milliseconds",
	Help: "Milliseconds it takes to defragment the head state",
})

// TODO
func (s *Service) runHeadStateDefragmentation() {
	if err := s.waitForSync(); err != nil {
		log.WithError(err).Error("Failed to wait for initial sync to complete")
		return
	}

	ticker := slots.NewSlotTickerWithOffset(s.genesisTime, slots.DivideSlotBy(2), params.BeaconConfig().SecondsPerSlot)
	for {
		select {
		case <-ticker.C():
			if !slots.IsEpochStart(s.headSlot()) {
				continue
			}
			s.headLock.Lock()
			log.Info("Head state defragmentation initialized") // TODO: change to debug
			startTime := time.Now()
			s.head.state.Defragment()
			elapsedTime := time.Since(startTime)
			log.Infof("Head state defragmentation completed in %s", elapsedTime.String()) // TODO: change to debug
			stateDefragmentationTime.Observe(float64(elapsedTime.Milliseconds()))
			s.headLock.Unlock()
		case <-s.ctx.Done():
			log.Debug("Context closed, exiting routine")
			return
		}
	}
}
