// Package state implements the whole state transition
// function which consists of per slot, per-epoch transitions.
// It also bootstraps the genesis beacon state for slot 0.
package state

import (
	"bytes"
	"context"
	"fmt"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	bal "github.com/prysmaticlabs/prysm/beacon-chain/core/balances"
	b "github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	e "github.com/prysmaticlabs/prysm/beacon-chain/core/epoch"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	v "github.com/prysmaticlabs/prysm/beacon-chain/core/validators"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/sliceutil"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
)

var log = logrus.WithField("prefix", "core/state")

var (
	correctAttestedValidatorGauge = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "correct_attested_validator_rate",
		Help: "The % of validators correctly attested for source and target",
	}, []string{
		"epoch",
	})
)

// TransitionConfig defines important configuration options
// for executing a state transition, which can have logging and signature
// verification on or off depending on when and where it is used.
type TransitionConfig struct {
	VerifySignatures bool
	Logging          bool
}

// DefaultConfig option for executing state transitions.
func DefaultConfig() *TransitionConfig {
	return &TransitionConfig{
		VerifySignatures: false,
		Logging:          false,
	}
}

// ExecuteStateTransition defines the procedure for a state transition function.
// Spec pseudocode definition:
//  We now define the state transition function. At a high level the state transition is made up of three parts:
//  - The per-slot transitions, which happens at the start of every slot.
//  - The per-block transitions, which happens at every block.
//  - The per-epoch transitions, which happens at the end of the last slot of every epoch (i.e. (state.slot + 1) % SLOTS_PER_EPOCH == 0).
//  The per-slot transitions focus on the slot counter and block roots records updates.
//  The per-block transitions focus on verifying aggregate signatures and saving temporary records relating to the per-block activity in the state.
//  The per-epoch transitions focus on the validator registry, including adjusting balances and activating and exiting validators,
//  as well as processing crosslinks and managing block justification/finalization.
func ExecuteStateTransition(
	ctx context.Context,
	state *pb.BeaconState,
	block *pb.BeaconBlock,
	headRoot [32]byte,
	config *TransitionConfig,
) (*pb.BeaconState, error) {
	ctx, span := trace.StartSpan(ctx, "beacon-chain.ChainService.StateTransition")
	defer span.End()
	var err error

	// Execute per slot transition.
	state = ProcessSlot(ctx, state, headRoot)

	// Execute per block transition.
	if block != nil {
		state, err = ProcessBlock(ctx, state, block, config)
		if err != nil {
			return nil, fmt.Errorf("could not process block: %v", err)
		}
	}

	// Execute per epoch transition.
	if e.CanProcessEpoch(state) {
		state, err = ProcessEpoch(ctx, state, block, config)
	}
	if err != nil {
		return nil, fmt.Errorf("could not process epoch: %v", err)
	}

	return state, nil
}

// ProcessSlot happens every slot and focuses on the slot counter and block roots record updates.
// It happens regardless if there's an incoming block or not.
//
// Spec pseudocode definition:
//	Set state.slot += 1
//	Let previous_block_root be the hash_tree_root of the previous beacon block processed in the chain
//	Set state.latest_block_roots[(state.slot - 1) % LATEST_BLOCK_ROOTS_LENGTH] = previous_block_root
//	If state.slot % LATEST_BLOCK_ROOTS_LENGTH == 0
//		append merkle_root(state.latest_block_roots) to state.batched_block_roots
func ProcessSlot(ctx context.Context, state *pb.BeaconState, headRoot [32]byte) *pb.BeaconState {
	ctx, span := trace.StartSpan(ctx, "beacon-chain.ChainService.state.ProcessSlot")
	defer span.End()
	state.Slot++
	state = b.ProcessBlockRoots(state, headRoot)
	return state
}

// ProcessBlock creates a new, modified beacon state by applying block operation
// transformations as defined in the Ethereum Serenity specification, including processing proposer slashings,
// processing block attestations, and more.
func ProcessBlock(
	ctx context.Context,
	state *pb.BeaconState,
	block *pb.BeaconBlock,
	config *TransitionConfig,
) (*pb.BeaconState, error) {

	ctx, span := trace.StartSpan(ctx, "beacon-chain.ChainService.state.ProcessBlock")
	defer span.End()

	r, err := hashutil.HashBeaconBlock(block)
	if err != nil {
		return nil, fmt.Errorf("could not hash block: %v", err)
	}

	// Below are the processing steps to verify every block.
	// Verify block slot.
	if block.Slot != state.Slot {
		return nil, fmt.Errorf(
			"block.slot != state.slot, block.slot = %d, state.slot = %d",
			block.Slot-params.BeaconConfig().GenesisSlot,
			state.Slot-params.BeaconConfig().GenesisSlot,
		)
	}

	// Verify block signature.
	if config.VerifySignatures {
		// TODO(#781): Verify Proposer Signature.
		if err := b.VerifyProposerSignature(block); err != nil {
			return nil, fmt.Errorf("could not verify proposer signature: %v", err)
		}
	}

	// Save latest block.
	state.LatestBlock = block

	// Verify block RANDAO.
	state, err = b.ProcessBlockRandao(state, block, config.VerifySignatures, config.Logging)
	if err != nil {
		return nil, fmt.Errorf("could not verify and process block randao: %v", err)
	}

	// Process ETH1 data.
	state = b.ProcessEth1DataInBlock(state, block)
	state, err = b.ProcessAttesterSlashings(state, block, config.VerifySignatures)
	if err != nil {
		return nil, fmt.Errorf("could not verify block attester slashings: %v", err)
	}

	state, err = b.ProcessProposerSlashings(state, block, config.VerifySignatures)
	if err != nil {
		return nil, fmt.Errorf("could not verify block proposer slashings: %v", err)
	}

	state, err = b.ProcessBlockAttestations(state, block, config.VerifySignatures)
	if err != nil {
		return nil, fmt.Errorf("could not process block attestations: %v", err)
	}

	state, err = b.ProcessValidatorDeposits(state, block)
	if err != nil {
		return nil, fmt.Errorf("could not process block validator deposits: %v", err)
	}
	state, err = b.ProcessValidatorExits(state, block, config.VerifySignatures)
	if err != nil {
		return nil, fmt.Errorf("could not process validator exits: %v", err)
	}

	if config.Logging {
		log.WithField("blockRoot", fmt.Sprintf("%#x", bytesutil.Trunc(r[:]))).Debugf("Verified block slot == state slot")
		log.WithField("blockRoot", fmt.Sprintf("%#x", bytesutil.Trunc(r[:]))).Debugf("Verified and processed block RANDAO")
		log.WithField("blockRoot", fmt.Sprintf("%#x", bytesutil.Trunc(r[:]))).Debugf("Processed ETH1 data")
		log.WithField(
			"attestationsInBlock", len(block.Body.Attestations),
		).Info("Block attestations")
		log.WithField(
			"depositsInBlock", len(block.Body.Deposits),
		).Info("Block deposits")
	}
	return state, nil
}

// ProcessEpoch describes the per epoch operations that are performed on the
// beacon state.
//
// Spec pseudocode definition:
// 	 process_candidate_receipt_roots(state)
// 	 update_justification(state)
// 	 update_finalization(state)
// 	 update_crosslinks(state)
// 	 process_attester_reward_penalties(state)
// 	 process_crosslink_reward_penalties(state)
// 	 update_validator_registry(state)
// 	 final_book_keeping(state)
func ProcessEpoch(ctx context.Context, state *pb.BeaconState, block *pb.BeaconBlock, config *TransitionConfig) (*pb.BeaconState, error) {
	ctx, span := trace.StartSpan(ctx, "beacon-chain.ChainService.state.ProcessEpoch")
	defer span.End()

	currentEpoch := helpers.CurrentEpoch(state)
	prevEpoch := helpers.PrevEpoch(state)

	// Calculate total balances of active validators of the current epoch.
	activeValidatorIndices := helpers.ActiveValidatorIndices(state.ValidatorRegistry, currentEpoch)
	totalBalance := e.TotalBalance(state, activeValidatorIndices)

	// We require the current epoch attestations, current epoch boundary attestations,
	// and current boundary attesting balances for processing.
	currentEpochAttestations := []*pb.PendingAttestation{}
	currentEpochBoundaryAttestations := []*pb.PendingAttestation{}
	currentBoundaryAttesterIndices := []uint64{}

	// We also the previous epoch attestations, previous epoch boundary attestations,
	// and previous boundary attesting balances for processing.
	prevEpochAttestations := []*pb.PendingAttestation{}
	prevEpochBoundaryAttestations := []*pb.PendingAttestation{}
	prevEpochAttesterIndices := []uint64{}
	prevEpochBoundaryAttesterIndices := []uint64{}
	prevEpochHeadAttestations := []*pb.PendingAttestation{}
	prevEpochHeadAttesterIndices := []uint64{}

	inclusionSlotByAttester := make(map[uint64]uint64)
	inclusionDistanceByAttester := make(map[uint64]uint64)

	for _, attestation := range state.LatestAttestations {

		// We determine the attestation participants.
		attesterIndices, err := helpers.AttestationParticipants(
			state,
			attestation.Data,
			attestation.AggregationBitfield)
		if err != nil {
			return nil, err
		}

		for _, participant := range attesterIndices {
			inclusionDistanceByAttester[participant] = state.Slot - attestation.Data.Slot
			inclusionSlotByAttester[participant] = attestation.InclusionSlot
		}

		// We extract the attestations from the current epoch.
		if currentEpoch == helpers.SlotToEpoch(attestation.Data.Slot) {
			currentEpochAttestations = append(currentEpochAttestations, attestation)

			// We then extract the boundary attestations.
			boundaryBlockRoot, err := b.BlockRoot(state, helpers.StartSlot(helpers.CurrentEpoch(state)))
			if err != nil {
				return nil, err
			}

			attestationData := attestation.Data
			sameRoot := bytes.Equal(attestationData.EpochBoundaryRootHash32, boundaryBlockRoot)
			if sameRoot {
				currentEpochBoundaryAttestations = append(currentEpochBoundaryAttestations, attestation)
				currentBoundaryAttesterIndices = sliceutil.UnionUint64(currentBoundaryAttesterIndices, attesterIndices)
			}
		}

		// We extract the attestations from the previous epoch.
		if prevEpoch == helpers.SlotToEpoch(attestation.Data.Slot) {
			prevEpochAttestations = append(prevEpochAttestations, attestation)
			prevEpochAttesterIndices = sliceutil.UnionUint64(prevEpochAttesterIndices, attesterIndices)

			// We extract the previous epoch boundary attestations.
			prevBoundaryBlockRoot, err := b.BlockRoot(state,
				helpers.StartSlot(helpers.PrevEpoch(state)))
			if err != nil {
				return nil, err
			}
			if bytes.Equal(attestation.Data.EpochBoundaryRootHash32, prevBoundaryBlockRoot) {
				prevEpochBoundaryAttestations = append(prevEpochBoundaryAttestations, attestation)
				prevEpochBoundaryAttesterIndices = sliceutil.UnionUint64(prevEpochBoundaryAttesterIndices, attesterIndices)
			}

			// We extract the previous epoch head attestations.
			canonicalBlockRoot, err := b.BlockRoot(state, attestation.Data.Slot)
			if err != nil {
				return nil, err
			}

			attestationData := attestation.Data
			if bytes.Equal(attestationData.BeaconBlockRootHash32, canonicalBlockRoot) {
				prevEpochHeadAttestations = append(prevEpochHeadAttestations, attestation)
				prevEpochHeadAttesterIndices = sliceutil.UnionUint64(prevEpochHeadAttesterIndices, attesterIndices)
			}
		}
	}

	// Calculate the attesting balances for previous and current epoch.
	currentBoundaryAttestingBalances := e.TotalBalance(state, currentBoundaryAttesterIndices)
	previousActiveValidatorIndices := helpers.ActiveValidatorIndices(state.ValidatorRegistry, prevEpoch)
	prevTotalBalance := e.TotalBalance(state, previousActiveValidatorIndices)
	prevEpochAttestingBalance := e.TotalBalance(state, prevEpochAttesterIndices)
	prevEpochBoundaryAttestingBalances := e.TotalBalance(state, prevEpochBoundaryAttesterIndices)
	prevEpochHeadAttestingBalances := e.TotalBalance(state, prevEpochHeadAttesterIndices)

	// Process eth1 data.
	if e.CanProcessEth1Data(state) {
		state = e.ProcessEth1Data(state)
	}

	// Update justification and finality.
	state, err := e.ProcessJustificationAndFinalization(
		state,
		currentBoundaryAttestingBalances,
		prevEpochAttestingBalance,
		prevTotalBalance,
		totalBalance,
	)
	if err != nil {
		return nil, fmt.Errorf("could not process justification and finalization of state: %v", err)
	}

	// Process crosslinks records.
	// TODO(#2072): Include an optimized process crosslinks version.
	if featureconfig.FeatureConfig().EnableCrosslinks {
		state, err = e.ProcessCrosslinks(
			state,
			currentEpochAttestations,
			prevEpochAttestations)
		if err != nil {
			return nil, fmt.Errorf("could not process crosslink records: %v", err)
		}
	}

	// Process attester rewards and penalties.
	epochsSinceFinality := e.SinceFinality(state)
	switch {
	case epochsSinceFinality <= 4:
		// Apply rewards/penalties to validators for attesting
		// expected FFG source.
		state = bal.ExpectedFFGSource(
			state,
			prevEpochAttesterIndices,
			prevEpochAttestingBalance,
			totalBalance)
		if config.Logging {
			log.WithField("balances", state.ValidatorBalances).Debug("Balance after FFG src calculation")
		}
		// Apply rewards/penalties to validators for attesting
		// expected FFG target.
		state = bal.ExpectedFFGTarget(
			state,
			prevEpochBoundaryAttesterIndices,
			prevEpochBoundaryAttestingBalances,
			totalBalance)
		if config.Logging {
			log.WithField("balances", state.ValidatorBalances).Debug("Balance after FFG target calculation")
		}
		// Apply rewards/penalties to validators for attesting
		// expected beacon chain head.
		state = bal.ExpectedBeaconChainHead(
			state,
			prevEpochHeadAttesterIndices,
			prevEpochHeadAttestingBalances,
			totalBalance)
		if config.Logging {
			log.WithField("balances", state.ValidatorBalances).Debug("Balance after chain head calculation")
		}
		// Apply rewards for to validators for including attestations
		// based on inclusion distance.
		state, err = bal.InclusionDistance(
			state,
			prevEpochAttesterIndices,
			totalBalance,
			inclusionDistanceByAttester)
		if err != nil {
			return nil, fmt.Errorf("could not calculate inclusion dist rewards: %v", err)
		}
		if config.Logging {
			log.WithField("balances", state.ValidatorBalances).Debug("Balance after inclusion distance calculation")
		}

	case epochsSinceFinality > 4:
		if config.Logging {
			log.WithField("epochSinceFinality", epochsSinceFinality).Info("Applying quadratic leak penalties")
		}
		// Apply penalties for long inactive FFG source participants.
		state = bal.InactivityFFGSource(
			state,
			prevEpochAttesterIndices,
			totalBalance,
			epochsSinceFinality)
		// Apply penalties for long inactive FFG target participants.
		state = bal.InactivityFFGTarget(
			state,
			prevEpochBoundaryAttesterIndices,
			totalBalance,
			epochsSinceFinality)
		// Apply penalties for long inactive validators who didn't
		// attest to head canonical chain.
		state = bal.InactivityChainHead(
			state,
			prevEpochHeadAttesterIndices,
			totalBalance)
		// Apply penalties for long inactive validators who also
		// exited with penalties.
		state = bal.InactivityExitedPenalties(
			state,
			totalBalance,
			epochsSinceFinality)
		// Apply penalties for long inactive validators that
		// don't include attestations.
		state, err = bal.InactivityInclusionDistance(
			state,
			prevEpochAttesterIndices,
			totalBalance,
			inclusionDistanceByAttester)
		if err != nil {
			return nil, fmt.Errorf("could not calculate inclusion penalties: %v", err)
		}
	}

	// Process Attestation Inclusion Rewards.
	state, err = bal.AttestationInclusion(
		state,
		totalBalance,
		prevEpochAttesterIndices,
		inclusionSlotByAttester)
	if err != nil {
		return nil, fmt.Errorf("could not process attestation inclusion rewards: %v", err)
	}

	// Process crosslink rewards and penalties.
	// TODO(#2072): Optimize crosslinks.
	if featureconfig.FeatureConfig().EnableCrosslinks {
		state, err = bal.Crosslinks(
			state,
			currentEpochAttestations,
			prevEpochAttestations)
		if err != nil {
			return nil, fmt.Errorf("could not process crosslink rewards and penalties: %v", err)
		}
	}

	// Process ejections.
	state, err = e.ProcessEjections(state, config.Logging)
	if err != nil {
		return nil, fmt.Errorf("could not process ejections: %v", err)
	}

	// Process validator registry.
	state = e.ProcessPrevSlotShardSeed(state)
	state = v.ProcessPenaltiesAndExits(state)
	if e.CanProcessValidatorRegistry(state) {
		if block != nil {
			state, err = v.UpdateRegistry(state)
			if err != nil {
				return nil, fmt.Errorf("could not update validator registry: %v", err)
			}
		}
		state, err = e.ProcessCurrSlotShardSeed(state)
		if err != nil {
			return nil, fmt.Errorf("could not update current shard shuffling seeds: %v", err)
		}
	} else {
		state, err = e.ProcessPartialValidatorRegistry(state)
		if err != nil {
			return nil, fmt.Errorf("could not process partial validator registry: %v", err)
		}
	}

	// Final housekeeping updates.
	// Update index roots from current epoch to next epoch.
	state, err = e.UpdateLatestActiveIndexRoots(state)
	if err != nil {
		return nil, fmt.Errorf("could not update latest index roots: %v", err)
	}

	// TODO(1763): Implement process_slashings from ETH2.0 beacon chain spec.

	// TODO(1764): Implement process_exit_queue from ETH2.0 beacon chain spec.

	// Update accumulated slashed balances from current epoch to next epoch.
	state = e.UpdateLatestSlashedBalances(state)

	// Update current epoch's randao seed to next epoch.
	state, err = e.UpdateLatestRandaoMixes(state)
	if err != nil {
		return nil, fmt.Errorf("could not update latest randao mixes: %v", err)
	}

	// Clean up processed attestations.
	state = e.CleanupAttestations(state)

	// Log the useful metrics via prometheus.
	correctAttestedValidatorGauge.WithLabelValues(
		strconv.Itoa(int(currentEpoch)),
	).Set(float64(len(currentBoundaryAttesterIndices) / len(activeValidatorIndices)))

	if config.Logging {
		log.WithField("currentEpochAttestations", len(currentEpochAttestations)).Info("Number of current epoch attestations")
		log.WithField("attesterIndices", currentBoundaryAttesterIndices).Debug("Current epoch boundary attester indices")
		log.WithField("prevEpochAttestations", len(prevEpochAttestations)).Info("Number of previous epoch attestations")
		log.WithField("attesterIndices", prevEpochAttesterIndices).Debug("Previous epoch attester indices")
		log.WithField("prevEpochBoundaryAttestations", len(prevEpochBoundaryAttestations)).Info("Number of previous epoch boundary attestations")
		log.WithField("attesterIndices", prevEpochBoundaryAttesterIndices).Debug("Previous epoch boundary attester indices")
		log.WithField(
			"previousJustifiedEpoch", state.PreviousJustifiedEpoch-params.BeaconConfig().GenesisEpoch,
		).Info("Previous justified epoch")
		log.WithField(
			"justifiedEpoch", state.JustifiedEpoch-params.BeaconConfig().GenesisEpoch,
		).Info("Justified epoch")
		log.WithField(
			"finalizedEpoch", state.FinalizedEpoch-params.BeaconConfig().GenesisEpoch,
		).Info("Finalized epoch")
		log.WithField(
			"validatorRegistryUpdateEpoch", state.ValidatorRegistryUpdateEpoch-params.BeaconConfig().GenesisEpoch,
		).Info("Validator Registry Update Epoch")
		log.WithField(
			"numValidators", len(state.ValidatorRegistry),
		).Info("Validator registry length")

		activeValidatorIndices := helpers.ActiveValidatorIndices(state.ValidatorRegistry, helpers.CurrentEpoch(state))
		log.WithField(
			"activeValidators", len(activeValidatorIndices),
		).Info("Active validators")
		totalBalance := float32(0)
		lowestBalance := float32(state.ValidatorBalances[activeValidatorIndices[0]])
		highestBalance := float32(state.ValidatorBalances[activeValidatorIndices[0]])
		for _, idx := range activeValidatorIndices {
			if float32(state.ValidatorBalances[idx]) < lowestBalance {
				lowestBalance = float32(state.ValidatorBalances[idx])
			}
			if float32(state.ValidatorBalances[idx]) > highestBalance {
				highestBalance = float32(state.ValidatorBalances[idx])
			}
			totalBalance += float32(state.ValidatorBalances[idx])
		}
		avgBalance := totalBalance / float32(len(activeValidatorIndices)) / float32(params.BeaconConfig().GweiPerEth)
		lowestBalance = lowestBalance / float32(params.BeaconConfig().GweiPerEth)
		highestBalance = highestBalance / float32(params.BeaconConfig().GweiPerEth)
		log.WithFields(logrus.Fields{
			"averageBalance": avgBalance,
			"lowestBalance":  lowestBalance,
			"highestBalance": highestBalance,
		}).Info("Active validator balances")
	}

	return state, nil
}
