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

// FFGSrcRewardsPenalties applies rewards or penalties
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
func FFGSrcRewardsPenalties(
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

	allValidatorIndices := validators.AllActiveValidatorsIndices(state)
	didNotAttestIndices := slices.Not(justifiedAttesterIndices, allValidatorIndices)

	for _, index := range didNotAttestIndices {
		state.ValidatorBalances[index] -=
			baseReward(state, index, baseRewardQuotient)
	}
	return state
}

// FFGTargetRewardsPenalties applies rewards or penalties
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
func FFGTargetRewardsPenalties(
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

	allValidatorIndices := validators.AllActiveValidatorsIndices(state)
	didNotAttestIndices := slices.Not(boundaryAttesterIndices, allValidatorIndices)

	for _, index := range didNotAttestIndices {
		state.ValidatorBalances[index] -=
			baseReward(state, index, baseRewardQuotient)
	}
	return state
}

// ChainHeadRewardsPenalties applies rewards or penalties
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
func ChainHeadRewardsPenalties(
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

	allValidatorIndices := validators.AllActiveValidatorsIndices(state)
	didNotAttestIndices := slices.Not(headAttesterIndices, allValidatorIndices)

	for _, index := range didNotAttestIndices {
		state.ValidatorBalances[index] -=
			baseReward(state, index, baseRewardQuotient)
	}
	return state
}

// InclusionDistRewards applies rewards based on
// inclusion distance. It uses calculated inclusion distance
// and base reward quotient to calculate the reward amount.
//
// Spec pseudocode definition:
//    Any validator index in previous_epoch_attester_indices gains
//    base_reward(state, index) * MIN_ATTESTATION_INCLUSION_DELAY //
//    inclusion_distance(state, index)
func InclusionDistRewards(
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

// InactivityFFGSrcPenalties applies penalties to inactive
// validators that missed to vote FFG source over an
// extended of time. (epochs_since_finality > 4)
//
// Spec pseudocode definition:
//    Any active validator index not in previous_epoch_justified_attester_indices,
//    loses inactivity_penalty(state, index, epochs_since_finality)
func InactivityFFGSrcPenalties(
	state *pb.BeaconState,
	justifiedAttesterIndices []uint32,
	totalBalance uint64,
	epochsSinceFinality uint64) *pb.BeaconState {

	baseRewardQuotient := baseRewardQuotient(totalBalance)
	allValidatorIndices := validators.AllActiveValidatorsIndices(state)
	didNotAttestIndices := slices.Not(justifiedAttesterIndices, allValidatorIndices)

	for _, index := range didNotAttestIndices {
		state.ValidatorBalances[index] -=
			inactivityPenalty(state, index, baseRewardQuotient, epochsSinceFinality)
	}
	return state
}

// InactivityFFGTargetPenalties applies penalties to inactive
// validators that missed to vote FFG target over an
// extended of time. (epochs_since_finality > 4)
//
// Spec pseudocode definition:
//    Any active validator index not in previous_epoch_boundary_attester_indices,
// 	  loses inactivity_penalty(state, index, epochs_since_finality)
func InactivityFFGTargetPenalties(
	state *pb.BeaconState,
	boundaryAttesterIndices []uint32,
	totalBalance uint64,
	epochsSinceFinality uint64) *pb.BeaconState {

	baseRewardQuotient := baseRewardQuotient(totalBalance)
	allValidatorIndices := validators.AllActiveValidatorsIndices(state)
	didNotAttestIndices := slices.Not(boundaryAttesterIndices, allValidatorIndices)

	for _, index := range didNotAttestIndices {
		state.ValidatorBalances[index] -=
			inactivityPenalty(state, index, baseRewardQuotient, epochsSinceFinality)
	}
	return state
}

// InactivityHeadPenalties applies penalties to inactive validators
// that missed to vote on canonical head over an extended of time.
// (epochs_since_finality > 4)
//
// Spec pseudocode definition:
//    Any active validator index not in previous_epoch_head_attester_indices,
// 	  loses base_reward(state, index)
func InactivityHeadPenalties(
	state *pb.BeaconState,
	headAttesterIndices []uint32,
	totalBalance uint64) *pb.BeaconState {

	baseRewardQuotient := baseRewardQuotient(totalBalance)
	allValidatorIndices := validators.AllActiveValidatorsIndices(state)
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
//    Any validator index with status == EXITED_WITH_PENALTY,
//    loses 2 * inactivity_penalty(state, index, epochs_since_finality) +
//    base_reward(state, index)
func InactivityExitedPenalties(
	state *pb.BeaconState,
	totalBalance uint64,
	epochsSinceFinality uint64) *pb.BeaconState {

	baseRewardQuotient := baseRewardQuotient(totalBalance)
	allValidatorIndices := validators.AllValidatorsIndices(state)

	for _, index := range allValidatorIndices {
		if state.ValidatorRegistry[index].Status == pb.ValidatorRecord_EXITED_WITH_PENALTY {
			state.ValidatorBalances[index] -=
				2*inactivityPenalty(state, index, baseRewardQuotient, epochsSinceFinality) +
					baseReward(state, index, baseRewardQuotient)
		}
	}
	return state
}

// InactivityInclusionPenalties applies penalties in relation with
// inclusion delay to inactive validators.
//
// Spec pseudocode definition:
//    Any validator index in previous_epoch_attester_indices loses
//    base_reward(state, index) - base_reward(state, index) *
//    MIN_ATTESTATION_INCLUSION_DELAY // inclusion_distance(state, index)
func InactivityInclusionPenalties(
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

// AttestationInclusionRewards awards the the beacon
// proposers who included previous epoch attestations.
//
// Spec pseudocode definition:
//    For each index in previous_epoch_attester_indices,
//    we determine the proposer proposer_index =
//    get_beacon_proposer_index(state, inclusion_slot(state, index))
//    and set state.validator_balances[proposer_index] +=
//    base_reward(state, index) // INCLUDER_REWARD_QUOTIENT
func AttestationInclusionRewards(
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
				params.BeaconConfig().IncluderRewardQuotient
	}
	return state, nil
}

// CrosslinksRewardsPenalties awards or penalizes attesters
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
func CrosslinksRewardsPenalties(
	state *pb.BeaconState,
	totalAttestingBalance uint64,
	totalBalance uint64,
	attesterIndices []uint32) *pb.BeaconState {

	epochLength := params.BeaconConfig().EpochLength
	baseRewardQuotient := baseRewardQuotient(totalBalance)
	for _, shardCommitteesAtSlot := range state.ShardAndCommitteesAtSlots[:epochLength] {
		for _, shardCommitees := range shardCommitteesAtSlot.ArrayShardAndCommittee {

			for _, index := range shardCommitees.Committee {
				baseReward := baseReward(state, index, baseRewardQuotient)
				if slices.IsIn(index, attesterIndices) {
					state.ValidatorBalances[index] +=
						baseReward * totalAttestingBalance / totalBalance
				} else {
					state.ValidatorBalances[index] -=
						baseReward * totalAttestingBalance / totalBalance
				}
			}
		}
	}
	return state
}
