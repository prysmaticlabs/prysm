package state

import (
	"encoding/hex"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

var (
	validatorBalancesGauge = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "state_validator_balances",
		Help: "Balances of validators, updated on epoch transition",
	}, []string{
		"validator",
	})
	lastSlotGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "state_last_slot",
		Help: "Last slot number of the processed state",
	})
)

func reportEpochTransitionMetrics(state *pb.BeaconState) {
	// Validator balances
	for i, bal := range state.ValidatorBalances {
		validatorBalancesGauge.WithLabelValues(
			"0x" + hex.EncodeToString(state.ValidatorRegistry[i].Pubkey), // Validator
		).Set(float64(bal))
	}
	// Slot number
	lastSlotGauge.Set(float64(state.Slot))
}
