// Package state implements the whole state transition
// function which consists of per slot, per-epoch transitions.
// It also bootstraps the genesis beacon state for slot 0.
package state

import (
	"fmt"

	bal "github.com/prysmaticlabs/prysm/beacon-chain/core/balances"
	b "github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	e "github.com/prysmaticlabs/prysm/beacon-chain/core/epoch"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/randao"
	v "github.com/prysmaticlabs/prysm/beacon-chain/core/validators"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

// ExecuteStateTransition defines the procedure for a state transition function.
// Spec:
//  We now define the state transition function. At a high level the state transition is made up of two parts:
//  - The per-slot transitions, which happens every slot, and only affects a parts of the state.
//  - The per-epoch transitions, which happens at every epoch boundary (i.e. state.slot % EPOCH_LENGTH == 0), and affects the entire state.
//  The per-slot transitions generally focus on verifying aggregate signatures and saving temporary records relating to the per-slot
//  activity in the BeaconState. The per-epoch transitions focus on the validator registry, including adjusting balances and activating
//  and exiting validators, as well as processing crosslinks and managing block justification/finalization.
func ExecuteStateTransition(
	beaconState *pb.BeaconState,
	block *pb.BeaconBlock,
	prevBlockRoot [32]byte,
) (*pb.BeaconState, error) {
	var err error

	currentSlot := beaconState.Slot
	beaconState.Slot = currentSlot + 1

	beaconState, err = randao.UpdateRandaoLayers(beaconState, beaconState.Slot)
	if err != nil {
		return nil, fmt.Errorf("unable to update randao layer %v", err)
	}
	beaconState = randao.UpdateRandaoMixes(beaconState)
	beaconState = b.ProcessBlockRoots(beaconState, prevBlockRoot)
	if block != nil {
		beaconState, err = ProcessBlock(beaconState, block)
		if err != nil {
			return nil, fmt.Errorf("unable to process block: %v", err)
		}

		if e.CanProcessEpoch(beaconState) {
			beaconState, err = ProcessEpoch(beaconState)
		}
		if err != nil {
			return nil, fmt.Errorf("unable to process epoch: %v", err)
		}
	}

	return beaconState, nil
}

// ProcessBlock creates a new, modified beacon state by applying block operation
// transformations as defined in the Ethereum Serenity specification, including processing proposer slashings,
// processing block attestations, and more.
func ProcessBlock(state *pb.BeaconState, block *pb.BeaconBlock) (*pb.BeaconState, error) {
	if block.Slot != state.Slot {
		return nil, fmt.Errorf(
			"block.slot != state.slot, block.slot = %d, state.slot = %d",
			block.Slot,
			state.Slot,
		)
	}
	// TODO(#781): Verify Proposer Signature.
	var err error
	state = b.ProcessDepositRoots(state, block)
	state, err = b.ProcessBlockRandao(state, block)
	if err != nil {
		return nil, fmt.Errorf("could not verify and process block randao: %v", err)
	}
	state, err = b.ProcessProposerSlashings(state, block)
	if err != nil {
		return nil, fmt.Errorf("could not verify block proposer slashings: %v", err)
	}
	state, err = b.ProcessCasperSlashings(state, block)
	if err != nil {
		return nil, fmt.Errorf("could not verify block casper slashings: %v", err)
	}
	state, err = b.ProcessBlockAttestations(state, block)
	if err != nil {
		return nil, fmt.Errorf("could not process block attestations: %v", err)
	}
	state, err = b.ProcessValidatorDeposits(state, block)
	if err != nil {
		return nil, fmt.Errorf("could not process block validator deposits: %v", err)
	}
	state, err = b.ProcessValidatorExits(state, block)
	if err != nil {
		return nil, fmt.Errorf("could not process validator exits: %v", err)
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
// 	 process_casper_reward_penalties(state)
// 	 process_crosslink_reward_penalties(state)
// 	 update_validator_registry(state)
// 	 final_book_keeping(state)
func ProcessEpoch(state *pb.BeaconState) (*pb.BeaconState, error) {

	// Calculate total balances of active validators of the current state.
	activeValidatorIndices := v.ActiveValidatorIndices(state.ValidatorRegistry, state.Slot)
	totalBalance := e.TotalBalance(state, activeValidatorIndices)

	// Calculate the attesting balances of validators that justified the
	// epoch boundary block at the start of the current epoch.
	currentAttestations := e.Attestations(state)
	currentBoundaryAttestations, err := e.BoundaryAttestations(state, currentAttestations)
	if err != nil {
		return nil, fmt.Errorf("could not get current boundary attestations: %v", err)
	}
	currentBoundaryAttesterIndices, err := v.ValidatorIndices(state, currentBoundaryAttestations)
	if err != nil {
		return nil, fmt.Errorf("could not get current boundary attester indices: %v", err)
	}
	currentBoundaryAttestingBalances := e.TotalBalance(state, currentBoundaryAttesterIndices)

	// Calculate the attesting balances of validators that made an attestation
	// during previous epoch.
	prevAttestations := e.PrevAttestations(state)
	prevAttesterIndices, err := v.ValidatorIndices(state, prevAttestations)
	if err != nil {
		return nil, fmt.Errorf("could not get prev epoch attester indices: %v", err)
	}

	// Calculate the attesting balances of validators that targeted
	// previous justified hash.
	prevJustifiedAttestations := e.PrevJustifiedAttestations(state,
		currentAttestations, prevAttestations)

	prevJustifiedAttesterIndices, err := v.ValidatorIndices(state, prevJustifiedAttestations)
	if err != nil {
		return nil, fmt.Errorf("could not get prev epoch justified attester indices: %v", err)
	}
	prevJustifiedAttestingBalance := e.TotalBalance(state, prevJustifiedAttesterIndices)

	// Calculate the attesting balances of validator justifying epoch boundary block
	// at the start of previous epoch.
	prevBoundaryAttestations, err := e.PrevBoundaryAttestations(state, prevJustifiedAttestations)
	if err != nil {
		return nil, fmt.Errorf("could not get prev boundary attestations: %v", err)
	}
	prevBoundaryAttesterIndices, err := v.ValidatorIndices(state, prevBoundaryAttestations)
	if err != nil {
		return nil, fmt.Errorf("could not get prev boundary attester indices: %v", err)
	}
	prevBoundaryAttestingBalances := e.TotalBalance(state, prevBoundaryAttesterIndices)

	// Calculate attesting balances of validator attesting to expected beacon chain head
	// during previous epoch.
	prevHeadAttestations, err := e.PrevHeadAttestations(state, prevAttestations)
	if err != nil {
		return nil, fmt.Errorf("could not get prev head attestations: %v", err)
	}
	prevHeadAttesterIndices, err := v.ValidatorIndices(state, prevHeadAttestations)
	if err != nil {
		return nil, fmt.Errorf("could not get prev head attester indices: %v", err)
	}
	prevHeadAttestingBalances := e.TotalBalance(state, prevHeadAttesterIndices)

	// Process receipt roots.
	if e.CanProcessDepositRoots(state) {
		e.ProcessDeposits(state)
	}

	// Update justification.
	state = e.ProcessJustification(
		state,
		currentBoundaryAttestingBalances,
		prevBoundaryAttestingBalances,
		totalBalance)

	// Update Finalization.
	state = e.ProcessFinalization(state)

	// Process crosslinks records.
	state, err = e.ProcessCrosslinks(
		state,
		currentAttestations,
		prevAttestations)
	if err != nil {
		return nil, fmt.Errorf("could not process crosslink records: %v", err)
	}

	// Process casper rewards and penalties.
	epochsSinceFinality := e.SinceFinality(state)
	switch {
	case epochsSinceFinality <= 4:
		// Apply rewards/penalties to validators for attesting
		// expected FFG source.
		state = bal.ExpectedFFGSource(
			state,
			prevJustifiedAttesterIndices,
			prevJustifiedAttestingBalance,
			totalBalance)
		// Apply rewards/penalties to validators for attesting
		// expected FFG target.
		state = bal.ExpectedFFGTarget(
			state,
			prevBoundaryAttesterIndices,
			prevBoundaryAttestingBalances,
			totalBalance)
		// Apply rewards/penalties to validators for attesting
		// expected beacon chain head.
		state = bal.ExpectedBeaconChainHead(
			state,
			prevHeadAttesterIndices,
			prevHeadAttestingBalances,
			totalBalance)
		// Apply rewards for to validators for including attestations
		// based on inclusion distance.
		state, err = bal.InclusionDistance(
			state,
			prevAttesterIndices,
			totalBalance)
		if err != nil {
			return nil, fmt.Errorf("could not calculate inclusion dist rewards: %v", err)
		}

	case epochsSinceFinality > 4:
		// Apply penalties for long inactive FFG source participants.
		state = bal.InactivityFFGSource(
			state,
			prevJustifiedAttesterIndices,
			totalBalance,
			epochsSinceFinality)
		// Apply penalties for long inactive FFG target participants.
		state = bal.InactivityFFGTarget(
			state,
			prevBoundaryAttesterIndices,
			totalBalance,
			epochsSinceFinality)
		// Apply penalties for long inactive validators who didn't
		// attest to head canonical chain.
		state = bal.InactivityChainHead(
			state,
			prevHeadAttesterIndices,
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
			prevAttesterIndices,
			totalBalance)
		if err != nil {
			return nil, fmt.Errorf("could not calculate inclusion penalties: %v", err)
		}
	}

	// Process Attestation Inclusion Rewards.
	state, err = bal.AttestationInclusion(
		state,
		totalBalance,
		prevAttesterIndices)
	if err != nil {
		return nil, fmt.Errorf("could not process attestation inclusion rewards: %v", err)
	}

	// Process crosslink rewards and penalties.
	state, err = bal.Crosslinks(
		state,
		currentAttestations,
		prevAttestations)
	if err != nil {
		return nil, fmt.Errorf("could not process crosslink rewards and penalties: %v", err)
	}

	// Process ejections.
	state, err = e.ProcessEjections(state)
	if err != nil {
		return nil, fmt.Errorf("could not process ejections: %v", err)
	}

	// Process validator registry.
	if e.CanProcessValidatorRegistry(state) {
		state = v.ProcessPenaltiesAndExits(state)
		state, err = e.ProcessValidatorRegistry(state)
		if err != nil {
			return nil, fmt.Errorf("could not process validator registry: %v", err)
		}
	} else {
		state = v.ProcessPenaltiesAndExits(state)
		state, err = e.ProcessPartialValidatorRegistry(state)
		if err != nil {
			return nil, fmt.Errorf("could not process partial validator registry: %v", err)
		}
	}

	// Clean up processed attestations.
	state = e.CleanupAttestations(state)

	return state, nil
}
