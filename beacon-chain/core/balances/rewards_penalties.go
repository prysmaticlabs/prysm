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

var config = params.BeaconConfig()

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

	allValidatorIndices := validators.ActiveValidatorIndices(state.ValidatorRegistry, state.Slot)
	didNotAttestIndices := slices.Not(justifiedAttesterIndices, allValidatorIndices)

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

	allValidatorIndices := validators.ActiveValidatorIndices(state.ValidatorRegistry, state.Slot)
	didNotAttestIndices := slices.Not(boundaryAttesterIndices, allValidatorIndices)

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

	allValidatorIndices := validators.ActiveValidatorIndices(state.ValidatorRegistry, state.Slot)
	didNotAttestIndices := slices.Not(headAttesterIndices, allValidatorIndices)

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
				config.MinAttestationInclusionDelay /
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
	allValidatorIndices := validators.ActiveValidatorIndices(state.ValidatorRegistry, state.Slot)
	didNotAttestIndices := slices.Not(justifiedAttesterIndices, allValidatorIndices)

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
	allValidatorIndices := validators.ActiveValidatorIndices(state.ValidatorRegistry, state.Slot)
	didNotAttestIndices := slices.Not(boundaryAttesterIndices, allValidatorIndices)

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
	allValidatorIndices := validators.ActiveValidatorIndices(state.ValidatorRegistry, state.Slot)
	didNotAttestIndices := slices.Not(headAttesterIndices, allValidatorIndices)

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
			baseReward*config.MinAttestationInclusionDelay/
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
		proposerIndex, err := validators.BeaconProposerIndex(state, slot)
		if err != nil {
			return nil, fmt.Errorf("could not get propoer index: %v", err)
		}
		state.ValidatorBalances[proposerIndex] +=
			baseReward(state, proposerIndex, baseRewardQuotient) /
				config.IncluderRewardQuotient
	}
	return state, nil
}

// Crosslinks awards or penalizes attesters
// for attesting shard cross links.
//
// Spec pseudocode definition:
// 	For every slot in range(state.slot - 2 * EPOCH_LENGTH, state.slot),
// 		let shard_committee_at_slot = get_shard_committees_at_slot(slot).
// 		For every (shard_committee, shard) in shard_committee_at_slot, compute:
//
//			Let shard_block_root be state.latest_crosslinks[shard].shard_block_root
//			Let attesting_validator_indices(shard_committee, shard_block_root)
// 				be the union of the validator index sets given by [get_attestation_participants(
// 				state, a.data, a.participation_bitfield) for a in current_epoch_attestations +
// 				previous_epoch_attestations if a.shard == shard and a.shard_block_root == shard_block_root].
//			Let winning_root(shard_committee)
// 				be equal to the value of shard_block_root such that sum([get_effective_balance(state, i)
// 				for i in attesting_validator_indices(shard_committee, shard_block_root)])
// 				is maximized (ties broken by favoring lower shard_block_root values).
//			Let attesting_validators(shard_committee)
// 				be equal to attesting_validator_indices(
// 				shard_committee, winning_root(shard_committee)) for convenience.
//			Let total_attesting_balance(shard_committee) =
// 				sum([get_effective_balance(state, i) for i in attesting_validators(shard_committee)]).
//			Let total_balance(shard_committee) =
// 				sum([get_effective_balance(state, i) for i in shard_committee]).
func Crosslinks(
	state *pb.BeaconState,
	thisEpochAttestations []*pb.PendingAttestationRecord,
	prevEpochAttestations []*pb.PendingAttestationRecord) (*pb.BeaconState, error) {

	epochLength := config.EpochLength
	startSlot := state.Slot - 2*epochLength
	for i := startSlot; i < state.Slot; i++ {
		shardCommittees, err := validators.ShardCommitteesAtSlot(state, i)
		if err != nil {
			return nil, fmt.Errorf("could not get shard committees for slot %d", i)
		}
		for _, shardCommittee := range shardCommittees {
			shard := shardCommittee.Shard
			committee := shardCommittee.Committee
			totalAttestingBalance, err :=
				epoch.TotalAttestingBalance(state, shard, thisEpochAttestations, prevEpochAttestations)
			if err != nil {
				return nil,
					fmt.Errorf("could not get attesting balance for shard committee %d: %v", shard, err)
			}
			totalBalance := epoch.TotalBalance(state, committee)
			baseRewardQuotient := baseRewardQuotient(totalBalance)
			attestingIndices, err := epoch.AttestingValidators(
				state,
				shard,
				thisEpochAttestations,
				prevEpochAttestations)
			if err != nil {
				return nil,
					fmt.Errorf("could not get attesting indices for shard committee %d: %v", shard, err)
			}
			for _, index := range committee {
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

	baseRewardQuotient := config.BaseRewardQuotient * mathutil.IntegerSquareRoot(
		totalBalance/config.Gwei)

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
	return baseReward + validatorBalance*epochsSinceFinality/config.InactivityPenaltyQuotient/2
}
