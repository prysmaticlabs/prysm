package forkchoice

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

var (
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
	activeValidatorsGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "state_active_validators",
		Help: "Total number of active validators",
	})
	slashedValidatorsGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "state_slashed_validators",
		Help: "Total slashed validators",
	})
	withdrawnValidatorsGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "state_withdrawn_validators",
		Help: "Total withdrawn validators",
	})
	totalValidatorsGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "state_total_validators",
		Help: "All time total validators",
	})
)

func reportStateMetrics(state *pb.BeaconState) {
	currentEpoch := state.Slot / params.BeaconConfig().SlotsPerEpoch

	// Validator counts
	var active float64
	var slashed float64
	var withdrawn float64
	for _, v := range state.Validators {
		if v.ActivationEpoch <= currentEpoch && currentEpoch < v.ExitEpoch {
			active++
		}
		if v.Slashed {
			slashed++
		}
		if currentEpoch >= v.ExitEpoch {
			withdrawn++
		}
	}
	activeValidatorsGauge.Set(active)
	slashedValidatorsGauge.Set(slashed)
	withdrawnValidatorsGauge.Set(withdrawn)
	totalValidatorsGauge.Set(float64(len(state.Validators)))

	// Slot number
	lastSlotGauge.Set(float64(state.Slot))

	// Last justified slot
	if state.CurrentJustifiedCheckpoint != nil {
		lastJustifiedEpochGauge.Set(float64(state.CurrentJustifiedCheckpoint.Epoch))
	}
	// Last previous justified slot
	if state.PreviousJustifiedCheckpoint != nil {
		lastPrevJustifiedEpochGauge.Set(float64(state.PreviousJustifiedCheckpoint.Epoch))
	}
	// Last finalized slot
	if state.FinalizedCheckpoint != nil {
		lastFinalizedEpochGauge.Set(float64(state.FinalizedCheckpoint.Epoch))
	}
}
