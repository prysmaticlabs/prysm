package forkchoice

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

var (
	beaconFinalizedEpoch = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "beacon_finalized_epoch",
		Help: "Last finalized epoch of the processed state",
	})
	beaconFinalizedRoot = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "beacon_finalized_root",
		Help: "Last finalized root of the processed state",
	})
	beaconCurrentJustifiedEpoch = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "beacon_current_justified_epoch",
		Help: "Current justified epoch of the processed state",
	})
	beaconCurrentJustifiedRoot = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "beacon_current_justified_root",
		Help: "Current justified root of the processed state",
	})
	beaconPrevJustifiedEpoch = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "beacon_previous_justified_epoch",
		Help: "Previous justified epoch of the processed state",
	})
	beaconPrevJustifiedRoot = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "beacon_previous_justified_root",
		Help: "Previous justified root of the processed state",
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
		Name: "beacon_current_validators",
		Help: "Number of status=pending|active|exited|withdrawable validators in current epoch",
	})
)

func reportEpochMetrics(state *pb.BeaconState) {
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

	// Last justified slot
	if state.CurrentJustifiedCheckpoint != nil {
		beaconCurrentJustifiedEpoch.Set(float64(state.CurrentJustifiedCheckpoint.Epoch))
		beaconCurrentJustifiedRoot.Set(float64(bytesutil.ToLowInt64(state.CurrentJustifiedCheckpoint.Root)))
	}
	// Last previous justified slot
	if state.PreviousJustifiedCheckpoint != nil {
		beaconPrevJustifiedEpoch.Set(float64(state.PreviousJustifiedCheckpoint.Epoch))
		beaconPrevJustifiedRoot.Set(float64(bytesutil.ToLowInt64(state.PreviousJustifiedCheckpoint.Root)))
	}
	// Last finalized slot
	if state.FinalizedCheckpoint != nil {
		beaconFinalizedEpoch.Set(float64(state.FinalizedCheckpoint.Epoch))
		beaconFinalizedRoot.Set(float64(bytesutil.ToLowInt64(state.FinalizedCheckpoint.Root)))
	}
}
