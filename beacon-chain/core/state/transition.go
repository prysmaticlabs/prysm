// Package state implements the whole state transition
// function which consists of per slot, per-epoch transitions.
// It also bootstraps the genesis beacon state for slot 0.
package state

import (
	"context"
	"fmt"

	bal "github.com/prysmaticlabs/prysm/beacon-chain/core/balances"
	b "github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	e "github.com/prysmaticlabs/prysm/beacon-chain/core/epoch"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	v "github.com/prysmaticlabs/prysm/beacon-chain/core/validators"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
)

var log = logrus.WithField("prefix", "core/state")

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
	verifySignatures bool,
) (*pb.BeaconState, error) {
	var err error

	// Execute per slot transition.
	state = ProcessSlot(ctx, state, headRoot)

	// Execute per block transition.
	if block != nil {
		state, err = ProcessBlock(ctx, state, block, verifySignatures)
		if err != nil {
			return nil, fmt.Errorf("could not process block: %v", err)
		}
	}

	// Execute per epoch transition.
	if e.CanProcessEpoch(state) {
		state, err = ProcessEpoch(ctx, state)
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
	state = b.ProcessBlockRoots(ctx, state, headRoot)
	return state
}

// ProcessBlock creates a new, modified beacon state by applying block operation
// transformations as defined in the Ethereum Serenity specification, including processing proposer slashings,
// processing block attestations, and more.
func ProcessBlock(
	ctx context.Context,
	state *pb.BeaconState,
	block *pb.BeaconBlock,
	verifySignatures bool) (*pb.BeaconState, error) {

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
	log.WithField("blockRoot", fmt.Sprintf("%#x", r)).Debugf("Verified block slot == state slot")

	// Verify block signature.
	if verifySignatures {
		// TODO(#781): Verify Proposer Signature.
		if err := b.VerifyProposerSignature(ctx, block); err != nil {
			return nil, fmt.Errorf("could not verify proposer signature: %v", err)
		}
	}
	log.WithField("blockRoot", fmt.Sprintf("%#x", r)).Debugf("Verified block signature")

	// Verify block RANDAO.
	state, err = b.ProcessBlockRandao(ctx, state, block, verifySignatures)
	if err != nil {
		return nil, fmt.Errorf("could not verify and process block randao: %v", err)
	}
	log.WithField("blockRoot", fmt.Sprintf("%#x", r)).Debugf("Verified and processed block RANDAO")

	// Process ETH1 data.
	state = b.ProcessEth1DataInBlock(ctx, state, block)
	state, err = b.ProcessAttesterSlashings(ctx, state, block, verifySignatures)
	if err != nil {
		return nil, fmt.Errorf("could not verify block attester slashings: %v", err)
	}
	log.WithField("blockRoot", fmt.Sprintf("%#x", r)).Debugf("Processed ETH1 data")

	state, err = b.ProcessProposerSlashings(ctx, state, block, verifySignatures)
	if err != nil {
		return nil, fmt.Errorf("could not verify block proposer slashings: %v", err)
	}

	state, err = b.ProcessBlockAttestations(ctx, state, block, verifySignatures)
	if err != nil {
		return nil, fmt.Errorf("could not process block attestations: %v", err)
	}

	state, err = b.ProcessValidatorDeposits(ctx, state, block)
	if err != nil {
		return nil, fmt.Errorf("could not process block validator deposits: %v", err)
	}
	state, err = b.ProcessValidatorExits(ctx, state, block, verifySignatures)
	if err != nil {
		return nil, fmt.Errorf("could not process validator exits: %v", err)
	}

	log.WithField(
		"attestationsInBlock", len(block.Body.Attestations),
	).Info("Block attestations")
	log.WithField(
		"depositsInBlock", len(block.Body.Deposits),
	).Info("Block deposits")
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
func ProcessEpoch(ctx context.Context, state *pb.BeaconState) (*pb.BeaconState, error) {

	ctx, span := trace.StartSpan(ctx, "beacon-chain.ChainService.state.ProcessEpoch")
	defer span.End()

	currentEpoch := helpers.CurrentEpoch(state)
	prevEpoch := helpers.PrevEpoch(state)

	// Calculate total balances of active validators of the current epoch.
	activeValidatorIndices := helpers.ActiveValidatorIndices(state.ValidatorRegistry, currentEpoch)
	totalBalance := e.TotalBalance(ctx, state, activeValidatorIndices)

	// Calculate the attesting balances of validators that justified the
	// epoch boundary block at the start of the current epoch.
	currentEpochAttestations := e.CurrentAttestations(ctx, state)
	log.Infof("Number of current epoch attestations: %d", len(currentEpochAttestations))

	currentEpochBoundaryAttestations, err := e.CurrentEpochBoundaryAttestations(ctx, state, currentEpochAttestations)
	if err != nil {
		return nil, fmt.Errorf("could not get current boundary attestations: %v", err)
	}

	currentBoundaryAttesterIndices, err := v.ValidatorIndices(ctx, state, currentEpochBoundaryAttestations)
	if err != nil {
		return nil, fmt.Errorf("could not get current boundary attester indices: %v", err)
	}
	log.Infof("Current epoch boundary attester indices: %v", currentBoundaryAttesterIndices)

	currentBoundaryAttestingBalances := e.TotalBalance(ctx, state, currentBoundaryAttesterIndices)

	// Calculate the attesting balances of validators from previous epoch.
	previousActiveValidatorIndices := helpers.ActiveValidatorIndices(state.ValidatorRegistry, prevEpoch)
	prevTotalBalance := e.TotalBalance(ctx, state, previousActiveValidatorIndices)

	prevEpochAttestations := e.PrevAttestations(ctx, state)
	log.Infof("Number of prev epoch attestations: %d", len(prevEpochAttestations))
	prevEpochAttesterIndices, err := v.ValidatorIndices(ctx, state, prevEpochAttestations)
	if err != nil {
		return nil, fmt.Errorf("could not get prev epoch attester indices: %v", err)
	}
	log.Infof("Previous epoch attester indices: %v", prevEpochAttesterIndices)

	prevEpochAttestingBalance := e.TotalBalance(ctx, state, prevEpochAttesterIndices)

	// Calculate the attesting balances of validator justifying epoch boundary block
	// at the start of previous epoch.
	prevEpochBoundaryAttestations, err := e.PrevEpochBoundaryAttestations(ctx, state, prevEpochAttestations)
	if err != nil {
		return nil, fmt.Errorf("could not get prev boundary attestations: %v", err)
	}
	log.Infof("Number of prev epoch boundary attestations: %d", len(prevEpochAttestations))

	prevEpochBoundaryAttesterIndices, err := v.ValidatorIndices(ctx, state, prevEpochBoundaryAttestations)
	if err != nil {
		return nil, fmt.Errorf("could not get prev boundary attester indices: %v", err)
	}
	log.Infof("Previous epoch boundary attester indices: %v", prevEpochBoundaryAttesterIndices)

	prevEpochBoundaryAttestingBalances := e.TotalBalance(ctx, state, prevEpochBoundaryAttesterIndices)

	// Calculate attesting balances of validator attesting to expected beacon chain head
	// during previous epoch.
	prevEpochHeadAttestations, err := e.PrevHeadAttestations(ctx, state, prevEpochAttestations)
	if err != nil {
		return nil, fmt.Errorf("could not get prev head attestations: %v", err)
	}
	prevEpochHeadAttesterIndices, err := v.ValidatorIndices(ctx, state, prevEpochHeadAttestations)
	if err != nil {
		return nil, fmt.Errorf("could not get prev head attester indices: %v", err)
	}
	prevEpochHeadAttestingBalances := e.TotalBalance(ctx, state, prevEpochHeadAttesterIndices)

	// Process eth1 data.
	if e.CanProcessEth1Data(state) {
		state = e.ProcessEth1Data(ctx, state)
	}

	// Update justification and finality.
	state = e.ProcessJustification(
		ctx,
		state,
		currentBoundaryAttestingBalances,
		prevEpochAttestingBalance,
		prevTotalBalance,
		totalBalance,
	)

	// Process crosslinks records.
	state, err = e.ProcessCrosslinks(
		ctx,
		state,
		currentEpochAttestations,
		prevEpochAttestations)
	if err != nil {
		return nil, fmt.Errorf("could not process crosslink records: %v", err)
	}

	// Process attester rewards and penalties.
	epochsSinceFinality := e.SinceFinality(state)
	switch {
	case epochsSinceFinality <= 4:
		// Apply rewards/penalties to validators for attesting
		// expected FFG source.
		state = bal.ExpectedFFGSource(
			ctx,
			state,
			prevEpochAttesterIndices,
			prevEpochAttestingBalance,
			totalBalance)
		log.Infof("Balance after FFG src calculation: %v", state.ValidatorBalances)
		// Apply rewards/penalties to validators for attesting
		// expected FFG target.
		state = bal.ExpectedFFGTarget(
			ctx,
			state,
			prevEpochBoundaryAttesterIndices,
			prevEpochBoundaryAttestingBalances,
			totalBalance)
		log.Infof("Balance after FFG target calculation: %v", state.ValidatorBalances)
		// Apply rewards/penalties to validators for attesting
		// expected beacon chain head.
		state = bal.ExpectedBeaconChainHead(
			ctx,
			state,
			prevEpochHeadAttesterIndices,
			prevEpochHeadAttestingBalances,
			totalBalance)
		log.Infof("Balance after chain head calculation: %v", state.ValidatorBalances)
		// Apply rewards for to validators for including attestations
		// based on inclusion distance.
		state, err = bal.InclusionDistance(
			ctx,
			state,
			prevEpochAttesterIndices,
			totalBalance)
		if err != nil {
			return nil, fmt.Errorf("could not calculate inclusion dist rewards: %v", err)
		}
		log.Infof("Balance after inclusion distance calculation: %v", state.ValidatorBalances)

	case epochsSinceFinality > 4:
		log.Infof("Applying more penalties. ESF %d greater than 4", epochsSinceFinality)
		// Apply penalties for long inactive FFG source participants.
		state = bal.InactivityFFGSource(
			ctx,
			state,
			prevEpochAttesterIndices,
			totalBalance,
			epochsSinceFinality)
		// Apply penalties for long inactive FFG target participants.
		state = bal.InactivityFFGTarget(
			ctx,
			state,
			prevEpochBoundaryAttesterIndices,
			totalBalance,
			epochsSinceFinality)
		// Apply penalties for long inactive validators who didn't
		// attest to head canonical chain.
		state = bal.InactivityChainHead(
			ctx,
			state,
			prevEpochHeadAttesterIndices,
			totalBalance)
		// Apply penalties for long inactive validators who also
		// exited with penalties.
		state = bal.InactivityExitedPenalties(
			ctx,
			state,
			totalBalance,
			epochsSinceFinality)
		// Apply penalties for long inactive validators that
		// don't include attestations.
		state, err = bal.InactivityInclusionDistance(
			ctx,
			state,
			prevEpochAttesterIndices,
			totalBalance)
		if err != nil {
			return nil, fmt.Errorf("could not calculate inclusion penalties: %v", err)
		}
	}

	// Process Attestation Inclusion Rewards.
	state, err = bal.AttestationInclusion(
		ctx,
		state,
		totalBalance,
		prevEpochAttesterIndices)
	if err != nil {
		return nil, fmt.Errorf("could not process attestation inclusion rewards: %v", err)
	}

	// Process crosslink rewards and penalties.
	state, err = bal.Crosslinks(
		ctx,
		state,
		currentEpochAttestations,
		prevEpochAttestations)
	if err != nil {
		return nil, fmt.Errorf("could not process crosslink rewards and penalties: %v", err)
	}

	// Process ejections.
	state, err = e.ProcessEjections(ctx, state)
	if err != nil {
		return nil, fmt.Errorf("could not process ejections: %v", err)
	}

	// Process validator registry.
	state = e.ProcessPrevSlotShardSeed(state)
	state = v.ProcessPenaltiesAndExits(ctx, state)
	if e.CanProcessValidatorRegistry(ctx, state) {
		state, err = v.UpdateRegistry(ctx, state)
		if err != nil {
			return nil, fmt.Errorf("could not update validator registry: %v", err)
		}
		state, err = e.ProcessCurrSlotShardSeed(state)
		if err != nil {
			return nil, fmt.Errorf("could not update current shard shuffling seeds: %v", err)
		}
	} else {
		state, err = e.ProcessPartialValidatorRegistry(ctx, state)
		if err != nil {
			return nil, fmt.Errorf("could not process partial validator registry: %v", err)
		}
	}

	// Final housekeeping updates.
	// Update index roots from current epoch to next epoch.
	state, err = e.UpdateLatestActiveIndexRoots(ctx, state)
	if err != nil {
		return nil, fmt.Errorf("could not update latest index roots: %v", err)
	}

	// TODO(1763): Implement process_slashings from ETH2.0 beacon chain spec.

	// TODO(1764): Implement process_exit_queue from ETH2.0 beacon chain spec.

	// Update accumulated slashed balances from current epoch to next epoch.
	state = e.UpdateLatestSlashedBalances(ctx, state)

	// Update current epoch's randao seed to next epoch.
	state, err = e.UpdateLatestRandaoMixes(ctx, state)
	if err != nil {
		return nil, fmt.Errorf("could not update latest randao mixes: %v", err)
	}

	// Clean up processed attestations.
	state = e.CleanupAttestations(ctx, state)

	log.WithField(
		"PreviousJustifiedEpoch", state.PreviousJustifiedEpoch-params.BeaconConfig().GenesisEpoch,
	).Info("Previous justified epoch")
	log.WithField(
		"JustifiedEpoch", state.JustifiedEpoch-params.BeaconConfig().GenesisEpoch,
	).Info("Justified epoch")
	log.WithField(
		"FinalizedEpoch", state.FinalizedEpoch-params.BeaconConfig().GenesisEpoch,
	).Info("Finalized epoch")
	log.WithField(
		"ValidatorRegistryUpdateEpoch", state.ValidatorRegistryUpdateEpoch-params.BeaconConfig().GenesisEpoch,
	).Info("Validator Registry Update Epoch")
	log.WithField(
		"NumValidators", len(state.ValidatorRegistry),
	).Info("Validator registry length")
	log.Infof("Validator balances: %v", state.ValidatorBalances)
	log.WithField(
		"ValidatorRegistryUpdateEpoch", state.ValidatorRegistryUpdateEpoch-params.BeaconConfig().GenesisEpoch,
	).Info("Validator registry update epoch")

	// Report interesting metrics.
	reportEpochTransitionMetrics(state)
	return state, nil
}
