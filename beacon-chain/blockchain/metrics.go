package blockchain

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/epoch/precompute"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
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
	competingAtts = promauto.NewCounter(prometheus.CounterOpts{
		Name: "competing_attestations",
		Help: "The # of attestations received and processed from a competing chain",
	})
	competingBlks = promauto.NewCounter(prometheus.CounterOpts{
		Name: "competing_blocks",
		Help: "The # of blocks received and processed from a competing chain",
	})
	processedBlkNoPubsub = promauto.NewCounter(prometheus.CounterOpts{
		Name: "processed_no_pubsub_block_counter",
		Help: "The # of processed block without pubsub, this usually means the blocks from sync",
	})
	processedBlkNoPubsubForkchoice = promauto.NewCounter(prometheus.CounterOpts{
		Name: "processed_no_pubsub_forkchoice_block_counter",
		Help: "The # of processed block without pubsub and forkchoice, this means indicate blocks from initial sync",
	})
	processedBlk = promauto.NewCounter(prometheus.CounterOpts{
		Name: "processed_block_counter",
		Help: "The # of total processed in block chain service, with fork choice and pubsub",
	})
	processedAttNoPubsub = promauto.NewCounter(prometheus.CounterOpts{
		Name: "processed_no_pubsub_attestation_counter",
		Help: "The # of processed attestation without pubsub, this usually means the attestations from sync",
	})
	processedAtt = promauto.NewCounter(prometheus.CounterOpts{
		Name: "processed_attestation_counter",
		Help: "The # of processed attestation with pubsub and fork choice, this ususally means attestations from rpc",
	})
	headFinalizedEpoch = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "chain_service_head_finalized_epoch",
		Help: "Last finalized epoch of the head state",
	})
	headFinalizedRoot = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "chain_service_head_finalized_root",
		Help: "Last finalized root of the head state",
	})
	beaconFinalizedEpoch = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "chain_service_beacon_finalized_epoch",
		Help: "Last finalized epoch of the processed state",
	})
	beaconFinalizedRoot = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "chain_service_beacon_finalized_root",
		Help: "Last finalized root of the processed state",
	})
	cacheFinalizedEpoch = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "chain_service_cache_finalized_epoch",
		Help: "Last cached finalized epoch",
	})
	cacheFinalizedRoot = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "chain_service_cache_finalized_root",
		Help: "Last cached finalized root",
	})
	beaconCurrentJustifiedEpoch = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "chain_service_beacon_current_justified_epoch",
		Help: "Current justified epoch of the processed state",
	})
	beaconCurrentJustifiedRoot = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "chain_service_beacon_current_justified_root",
		Help: "Current justified root of the processed state",
	})
	beaconPrevJustifiedEpoch = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "chain_service_beacon_previous_justified_epoch",
		Help: "Previous justified epoch of the processed state",
	})
	beaconPrevJustifiedRoot = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "chain_service_beacon_previous_justified_root",
		Help: "Previous justified root of the processed state",
	})
	sigFailsToVerify = promauto.NewCounter(prometheus.CounterOpts{
		Name: "chain_service_att_signature_failed_to_verify_with_cache",
		Help: "Number of attestation signatures that failed to verify with cache on, but succeeded without cache",
	})
	validatorsCount = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "chain_service_validator_count",
		Help: "The total number of validators",
	}, []string{"state"})
	validatorsBalance = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "chain_service_validators_total_balance",
		Help: "The total balance of validators, in GWei",
	}, []string{"state"})
	validatorsEffectiveBalance = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "chain_service_validators_total_effective_balance",
		Help: "The total effective balance of validators, in GWei",
	}, []string{"state"})
	currentEth1DataDepositCount = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "chain_service_current_eth1_data_deposit_count",
		Help: "The current eth1 deposit count in the last processed state eth1data field.",
	})
	totalEligibleBalances = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "chain_service_total_eligible_balances",
		Help: "The total amount of ether, in gwei, that has been used in voting attestation target of previous epoch",
	})
	totalVotedTargetBalances = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "chain_service_total_voted_target_balances",
		Help: "The total amount of ether, in gwei, that is eligible for voting of previous epoch",
	})
)

func (s *Service) reportSlotMetrics(currentSlot uint64) {
	beaconSlot.Set(float64(currentSlot))
	beaconHeadSlot.Set(float64(s.HeadSlot()))
	if s.headState != nil {
		headFinalizedEpoch.Set(float64(s.headState.FinalizedCheckpoint.Epoch))
		headFinalizedRoot.Set(float64(bytesutil.ToLowInt64(s.headState.FinalizedCheckpoint.Root)))
	}
}

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
		currentEth1DataDepositCount.Set(float64(state.Eth1Data.DepositCount))
	}

	if precompute.Balances != nil {
		totalEligibleBalances.Set(float64(precompute.Balances.PrevEpoch))
		totalVotedTargetBalances.Set(float64(precompute.Balances.PrevEpochTargetAttesters))
	}
}
