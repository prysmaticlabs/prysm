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
	cacheFinalizedEpoch = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "cache_finalized_epoch",
		Help: "Last cached finalized epoch",
	})
	cacheFinalizedRoot = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "cache_finalized_root",
		Help: "Last cached finalized root",
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
	sigFailsToVerify = promauto.NewCounter(prometheus.CounterOpts{
		Name: "att_signature_failed_to_verify_with_cache",
		Help: "Number of attestation signatures that failed to verify with cache on, but succeeded without cache",
	})
	validatorsCount = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "validator_count",
		Help: "The total number of validators, in GWei",
	}, []string{"state"})
	validatorsBalance = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "validators_total_balance",
		Help: "The total balance of validators, in GWei",
	}, []string{"state"})
	validatorsEffectiveBalance = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "validators_total_effective_balance",
		Help: "The total effective balance of validators, in GWei",
	}, []string{"state"})
	currentEth1DataDepositCount = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "current_eth1_data_deposit_count",
		Help: "The current eth1 deposit count in the last processed state eth1data field.",
	})
)

func reportEpochMetrics(state *pb.BeaconState) {
	currentEpoch := state.Slot / params.BeaconConfig().SlotsPerEpoch

	// Validator instances
	pendingInstances := 0
	activeInstances := 0
	slashingInstances := 0
	slashedInstances := 0
	exitingInstances := 0
	exitedInstances := 0
	// Validator balances
	pendingBalance := uint64(0)
	activeBalance := uint64(0)
	activeEffectiveBalance := uint64(0)
	exitingBalance := uint64(0)
	exitingEffectiveBalance := uint64(0)
	slashingBalance := uint64(0)
	slashingEffectiveBalance := uint64(0)

	for i, validator := range state.Validators {
		if validator.Slashed {
			if currentEpoch < validator.ExitEpoch {
				slashingInstances++
				slashingBalance += state.Balances[i]
				slashingEffectiveBalance += validator.EffectiveBalance
			} else {
				slashedInstances++
			}
			continue
		}
		if validator.ExitEpoch != params.BeaconConfig().FarFutureEpoch {
			if currentEpoch < validator.ExitEpoch {
				exitingInstances++
				exitingBalance += state.Balances[i]
				exitingEffectiveBalance += validator.EffectiveBalance
			} else {
				exitedInstances++
			}
			continue
		}
		if currentEpoch < validator.ActivationEpoch {
			pendingInstances++
			pendingBalance += state.Balances[i]
			continue
		}
		activeInstances++
		activeBalance += state.Balances[i]
		activeEffectiveBalance += validator.EffectiveBalance
	}
	validatorsCount.WithLabelValues("Pending").Set(float64(pendingInstances))
	validatorsCount.WithLabelValues("Active").Set(float64(activeInstances))
	validatorsCount.WithLabelValues("Exiting").Set(float64(exitingInstances))
	validatorsCount.WithLabelValues("Exited").Set(float64(exitedInstances))
	validatorsCount.WithLabelValues("Slashing").Set(float64(slashingInstances))
	validatorsCount.WithLabelValues("Slashed").Set(float64(slashedInstances))
	validatorsBalance.WithLabelValues("Pending").Set(float64(pendingBalance))
	validatorsBalance.WithLabelValues("Active").Set(float64(activeBalance))
	validatorsBalance.WithLabelValues("Exiting").Set(float64(exitingBalance))
	validatorsBalance.WithLabelValues("Slashing").Set(float64(slashingBalance))
	validatorsEffectiveBalance.WithLabelValues("Active").Set(float64(activeEffectiveBalance))
	validatorsEffectiveBalance.WithLabelValues("Exiting").Set(float64(exitingEffectiveBalance))
	validatorsEffectiveBalance.WithLabelValues("Slashing").Set(float64(slashingEffectiveBalance))

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
	if state.Eth1Data != nil {
		currentEth1DataDepositCount.Set(state.Eth1Data.DepositCount)
	}
}
