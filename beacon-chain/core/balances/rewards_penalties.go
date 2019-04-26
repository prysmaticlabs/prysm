// Package balances contains libraries to calculate reward and
// penalty quotients. It computes new validator balances
// for justifications, crosslinks and attestation inclusions. It
// also computes penalties for the inactive validators.
package balances

import (
	"errors"
	"fmt"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/epoch"
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
		state.ValidatorBalances[index] +=
			helpers.BaseReward(state, index, baseRewardQuotient) *
				justifiedAttestingBalance /
				totalBalance
	}
	activeValidatorIndices := helpers.ActiveValidatorIndices(state.ValidatorRegistry, helpers.CurrentEpoch(state))
	didNotAttestIndices := sliceutil.NotUint64(justifiedAttesterIndices, activeValidatorIndices)

	for _, index := range didNotAttestIndices {
		state.ValidatorBalances[index] -=
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
		state.ValidatorBalances[index] +=
			helpers.BaseReward(state, index, baseRewardQuotient) *
				boundaryAttestingBalance /
				totalBalance
	}
	activeValidatorIndices := helpers.ActiveValidatorIndices(state.ValidatorRegistry, helpers.CurrentEpoch(state))
	didNotAttestIndices := sliceutil.NotUint64(boundaryAttesterIndices, activeValidatorIndices)

	for _, index := range didNotAttestIndices {
		state.ValidatorBalances[index] -=
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
		state.ValidatorBalances[index] +=
			helpers.BaseReward(state, index, baseRewardQuotient) *
				headAttestingBalance /
				totalBalance
	}
	activeValidatorIndices := helpers.ActiveValidatorIndices(state.ValidatorRegistry, helpers.CurrentEpoch(state))
	didNotAttestIndices := sliceutil.NotUint64(headAttesterIndices, activeValidatorIndices)

	for _, index := range didNotAttestIndices {
		state.ValidatorBalances[index] -=
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
		state.ValidatorBalances[index] +=
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
	activeValidatorIndices := helpers.ActiveValidatorIndices(state.ValidatorRegistry, helpers.CurrentEpoch(state))
	didNotAttestIndices := sliceutil.NotUint64(justifiedAttesterIndices, activeValidatorIndices)

	for _, index := range didNotAttestIndices {
		state.ValidatorBalances[index] -=
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
	activeValidatorIndices := helpers.ActiveValidatorIndices(state.ValidatorRegistry, helpers.CurrentEpoch(state))
	didNotAttestIndices := sliceutil.NotUint64(boundaryAttesterIndices, activeValidatorIndices)

	for _, index := range didNotAttestIndices {
		state.ValidatorBalances[index] -=
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
	activeValidatorIndices := helpers.ActiveValidatorIndices(state.ValidatorRegistry, helpers.CurrentEpoch(state))
	didNotAttestIndices := sliceutil.NotUint64(headAttesterIndices, activeValidatorIndices)

	for _, index := range didNotAttestIndices {
		state.ValidatorBalances[index] -=
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
	activeValidatorIndices := helpers.ActiveValidatorIndices(state.ValidatorRegistry, currentEpoch)

	for _, index := range activeValidatorIndices {
		if state.ValidatorRegistry[index].SlashedEpoch <= currentEpoch {
			state.ValidatorBalances[index] -=
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
		proposerIndex, err := helpers.BeaconProposerIndex(state, slot)
		if err != nil {
			return nil, fmt.Errorf("could not get proposer index: %v", err)
		}
		state.ValidatorBalances[proposerIndex] +=
			helpers.BaseReward(state, proposerIndex, baseRewardQuotient) /
				params.BeaconConfig().AttestationInclusionRewardQuotient
	}
	return state, nil
}

// Crosslinks awards or slashs attesters
// for attesting shard cross links.
//
// Spec pseudocode definition:
// 	For slot in range(get_epoch_start_slot(previous_epoch), get_epoch_start_slot(current_epoch)),
// 		let crosslink_committees_at_slot = get_crosslink_committees_at_slot(slot).
// 		For every (crosslink_committee, shard) in crosslink_committee_at_slot,
// 		and every index in crosslink_committee:
//			If index in attesting_validators(crosslink_committee),
//			state.validator_balances[index] += base_reward(state, index) *
//			total_attesting_balance(crosslink_committee) //
//			get_total_balance(state, crosslink_committee)).
//			If index not in attesting_validators(crosslink_committee),
//			state.validator_balances[index] -= base_reward(state, index).
func Crosslinks(
	state *pb.BeaconState,
	thisEpochAttestations []*pb.PendingAttestation,
	prevEpochAttestations []*pb.PendingAttestation) (*pb.BeaconState, error) {

	prevEpoch := helpers.PrevEpoch(state)
	currentEpoch := helpers.CurrentEpoch(state)
	startSlot := helpers.StartSlot(prevEpoch)
	endSlot := helpers.StartSlot(currentEpoch)

	for i := startSlot; i < endSlot; i++ {
		// RegistryChange is a no-op when requesting slot in current and previous epoch.
		// Process crosslinks rewards will never request crosslink committees of next epoch.
		crosslinkCommittees, err := helpers.CrosslinkCommitteesAtSlot(state, i, false /* registryChange */)
		if err != nil {
			return nil, fmt.Errorf("could not get shard committees for slot %d: %v",
				i-params.BeaconConfig().GenesisSlot, err)
		}
		for _, crosslinkCommittee := range crosslinkCommittees {
			shard := crosslinkCommittee.Shard
			committee := crosslinkCommittee.Committee
			totalAttestingBalance, err :=
				epoch.TotalAttestingBalance(state, shard, thisEpochAttestations, prevEpochAttestations)
			if err != nil {
				return nil,
					fmt.Errorf("could not get attesting balance for shard committee %d: %v", shard, err)
			}
			totalBalance := epoch.TotalBalance(state, committee)
			baseRewardQuotient := helpers.BaseRewardQuotient(totalBalance)

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
				baseReward := helpers.BaseReward(state, index, baseRewardQuotient)
				if sliceutil.IsInUint64(index, attestingIndices) {
					state.ValidatorBalances[index] +=
						baseReward * totalAttestingBalance / totalBalance
				} else {
					state.ValidatorBalances[index] -= baseReward
				}
			}
		}
	}
	return state, nil
}
