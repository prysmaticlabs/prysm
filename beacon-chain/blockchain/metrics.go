package blockchain

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/epoch/precompute"
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

var (
	beaconSlot = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "beacon_slot",
		Help: "Latest slot of the beacon chain state",
	})
	beaconHeadSlot = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "beacon_head_slot",
		Help: "Slot of the head block of the beacon chain",
	})
	clockTimeSlot = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "beacon_clock_time_slot",
		Help: "The current slot based on the genesis time and current clock",
	})

	headFinalizedEpoch = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "head_finalized_epoch",
		Help: "Last finalized epoch of the head state",
	})
	headFinalizedRoot = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "head_finalized_root",
		Help: "Last finalized root of the head state",
	})
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
	validatorsCount = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "validator_count",
		Help: "The total number of validators",
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
	totalEligibleBalances = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "total_eligible_balances",
		Help: "The total amount of ether, in gwei, that is eligible for voting of previous epoch",
	})
	totalVotedTargetBalances = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "total_voted_target_balances",
		Help: "The total amount of ether, in gwei, that has been used in voting attestation target of previous epoch",
	})
	reorgCount = promauto.NewCounter(prometheus.CounterOpts{
		Name: "beacon_reorg_total",
		Help: "Count the number of times beacon chain has a reorg",
	})
	attestationInclusionDelay = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "attestation_inclusion_delay_slots",
			Help:    "The number of slots between att.Slot and block.Slot",
			Buckets: []float64{1, 2, 3, 4, 6, 32, 64},
		},
	)
)

// reportSlotMetrics reports slot related metrics.
func reportSlotMetrics(stateSlot uint64, headSlot uint64, clockSlot uint64, finalizedCheckpoint *ethpb.Checkpoint) {
	clockTimeSlot.Set(float64(clockSlot))
	beaconSlot.Set(float64(stateSlot))
	beaconHeadSlot.Set(float64(headSlot))
	if finalizedCheckpoint != nil {
		headFinalizedEpoch.Set(float64(finalizedCheckpoint.Epoch))
		headFinalizedRoot.Set(float64(bytesutil.ToLowInt64(finalizedCheckpoint.Root)))
	}
}

// reportEpochMetrics reports epoch related metrics.
func reportEpochMetrics(state *stateTrie.BeaconState) {
	currentEpoch := state.Slot() / params.BeaconConfig().SlotsPerEpoch

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

	for i, validator := range state.Validators() {
		bal, err := state.BalanceAtIndex(uint64(i))
		if err != nil {
			log.Errorf("Could not load validator balance: %v", err)
			continue
		}
		if validator.Slashed {
			if currentEpoch < validator.ExitEpoch {
				slashingInstances++
				slashingBalance += bal
				slashingEffectiveBalance += validator.EffectiveBalance
			} else {
				slashedInstances++
			}
			continue
		}
		if validator.ExitEpoch != params.BeaconConfig().FarFutureEpoch {
			if currentEpoch < validator.ExitEpoch {
				exitingInstances++
				exitingBalance += bal
				exitingEffectiveBalance += validator.EffectiveBalance
			} else {
				exitedInstances++
			}
			continue
		}
		if currentEpoch < validator.ActivationEpoch {
			pendingInstances++
			pendingBalance += bal
			continue
		}
		activeInstances++
		activeBalance += bal
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
	beaconCurrentJustifiedEpoch.Set(float64(state.CurrentJustifiedCheckpoint().Epoch))
	beaconCurrentJustifiedRoot.Set(float64(bytesutil.ToLowInt64(state.CurrentJustifiedCheckpoint().Root)))

	// Last previous justified slot
	beaconPrevJustifiedEpoch.Set(float64(state.PreviousJustifiedCheckpoint().Epoch))
	beaconPrevJustifiedRoot.Set(float64(bytesutil.ToLowInt64(state.PreviousJustifiedCheckpoint().Root)))

	// Last finalized slot
	beaconFinalizedEpoch.Set(float64(state.FinalizedCheckpointEpoch()))
	beaconFinalizedRoot.Set(float64(bytesutil.ToLowInt64(state.FinalizedCheckpoint().Root)))

	currentEth1DataDepositCount.Set(float64(state.Eth1Data().DepositCount))

	if precompute.Balances != nil {
		totalEligibleBalances.Set(float64(precompute.Balances.ActivePrevEpoch))
		totalVotedTargetBalances.Set(float64(precompute.Balances.PrevEpochTargetAttested))
	}
}

func reportAttestationInclusion(blk *ethpb.BeaconBlock) {
	for _, att := range blk.Body.Attestations {
		attestationInclusionDelay.Observe(float64(blk.Slot - att.Data.Slot))
	}
}
