// Package balances contains libraries to calculate reward and
// penalty quotients. It computes new validator balances
// for justifications, crosslinks and attestation inclusions. It
// also computes penalties for the inactive validators.
package balances

import (
	"fmt"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/epoch"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/validators"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/mathutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/slices"
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
	justifiedAttesterIndices []uint32,
	justifiedAttestingBalance uint64,
	totalBalance uint64) *pb.BeaconState {

	baseRewardQuotient := baseRewardQuotient(totalBalance)

	for _, index := range justifiedAttesterIndices {
		state.ValidatorBalances[index] +=
			baseReward(state, index, baseRewardQuotient) *
				justifiedAttestingBalance /
				totalBalance
	}

	activeValidatorIndices := validators.ActiveValidatorIndices(state.ValidatorRegistry, state.Slot)
	didNotAttestIndices := slices.Not(justifiedAttesterIndices, activeValidatorIndices)

	for _, index := range didNotAttestIndices {
		state.ValidatorBalances[index] -=
			baseReward(state, index, baseRewardQuotient)
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
	boundaryAttesterIndices []uint32,
	boundaryAttestingBalance uint64,
	totalBalance uint64) *pb.BeaconState {

	baseRewardQuotient := baseRewardQuotient(totalBalance)

	for _, index := range boundaryAttesterIndices {
		state.ValidatorBalances[index] +=
			baseReward(state, index, baseRewardQuotient) *
				boundaryAttestingBalance /
				totalBalance
	}

	activeValidatorIndices := validators.ActiveValidatorIndices(state.ValidatorRegistry, state.Slot)
	didNotAttestIndices := slices.Not(boundaryAttesterIndices, activeValidatorIndices)

	for _, index := range didNotAttestIndices {
		state.ValidatorBalances[index] -=
			baseReward(state, index, baseRewardQuotient)
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
	headAttesterIndices []uint32,
	headAttestingBalance uint64,
	totalBalance uint64) *pb.BeaconState {

	baseRewardQuotient := baseRewardQuotient(totalBalance)

	for _, index := range headAttesterIndices {
		state.ValidatorBalances[index] +=
			baseReward(state, index, baseRewardQuotient) *
				headAttestingBalance /
				totalBalance
	}

	activeValidatorIndices := validators.ActiveValidatorIndices(state.ValidatorRegistry, state.Slot)
	didNotAttestIndices := slices.Not(headAttesterIndices, activeValidatorIndices)

	for _, index := range didNotAttestIndices {
		state.ValidatorBalances[index] -=
			baseReward(state, index, baseRewardQuotient)
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
	attesterIndices []uint32,
	totalBalance uint64) (*pb.BeaconState, error) {

	baseRewardQuotient := baseRewardQuotient(totalBalance)

	for _, index := range attesterIndices {
		inclusionDistance, err := epoch.InclusionDistance(state, index)
		if err != nil {
			return nil, fmt.Errorf("could not get inclusion distance: %v", err)
		}
		state.ValidatorBalances[index] +=
			baseReward(state, index, baseRewardQuotient) *
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
	justifiedAttesterIndices []uint32,
	totalBalance uint64,
	epochsSinceFinality uint64) *pb.BeaconState {

	baseRewardQuotient := baseRewardQuotient(totalBalance)
	activeValidatorIndices := validators.ActiveValidatorIndices(state.ValidatorRegistry, state.Slot)
	didNotAttestIndices := slices.Not(justifiedAttesterIndices, activeValidatorIndices)

	for _, index := range didNotAttestIndices {
		state.ValidatorBalances[index] -=
			inactivityPenalty(state, index, baseRewardQuotient, epochsSinceFinality)
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
	boundaryAttesterIndices []uint32,
	totalBalance uint64,
	epochsSinceFinality uint64) *pb.BeaconState {

	baseRewardQuotient := baseRewardQuotient(totalBalance)
	activeValidatorIndices := validators.ActiveValidatorIndices(state.ValidatorRegistry, state.Slot)
	didNotAttestIndices := slices.Not(boundaryAttesterIndices, activeValidatorIndices)

	for _, index := range didNotAttestIndices {
		state.ValidatorBalances[index] -=
			inactivityPenalty(state, index, baseRewardQuotient, epochsSinceFinality)
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
	headAttesterIndices []uint32,
	totalBalance uint64) *pb.BeaconState {

	baseRewardQuotient := baseRewardQuotient(totalBalance)
	activeValidatorIndices := validators.ActiveValidatorIndices(state.ValidatorRegistry, state.Slot)
	didNotAttestIndices := slices.Not(headAttesterIndices, activeValidatorIndices)

	for _, index := range didNotAttestIndices {
		state.ValidatorBalances[index] -=
			baseReward(state, index, baseRewardQuotient)
	}
	return state
}

// InactivityExitedPenalties applies additional (2x) penalties
// to inactive validators with status EXITED_WITH_PENALTY.
//
// Spec pseudocode definition:
//    Any active_validator index with validator.penalized_slot <= state.slot,
//    loses 2 * inactivity_penalty(state, index, epochs_since_finality) +
//    base_reward(state, index).
func InactivityExitedPenalties(
	state *pb.BeaconState,
	totalBalance uint64,
	epochsSinceFinality uint64) *pb.BeaconState {

	baseRewardQuotient := baseRewardQuotient(totalBalance)
	activeValidatorIndices := validators.ActiveValidatorIndices(state.ValidatorRegistry, state.Slot)

	for _, index := range activeValidatorIndices {
		if state.ValidatorRegistry[index].PenalizedSlot <= state.Slot {
			state.ValidatorBalances[index] -=
				2*inactivityPenalty(state, index, baseRewardQuotient, epochsSinceFinality) +
					baseReward(state, index, baseRewardQuotient)
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
	attesterIndices []uint32,
	totalBalance uint64) (*pb.BeaconState, error) {

	baseRewardQuotient := baseRewardQuotient(totalBalance)

	for _, index := range attesterIndices {
		inclusionDistance, err := epoch.InclusionDistance(state, index)
		if err != nil {
			return nil, fmt.Errorf("could not get inclusion distance: %v", err)
		}
		baseReward := baseReward(state, index, baseRewardQuotient)
		state.ValidatorBalances[index] -= baseReward -
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
//    base_reward(state, index) // INCLUDER_REWARD_QUOTIENT
func AttestationInclusion(
	state *pb.BeaconState,
	totalBalance uint64,
	prevEpochAttesterIndices []uint32) (*pb.BeaconState, error) {

	baseRewardQuotient := baseRewardQuotient(totalBalance)
	for _, index := range prevEpochAttesterIndices {
		slot, err := epoch.InclusionSlot(state, index)
		if err != nil {
			return nil, fmt.Errorf("could not get inclusion slot: %v", err)
		}
		proposerIndex, err := validators.BeaconProposerIdx(state, slot)
		if err != nil {
			return nil, fmt.Errorf("could not get propoer index: %v", err)
		}
		state.ValidatorBalances[proposerIndex] +=
			baseReward(state, proposerIndex, baseRewardQuotient) /
				params.BeaconConfig().IncluderRewardQuotient
	}
	return state, nil
}

// Crosslinks awards or penalizes attesters
// for attesting shard cross links.
//
// Spec pseudocode definition:
//    For every shard_committee in state.shard_committees_at_slots[:EPOCH_LENGTH]:
// 	  	For each index in shard_committee.committee, adjust balances as follows:
// 			If index in attesting_validators(shard_committee), state.validator_balances[index]
// 				+= base_reward(state, index) * total_attesting_balance(shard_committee)
// 				   total_balance(shard_committee)).
//			If index not in attesting_validators(shard_committee), state.validator_balances[index]
// 				-= base_reward(state, index).
func Crosslinks(
	state *pb.BeaconState,
	thisEpochAttestations []*pb.PendingAttestationRecord,
	prevEpochAttestations []*pb.PendingAttestationRecord) (*pb.BeaconState, error) {

	epochLength := params.BeaconConfig().EpochLength

	for _, shardCommitteesAtSlot := range state.ShardCommitteesAtSlots[:epochLength] {
		for _, shardCommitee := range shardCommitteesAtSlot.ArrayShardCommittee {
			totalAttestingBalance, err :=
				epoch.TotalAttestingBalance(state, shardCommitee, thisEpochAttestations, prevEpochAttestations)
			if err != nil {
				return nil,
					fmt.Errorf("could not get attesting balance for shard committee %d: %v", shardCommitee.Shard, err)
			}
			totalBalance := epoch.TotalBalance(state, shardCommitee.Committee)
			baseRewardQuotient := baseRewardQuotient(totalBalance)

			attestingIndices, err := epoch.AttestingValidators(
				state,
				shardCommitee,
				thisEpochAttestations,
				prevEpochAttestations)
			if err != nil {
				return nil,
					fmt.Errorf("could not get attesting indices for shard committee %d: %v", shardCommitee.Shard, err)
			}
			for _, index := range shardCommitee.Committee {
				baseReward := baseReward(state, index, baseRewardQuotient)
				if slices.IsIn(index, attestingIndices) {
					state.ValidatorBalances[index] +=
						baseReward * totalAttestingBalance / totalBalance
				} else {
					state.ValidatorBalances[index] -=
						baseReward * totalAttestingBalance / totalBalance
				}
			}
		}
	}
	return state, nil
}

// baseRewardQuotient takes the total balance and calculates for
// the quotient of the base reward.
//
// Spec pseudocode definition:
//    base_reward_quotient =
//    	BASE_REWARD_QUOTIENT * integer_squareroot(total_balance // GWEI_PER_ETH)
func baseRewardQuotient(totalBalance uint64) uint64 {

	baseRewardQuotient := params.BeaconConfig().BaseRewardQuotient * mathutil.IntegerSquareRoot(
		totalBalance/params.BeaconConfig().Gwei)

	return baseRewardQuotient
}

// baseReward takes state and validator index to calculate for
// individual validator's base reward.
//
// Spec pseudocode definition:
//    base_reward(state, index) =
//    	get_effective_balance(state, index) // base_reward_quotient // 5
func baseReward(
	state *pb.BeaconState,
	validatorIndex uint32,
	baseRewardQuotient uint64) uint64 {

	validatorBalance := validators.EffectiveBalance(state, validatorIndex)
	return validatorBalance / baseRewardQuotient / 5
}

// inactivityPenalty takes state and validator index to calculate for
// individual validator's penalty for being offline.
//
// Spec pseudocode definition:
//    inactivity_penalty(state, index, epochs_since_finality) =
//    	base_reward(state, index) + get_effective_balance(state, index)
//    	* epochs_since_finality // INACTIVITY_PENALTY_QUOTIENT // 2
func inactivityPenalty(
	state *pb.BeaconState,
	validatorIndex uint32,
	baseRewardQuotient uint64,
	epochsSinceFinality uint64) uint64 {

	baseReward := baseReward(state, validatorIndex, baseRewardQuotient)
	validatorBalance := validators.EffectiveBalance(state, validatorIndex)
	return baseReward + validatorBalance*epochsSinceFinality/params.BeaconConfig().InactivityPenaltyQuotient/2
}
