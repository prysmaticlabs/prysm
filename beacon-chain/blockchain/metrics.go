package blockchain

import (
	"context"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/altair"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/epoch/precompute"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/block"
	"github.com/prysmaticlabs/prysm/runtime/version"
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
	activeValidatorCount = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "beacon_current_active_validators",
		Help: "Current total active validators",
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
	processedDepositsCount = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "beacon_processed_deposits_total",
		Help: "Total number of deposits processed",
	})
	stateTrieReferences = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "field_references",
		Help: "The number of states a particular field is shared with.",
	}, []string{"state"})
	prevEpochActiveBalances = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "beacon_prev_epoch_active_gwei",
		Help: "The total amount of ether, in gwei, that was active for voting of previous epoch",
	})
	prevEpochSourceBalances = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "beacon_prev_epoch_source_gwei",
		Help: "The total amount of ether, in gwei, that has been used in voting attestation source of previous epoch",
	})
	prevEpochTargetBalances = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "beacon_prev_epoch_target_gwei",
		Help: "The total amount of ether, in gwei, that has been used in voting attestation target of previous epoch",
	})
	prevEpochHeadBalances = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "beacon_prev_epoch_head_gwei",
		Help: "The total amount of ether, in gwei, that has been used in voting attestation head of previous epoch",
	})
	reorgCount = promauto.NewCounter(prometheus.CounterOpts{
		Name: "beacon_reorgs_total",
		Help: "Count the number of times beacon chain has a reorg",
	})
	saveOrphanedAttCount = promauto.NewCounter(prometheus.CounterOpts{
		Name: "saved_orphaned_att_total",
		Help: "Count the number of times an orphaned attestation is saved",
	})
	attestationInclusionDelay = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "attestation_inclusion_delay_slots",
			Help:    "The number of slots between att.Slot and block.Slot",
			Buckets: []float64{1, 2, 3, 4, 6, 32, 64},
		},
	)
	syncHeadStateMiss = promauto.NewCounter(prometheus.CounterOpts{
		Name: "sync_head_state_miss",
		Help: "The number of sync head state requests that are not present in the cache.",
	})
	syncHeadStateHit = promauto.NewCounter(prometheus.CounterOpts{
		Name: "sync_head_state_hit",
		Help: "The number of sync head state requests that are present in the cache.",
	})
	stateBalanceCacheHit = promauto.NewCounter(prometheus.CounterOpts{
		Name: "state_balance_cache_hit",
		Help: "Count the number of state balance cache hits.",
	})
	stateBalanceCacheMiss = promauto.NewCounter(prometheus.CounterOpts{
		Name: "state_balance_cache_miss",
		Help: "Count the number of state balance cache hits.",
	})
)

// reportSlotMetrics reports slot related metrics.
func reportSlotMetrics(stateSlot, headSlot, clockSlot types.Slot, finalizedCheckpoint *ethpb.Checkpoint) {
	clockTimeSlot.Set(float64(clockSlot))
	beaconSlot.Set(float64(stateSlot))
	beaconHeadSlot.Set(float64(headSlot))
	if finalizedCheckpoint != nil {
		headFinalizedEpoch.Set(float64(finalizedCheckpoint.Epoch))
		headFinalizedRoot.Set(float64(bytesutil.ToLowInt64(finalizedCheckpoint.Root)))
	}
}

// reportEpochMetrics reports epoch related metrics.
func reportEpochMetrics(ctx context.Context, postState, headState state.BeaconState) error {
	currentEpoch := types.Epoch(postState.Slot() / params.BeaconConfig().SlotsPerEpoch)

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

	for i, validator := range postState.Validators() {
		bal, err := postState.BalanceAtIndex(types.ValidatorIndex(i))
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
	activeInstances += exitingInstances + slashingInstances
	activeBalance += exitingBalance + slashingBalance
	activeEffectiveBalance += exitingEffectiveBalance + slashingEffectiveBalance

	activeValidatorCount.Set(float64(activeInstances))
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
	beaconCurrentJustifiedEpoch.Set(float64(postState.CurrentJustifiedCheckpoint().Epoch))
	beaconCurrentJustifiedRoot.Set(float64(bytesutil.ToLowInt64(postState.CurrentJustifiedCheckpoint().Root)))

	// Last previous justified slot
	beaconPrevJustifiedEpoch.Set(float64(postState.PreviousJustifiedCheckpoint().Epoch))
	beaconPrevJustifiedRoot.Set(float64(bytesutil.ToLowInt64(postState.PreviousJustifiedCheckpoint().Root)))

	// Last finalized slot
	beaconFinalizedEpoch.Set(float64(postState.FinalizedCheckpointEpoch()))
	beaconFinalizedRoot.Set(float64(bytesutil.ToLowInt64(postState.FinalizedCheckpoint().Root)))
	currentEth1DataDepositCount.Set(float64(postState.Eth1Data().DepositCount))
	processedDepositsCount.Set(float64(postState.Eth1DepositIndex() + 1))

	var b *precompute.Balance
	var v []*precompute.Validator
	var err error
	switch headState.Version() {
	case version.Phase0:
		// Validator participation should be viewed on the canonical chain.
		v, b, err = precompute.New(ctx, headState)
		if err != nil {
			return err
		}
		_, b, err = precompute.ProcessAttestations(ctx, headState, v, b)
		if err != nil {
			return err
		}
	case version.Altair, version.Merge:
		v, b, err = altair.InitializePrecomputeValidators(ctx, headState)
		if err != nil {
			return err
		}
		_, b, err = altair.ProcessEpochParticipation(ctx, headState, b, v)
		if err != nil {
			return err
		}
	default:
		return errors.Errorf("invalid state type provided: %T", headState.InnerStateUnsafe())
	}
	prevEpochActiveBalances.Set(float64(b.ActivePrevEpoch))
	prevEpochSourceBalances.Set(float64(b.PrevEpochAttested))
	prevEpochTargetBalances.Set(float64(b.PrevEpochTargetAttested))
	prevEpochHeadBalances.Set(float64(b.PrevEpochHeadAttested))

	refMap := postState.FieldReferencesCount()
	for name, val := range refMap {
		stateTrieReferences.WithLabelValues(name).Set(float64(val))
	}

	return nil
}

func reportAttestationInclusion(blk block.BeaconBlock) {
	for _, att := range blk.Body().Attestations() {
		attestationInclusionDelay.Observe(float64(blk.Slot() - att.Data.Slot))
	}
}
