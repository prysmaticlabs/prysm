package state

import (
	"encoding/hex"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

var (
	validatorBalancesGauge = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "state_validator_balances",
		Help: "Balances of validators, updated on epoch transition",
	}, []string{
		"validator",
	})
	lastSlotGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "state_last_slot",
		Help: "Last slot number of the processed state",
	})
	lastJustifiedEpochGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "state_last_justified_epoch",
		Help: "Last justified epoch of the processed state",
	})
	lastPrevJustifiedEpochGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "state_last_prev_justified_epoch",
		Help: "Last prev justified epoch of the processed state",
	})
	lastFinalizedEpochGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "state_last_finalized_epoch",
		Help: "Last finalized epoch of the processed state",
	})
)

func reportEpochTransitionMetrics(state *pb.BeaconState) {
	// Validator balances
	for i, bal := range state.ValidatorBalances {
		validatorBalancesGauge.WithLabelValues(
			"0x" + hex.EncodeToString(state.ValidatorRegistry[i].Pubkey), // Validator
		).Set(float64(bal))
	}
	s := params.BeaconConfig().GenesisSlot
	e := params.BeaconConfig().GenesisEpoch
	// Slot number
	lastSlotGauge.Set(float64(state.Slot - s))
	// Last justified slot
	lastJustifiedEpochGauge.Set(float64(state.JustifiedEpoch - e))
	// Last previous justified slot
	lastPrevJustifiedEpochGauge.Set(float64(state.PreviousJustifiedEpoch - e))
	// Last finalized slot
	lastFinalizedEpochGauge.Set(float64(state.FinalizedEpoch - e))
}
