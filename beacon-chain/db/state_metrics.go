package db

import (
	"encoding/hex"
	"strconv"

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
	validatorActivatedGauge = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "state_validator_activated_epoch",
		Help: "Activated epoch of validators, updated on epoch transition",
	}, []string{
		"validatorIndex",
	})
	validatorExitedGauge = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "state_validator_exited_epoch",
		Help: "Exited epoch of validators, updated on epoch transition",
	}, []string{
		"validatorIndex",
	})
	validatorSlashedGauge = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "state_validator_slashed_epoch",
		Help: "Slashed epoch of validators, updated on epoch transition",
	}, []string{
		"validatorIndex",
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
	activeValidatorsGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "state_active_validators",
		Help: "Total number of active validators",
	})
)

func reportStateMetrics(state *pb.BeaconState) {
	s := params.BeaconConfig().GenesisSlot
	e := params.BeaconConfig().GenesisEpoch
	currentEpoch := state.Slot / params.BeaconConfig().SlotsPerEpoch
	// Validator balances
	for i, bal := range state.ValidatorBalances {
		validatorBalancesGauge.WithLabelValues(
			"0x" + hex.EncodeToString(state.ValidatorRegistry[i].Pubkey), // Validator
		).Set(float64(bal))
	}

	var active float64
	for i, v := range state.ValidatorRegistry {
		// Track individual Validator's activation epochs
		validatorActivatedGauge.WithLabelValues(
			strconv.Itoa(i), //Validator index
		).Set(float64(v.ActivationEpoch - e))
		// Track individual Validator's exited epochs
		validatorExitedGauge.WithLabelValues(
			strconv.Itoa(i), //Validator index
		).Set(float64(v.ExitEpoch - e))
		// Track individual Validator's slashed epochs
		validatorSlashedGauge.WithLabelValues(
			strconv.Itoa(i), //Validator index
		).Set(float64(v.SlashedEpoch - e))
		// Total number of active validators
		if v.ActivationEpoch <= currentEpoch && currentEpoch < v.ExitEpoch {
			active++
		}
	}
	activeValidatorsGauge.Set(active)

	// Slot number
	lastSlotGauge.Set(float64(state.Slot - s))
	// Last justified slot
	lastJustifiedEpochGauge.Set(float64(state.JustifiedEpoch - e))
	// Last previous justified slot
	lastPrevJustifiedEpochGauge.Set(float64(state.PreviousJustifiedEpoch - e))
	// Last finalized slot
	lastFinalizedEpochGauge.Set(float64(state.FinalizedEpoch - e))
}
