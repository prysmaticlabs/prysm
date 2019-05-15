// Package balances contains libraries to calculate reward and
// penalty quotients. It computes new validator balances
// for justifications, crosslinks and attestation inclusions. It
// also computes penalties for the inactive validators.
package balances

import (
	"errors"
	"fmt"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/sliceutil"
)

// ExpectedFFGSource applies rewards or penalties
// for an expected FFG source. It uses total justified
// attesting balances, total validator balances and base
// reward quotient to calculate the reward amount.
// Validators who voted for previous justified hash
// will get a reward, everyone else will get a penalty.
//
// Spec pseudocode definition:
//    Any validator index in previous_epoch_justified_attester_indices
//    gains base_reward(state, index) * previous_epoch_justified_attesting_balance // total_balance.
//	  Any active validator v not in previous_epoch_justified_attester_indices
//	  loses base_reward(state, index).
func ExpectedFFGSource(
	state *pb.BeaconState,
	justifiedAttesterIndices []uint64,
	justifiedAttestingBalance uint64,
	totalBalance uint64) *pb.BeaconState {
	baseRewardQuotient := helpers.BaseRewardQuotient(totalBalance)

	for _, index := range justifiedAttesterIndices {
		state.Balances[index] +=
			helpers.BaseReward(state, index, baseRewardQuotient) *
				justifiedAttestingBalance /
				totalBalance
	}
	activeValidatorIndices := helpers.ActiveValidatorIndices(state, helpers.CurrentEpoch(state))
	didNotAttestIndices := sliceutil.NotUint64(justifiedAttesterIndices, activeValidatorIndices)

	for _, index := range didNotAttestIndices {
		state.Balances[index] -=
			helpers.BaseReward(state, index, baseRewardQuotient)
	}
	return state
}

// ExpectedFFGTarget applies rewards or penalties
// for an expected FFG target. It uses total boundary
// attesting balances, total validator balances and base
// reward quotient to calculate the reward amount.
// Validators who voted for epoch boundary block
// will get a reward, everyone else will get a penalty.
//
// Spec pseudocode definition:
//    Any validator index in previous_epoch_boundary_attester_indices gains
//    base_reward(state, index) * previous_epoch_boundary_attesting_balance // total_balance.
//	  Any active validator index not in previous_epoch_boundary_attester_indices loses
//	  base_reward(state, index).
func ExpectedFFGTarget(
	state *pb.BeaconState,
	boundaryAttesterIndices []uint64,
	boundaryAttestingBalance uint64,
	totalBalance uint64) *pb.BeaconState {

	baseRewardQuotient := helpers.BaseRewardQuotient(totalBalance)

	for _, index := range boundaryAttesterIndices {
		state.Balances[index] +=
			helpers.BaseReward(state, index, baseRewardQuotient) *
				boundaryAttestingBalance /
				totalBalance
	}
	activeValidatorIndices := helpers.ActiveValidatorIndices(state, helpers.CurrentEpoch(state))
	didNotAttestIndices := sliceutil.NotUint64(boundaryAttesterIndices, activeValidatorIndices)

	for _, index := range didNotAttestIndices {
		state.Balances[index] -=
			helpers.BaseReward(state, index, baseRewardQuotient)
	}
	return state
}

// ExpectedBeaconChainHead applies rewards or penalties
// for an expected beacon chain head. It uses total head
// attesting balances, total validator balances and base
// reward quotient to calculate the reward amount.
// Validators who voted for the canonical head block
// will get a reward, everyone else will get a penalty.
//
// Spec pseudocode definition:
//    Any validator index in previous_epoch_head_attester_indices gains
//    base_reward(state, index) * previous_epoch_head_attesting_balance // total_balance).
//    Any active validator index not in previous_epoch_head_attester_indices loses
//    base_reward(state, index).
func ExpectedBeaconChainHead(
	state *pb.BeaconState,
	headAttesterIndices []uint64,
	headAttestingBalance uint64,
	totalBalance uint64) *pb.BeaconState {

	baseRewardQuotient := helpers.BaseRewardQuotient(totalBalance)

	for _, index := range headAttesterIndices {
		state.Balances[index] +=
			helpers.BaseReward(state, index, baseRewardQuotient) *
				headAttestingBalance /
				totalBalance
	}
	activeValidatorIndices := helpers.ActiveValidatorIndices(state, helpers.CurrentEpoch(state))
	didNotAttestIndices := sliceutil.NotUint64(headAttesterIndices, activeValidatorIndices)

	for _, index := range didNotAttestIndices {
		state.Balances[index] -=
			helpers.BaseReward(state, index, baseRewardQuotient)
	}
	return state
}

// InclusionDistance applies rewards based on
// inclusion distance. It uses calculated inclusion distance
// and base reward quotient to calculate the reward amount.
//
// Spec pseudocode definition:
//    Any validator index in previous_epoch_attester_indices gains
//    base_reward(state, index) * MIN_ATTESTATION_INCLUSION_DELAY //
//    inclusion_distance(state, index)
func InclusionDistance(
	state *pb.BeaconState,
	attesterIndices []uint64,
	totalBalance uint64,
	inclusionDistanceByAttester map[uint64]uint64) (*pb.BeaconState, error) {

	baseRewardQuotient := helpers.BaseRewardQuotient(totalBalance)

	for _, index := range attesterIndices {
		inclusionDistance, ok := inclusionDistanceByAttester[index]
		if !ok {
			return nil, fmt.Errorf("could not get inclusion distance for attester: %d", index)
		}
		if inclusionDistance == 0 {
			return nil, errors.New("could not process inclusion distance: 0")
		}
		state.Balances[index] +=
			helpers.BaseReward(state, index, baseRewardQuotient) *
				params.BeaconConfig().MinAttestationInclusionDelay /
				inclusionDistance
	}
	return state, nil
}

// InactivityFFGSource applies penalties to inactive
// validators that missed to vote FFG source over an
// extended of time. (epochs_since_finality > 4)
//
// Spec pseudocode definition:
//    Any active validator index not in previous_epoch_justified_attester_indices,
//    loses inactivity_penalty(state, index, epochs_since_finality)
func InactivityFFGSource(
	state *pb.BeaconState,
	justifiedAttesterIndices []uint64,
	totalBalance uint64,
	epochsSinceFinality uint64) *pb.BeaconState {

	baseRewardQuotient := helpers.BaseRewardQuotient(totalBalance)
	activeValidatorIndices := helpers.ActiveValidatorIndices(state, helpers.CurrentEpoch(state))
	didNotAttestIndices := sliceutil.NotUint64(justifiedAttesterIndices, activeValidatorIndices)

	for _, index := range didNotAttestIndices {
		state.Balances[index] -=
			helpers.InactivityPenalty(state, index, baseRewardQuotient, epochsSinceFinality)
	}
	return state
}

// InactivityFFGTarget applies penalties to inactive
// validators that missed to vote FFG target over an
// extended of time. (epochs_since_finality > 4)
//
// Spec pseudocode definition:
//    Any active validator index not in previous_epoch_boundary_attester_indices,
// 	  loses inactivity_penalty(state, index, epochs_since_finality)
func InactivityFFGTarget(
	state *pb.BeaconState,
	boundaryAttesterIndices []uint64,
	totalBalance uint64,
	epochsSinceFinality uint64) *pb.BeaconState {

	baseRewardQuotient := helpers.BaseRewardQuotient(totalBalance)
	activeValidatorIndices := helpers.ActiveValidatorIndices(state, helpers.CurrentEpoch(state))
	didNotAttestIndices := sliceutil.NotUint64(boundaryAttesterIndices, activeValidatorIndices)

	for _, index := range didNotAttestIndices {
		state.Balances[index] -=
			helpers.InactivityPenalty(state, index, baseRewardQuotient, epochsSinceFinality)
	}
	return state
}

// InactivityChainHead applies penalties to inactive validators
// that missed to vote on canonical head over an extended of time.
// (epochs_since_finality > 4)
//
// Spec pseudocode definition:
//    Any active validator index not in previous_epoch_head_attester_indices,
// 	  loses base_reward(state, index)
func InactivityChainHead(
	state *pb.BeaconState,
	headAttesterIndices []uint64,
	totalBalance uint64) *pb.BeaconState {

	baseRewardQuotient := helpers.BaseRewardQuotient(totalBalance)
	activeValidatorIndices := helpers.ActiveValidatorIndices(state, helpers.CurrentEpoch(state))
	didNotAttestIndices := sliceutil.NotUint64(headAttesterIndices, activeValidatorIndices)

	for _, index := range didNotAttestIndices {
		state.Balances[index] -=
			helpers.BaseReward(state, index, baseRewardQuotient)
	}
	return state
}

// InactivityExitedPenalties applies additional (2x) penalties
// to inactive validators with status EXITED_WITH_PENALTY.
//
// Spec pseudocode definition:
//    Any active_validator index with validator.slashed_epoch <= current_epoch,
//    loses 2 * inactivity_penalty(state, index, epochs_since_finality) +
//    base_reward(state, index).
func InactivityExitedPenalties(
	state *pb.BeaconState,
	totalBalance uint64,
	epochsSinceFinality uint64) *pb.BeaconState {

	baseRewardQuotient := helpers.BaseRewardQuotient(totalBalance)
	currentEpoch := helpers.CurrentEpoch(state)
	activeValidatorIndices := helpers.ActiveValidatorIndices(state, currentEpoch)

	for _, index := range activeValidatorIndices {
		if state.ValidatorRegistry[index].SlashedEpoch <= currentEpoch {
			state.Balances[index] -=
				2*helpers.InactivityPenalty(state, index, baseRewardQuotient, epochsSinceFinality) +
					helpers.BaseReward(state, index, baseRewardQuotient)
		}
	}
	return state
}

// InactivityInclusionDistance applies penalties in relation with
// inclusion delay to inactive validators.
//
// Spec pseudocode definition:
//    Any validator index in previous_epoch_attester_indices loses
//    base_reward(state, index) - base_reward(state, index) *
//    MIN_ATTESTATION_INCLUSION_DELAY // inclusion_distance(state, index)
func InactivityInclusionDistance(
	state *pb.BeaconState,
	attesterIndices []uint64,
	totalBalance uint64,
	inclusionDistanceByAttester map[uint64]uint64) (*pb.BeaconState, error) {
	baseRewardQuotient := helpers.BaseRewardQuotient(totalBalance)

	for _, index := range attesterIndices {
		inclusionDistance, ok := inclusionDistanceByAttester[index]
		if !ok {
			return nil, fmt.Errorf("could not get inclusion distance for attester: %d", index)
		}
		baseReward := helpers.BaseReward(state, index, baseRewardQuotient)
		state.Balances[index] -= baseReward -
			baseReward*params.BeaconConfig().MinAttestationInclusionDelay/
				inclusionDistance
	}
	return state, nil
}

// AttestationInclusion awards the the beacon
// proposers who included previous epoch attestations.
//
// Spec pseudocode definition:
//    For each index in previous_epoch_attester_indices,
//    we determine the proposer proposer_index =
//    get_beacon_proposer_index(state, inclusion_slot(state, index))
//    and set state.validator_balances[proposer_index] +=
//    base_reward(state, index) // ATTESTATION_INCLUSION_REWARD_QUOTIENT
func AttestationInclusion(
	state *pb.BeaconState,
	totalBalance uint64,
	prevEpochAttesterIndices []uint64,
	inclusionSlotByAttester map[uint64]uint64) (*pb.BeaconState, error) {

	baseRewardQuotient := helpers.BaseRewardQuotient(totalBalance)
	for _, index := range prevEpochAttesterIndices {
		// Get the attestation's inclusion slot using the attestor's index.
		slot, ok := inclusionSlotByAttester[index]
		if !ok {
			return nil, fmt.Errorf("could not get inclusion slot for attester: %d", index)
		}
		state.Slot = slot
		proposerIndex, err := helpers.BeaconProposerIndex(state)
		if err != nil {
			return nil, fmt.Errorf("could not get proposer index: %v", err)
		}
		state.Balances[proposerIndex] +=
			helpers.BaseReward(state, proposerIndex, baseRewardQuotient) /
				params.BeaconConfig().AttestationInclusionRewardQuotient
	}
	return state, nil
}
