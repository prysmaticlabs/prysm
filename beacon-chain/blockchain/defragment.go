package blockchain

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/v4/time"
)

var stateDefragmentationTime = promauto.NewSummary(prometheus.SummaryOpts{
	Name: "head_state_defragmentation_milliseconds",
	Help: "Milliseconds it takes to defragment the head state",
})

// TODO
func (s *Service) runStateDefragmentation(st state.BeaconState) {
	log.Debug("Head state defragmentation initialized")
	startTime := time.Now()
	st.Defragment()
	elapsedTime := time.Since(startTime)
	log.Debugf("Head state defragmentation completed in %s", elapsedTime.String())
	stateDefragmentationTime.Observe(float64(elapsedTime.Milliseconds()))

}
