package state

import (
	"fmt"

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
)

func reportEpochTransitionMetrics(state *pb.BeaconState) {
	// Validator balances
	for i, bal := range state.ValidatorBalances {
		validatorBalancesGauge.WithLabelValues(
			fmt.Sprintf("%#x", state.ValidatorRegistry[i].Pubkey), // Validator
		).Set(float64(bal))
	}
}
