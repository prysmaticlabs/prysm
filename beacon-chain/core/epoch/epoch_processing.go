// Package epoch contains epoch processing libraries. These libraries
// process new balance for the validators, justify and finalize new
// check points, shuffle and reassign validators to different slots and
// shards.
package epoch

import (
	"fmt"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/validators"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/mathutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// CanProcessEpoch checks the eligibility to process epoch.
// The epoch can be processed every EPOCH_LENGTH.
//
// Spec pseudocode definition:
//    If state.slot % EPOCH_LENGTH == 0:
func CanProcessEpoch(state *pb.BeaconState) bool {
	return state.Slot%params.BeaconConfig().EpochLength == 0
}

// CanProcessEth1Data checks the eligibility to process the eth1 data.
// The eth1 data can be processed every ETH1_DATA_VOTING_PERIOD.
//
// Spec pseudocode definition:
//    If state.slot % ETH1_DATA_VOTING_PERIOD == 0:
func CanProcessEth1Data(state *pb.BeaconState) bool {
	return state.Slot%params.BeaconConfig().Eth1DataVotingPeriod == 0
}

// CanProcessValidatorRegistry checks the eligibility to process validator registry.
// It checks crosslink committees last changed slot and finalized slot against
// latest change slot.
//
// Spec pseudocode definition:
//    If the following are satisfied:
//		* state.finalized_epoch > state.validator_registry_latest_change_epoch
//		* state.latest_crosslinks[shard].epoch > state.validator_registry_update_epoch
// 			for every shard number shard in [(state.current_epoch_start_shard + i) %
//	 			SHARD_COUNT for i in range(get_current_epoch_committee_count(state) *
//	 			EPOCH_LENGTH)] (that is, for every shard in the current committees)
func CanProcessValidatorRegistry(state *pb.BeaconState) bool {
	if state.FinalizedEpoch <= state.ValidatorRegistryUpdateEpoch {
		return false
	}
	shardsProcessed := helpers.CurrentEpochCommitteeCount(state) * params.BeaconConfig().EpochLength
	startShard := state.CurrentEpochStartShard
	for i := startShard; i < shardsProcessed; i++ {

		if state.LatestCrosslinks[i%params.BeaconConfig().ShardCount].Epoch <=
			state.ValidatorRegistryUpdateEpoch {
			return false
		}
	}
	return true
}

// ProcessEth1Data processes eth1 block deposit roots by checking its vote count.
// With sufficient votes (>2*ETH1_DATA_VOTING_PERIOD), it then
// marks the voted Eth1 data as the latest data set.
//
// Official spec definition:
//   if state.slot % ETH1_DATA_VOTING_PERIOD == 0:
//     Set state.latest_eth1_data = eth1_data_vote.data
//     if eth1_data_vote.vote_count * 2 > ETH1_DATA_VOTING_PERIOD for
//       some eth1_data_vote in state.eth1_data_votes.
//       Set state.eth1_data_votes = [].
//
func ProcessEth1Data(state *pb.BeaconState) *pb.BeaconState {
	if state.Slot%params.BeaconConfig().Eth1DataVotingPeriod == 0 {
		for _, eth1DataVote := range state.Eth1DataVotes {
			if eth1DataVote.VoteCount*2 > params.BeaconConfig().Eth1DataVotingPeriod {
				state.LatestEth1Data.DepositRootHash32 = eth1DataVote.Eth1Data.DepositRootHash32
				state.LatestEth1Data.BlockHash32 = eth1DataVote.Eth1Data.BlockHash32
			}
		}
		state.Eth1DataVotes = make([]*pb.Eth1DataVote, 0)
	}
	return state
}

// ProcessJustification processes for justified slot by comparing
// epoch boundary balance and total balance.
//
// Spec pseudocode definition:
//    Set state.previous_justified_epoch = state.justified_epoch.
//    Set state.justification_bitfield = (state.justification_bitfield * 2) % 2**64.
//    Set state.justification_bitfield |= 2 and state.justified_epoch =
//    slot_to_epoch(state.slot) - 2  if 3 * previous_epoch_boundary_attesting_balance >= 2 * total_balance
//    Set state.justification_bitfield |= 1 and state.justified_epoch =
//    slot_to_epoch(state.slot) - 1 if 3 * this_epoch_boundary_attesting_balance >= 2 * total_balance
func ProcessJustification(
	state *pb.BeaconState,
	thisEpochBoundaryAttestingBalance uint64,
	prevEpochBoundaryAttestingBalance uint64,
	totalBalance uint64) *pb.BeaconState {

	state.PreviousJustifiedEpoch = state.JustifiedEpoch
	// Shifts all the bits over one to create a new bit for the recent epoch.
	state.JustificationBitfield = state.JustificationBitfield * 2

	// If prev prev epoch was justified then we ensure the 2nd bit in the bitfield is set,
	// assign new justified slot to 2 * EPOCH_LENGTH before.
	if 3*prevEpochBoundaryAttestingBalance >= 2*totalBalance {
		state.JustificationBitfield |= 2
		state.JustifiedEpoch = helpers.CurrentEpoch(state) - 2
	}

	// If this epoch was justified then we ensure the 1st bit in the bitfield is set,
	// assign new justified slot to 1 * EPOCH_LENGTH before.
	if 3*thisEpochBoundaryAttestingBalance >= 2*totalBalance {
		state.JustificationBitfield |= 1
		state.JustifiedEpoch = helpers.CurrentEpoch(state) - 1
	}
	return state
}

// ProcessFinalization processes for finalized slot by checking
// consecutive justified slots.
//
// Spec pseudocode definition:
//   Set state.finalized_epoch = state.previous_justified_epoch if any of the following are true:
//		state.previous_justified_epoch == slot_to_epoch(state.slot) - 2 and state.justification_bitfield % 4 == 3
//		state.previous_justified_epoch == slot_to_epoch(state.slot) - 3 and state.justification_bitfield % 8 == 7
//		state.previous_justified_epoch == slot_to_epoch(state.slot) - 4 and state.justification_bitfield % 16 in (15, 14)
func ProcessFinalization(state *pb.BeaconState) *pb.BeaconState {

	if state.PreviousJustifiedEpoch == helpers.CurrentEpoch(state)-2 &&
		state.JustificationBitfield%4 == 3 {
		state.FinalizedEpoch = state.JustifiedEpoch
		return state
	}
	if state.PreviousJustifiedEpoch == helpers.CurrentEpoch(state)-3 &&
		state.JustificationBitfield%8 == 7 {
		state.FinalizedEpoch = state.JustifiedEpoch
		return state
	}
	if state.PreviousJustifiedEpoch == helpers.CurrentEpoch(state)-4 &&
		(state.JustificationBitfield%16 == 15 ||
			state.JustificationBitfield%16 == 14) {
		state.FinalizedEpoch = state.JustifiedEpoch
		return state
	}
	return state
}

// ProcessCrosslinks goes through each crosslink committee and check
// crosslink committee's attested balance * 3 is greater than total balance *2.
// If it's greater then beacon node updates crosslink committee with
// the state epoch and wining root.
//
// Spec pseudocode definition:
//	For every slot in range(get_epoch_start_slot(previous_epoch), get_epoch_start_slot(next_epoch)),
// 	let `crosslink_committees_at_slot = get_crosslink_committees_at_slot(state, slot)`.
// 		For every `(crosslink_committee, shard)` in `crosslink_committees_at_slot`, compute:
// 			Set state.latest_crosslinks[shard] = Crosslink(
// 			epoch=current_epoch, shard_block_root=winning_root(crosslink_committee))
// 			if 3 * total_attesting_balance(crosslink_committee) >= 2 * total_balance(crosslink_committee)
func ProcessCrosslinks(
	state *pb.BeaconState,
	thisEpochAttestations []*pb.PendingAttestationRecord,
	prevEpochAttestations []*pb.PendingAttestationRecord) (*pb.BeaconState, error) {

	prevEpoch := helpers.PrevEpoch(state)
	currentEpoch := helpers.CurrentEpoch(state)
	nextEpoch := helpers.NextEpoch(state)
	startSlot := helpers.StartSlot(prevEpoch)
	endSlot := helpers.StartSlot(nextEpoch)

	for i := startSlot; i < endSlot; i++ {
		crosslinkCommittees, err := helpers.CrosslinkCommitteesAtSlot(state, i, false)
		if err != nil {
			return nil, fmt.Errorf("could not get committees for slot %d: %v", i, err)
		}
		for _, crosslinkCommittee := range crosslinkCommittees {
			shard := crosslinkCommittee.Shard
			committee := crosslinkCommittee.Committee
			attestingBalance, err := TotalAttestingBalance(state, shard, thisEpochAttestations, prevEpochAttestations)
			if err != nil {
				return nil, fmt.Errorf("could not get attesting balance for shard committee %d: %v", shard, err)
			}
			totalBalance := TotalBalance(state, committee)
			if attestingBalance*3 > totalBalance*2 {
				winningRoot, err := winningRoot(state, shard, thisEpochAttestations, prevEpochAttestations)
				if err != nil {
					return nil, fmt.Errorf("could not get winning root: %v", err)
				}
				state.LatestCrosslinks[shard] = &pb.CrosslinkRecord{
					Epoch:                currentEpoch,
					ShardBlockRootHash32: winningRoot,
				}
			}
		}
	}
	return state, nil
}

// ProcessEjections iterates through every validator and find the ones below
// ejection balance and eject them.
//
// Spec pseudocode definition:
//	def process_ejections(state: BeaconState) -> None:
//    """
//    Iterate through the validator registry
//    and eject active validators with balance below ``EJECTION_BALANCE``.
//    """
//    for index in active_validator_indices(state.validator_registry):
//        if state.validator_balances[index] < EJECTION_BALANCE:
//            exit_validator(state, index)
func ProcessEjections(state *pb.BeaconState) (*pb.BeaconState, error) {
	var err error
	activeValidatorIndices := helpers.ActiveValidatorIndices(state.ValidatorRegistry, state.Slot)
	for _, index := range activeValidatorIndices {
		if state.ValidatorBalances[index] < params.BeaconConfig().EjectionBalance {
			state, err = validators.ExitValidator(state, index)
			if err != nil {
				return nil, fmt.Errorf("could not exit validator %d: %v", index, err)
			}
		}
	}
	return state, nil
}

// ProcessPrevSlotShardSeed computes and sets current epoch's calculation slot
// and start shard to previous epoch. Then it returns the updated state.
//
// Spec pseudocode definition:
//	Set state.previous_epoch_randao_mix = state.current_epoch_randao_mix
//	Set state.previous_calculation_epoch = state.current_calculation_epoch
//  Set state.previous_epoch_seed = state.current_epoch_seed.
func ProcessPrevSlotShardSeed(state *pb.BeaconState) *pb.BeaconState {
	state.PreviousCalculationEpoch = state.CurrentCalculationEpoch
	state.PreviousEpochStartShard = state.CurrentEpochStartShard
	state.PreviousEpochSeedHash32 = state.CurrentEpochSeedHash32
	return state
}

// ProcessValidatorRegistry computes and sets new validator registry fields,
// reshuffles shard committees and returns the recomputed state with the updated registry.
//
// Spec pseudocode definition:
//  Set state.current_calculation_epoch = next_epoch
//  Set state.current_epoch_start_shard = (state.current_epoch_start_shard +
//  	get_current_epoch_committee_count(state)) % SHARD_COUNT
//	Set state.current_epoch_seed = generate_seed(state, state.current_calculation_epoch)
func ProcessValidatorRegistry(
	state *pb.BeaconState) (*pb.BeaconState, error) {
	state.CurrentCalculationEpoch = state.Slot

	nextStartShard := (state.CurrentEpochStartShard +
		helpers.CurrentEpochCommitteeCount(state)*params.BeaconConfig().EpochLength) %
		params.BeaconConfig().EpochLength
	state.CurrentEpochStartShard = nextStartShard

	var randaoMixSlot uint64
	if state.CurrentCalculationEpoch > params.BeaconConfig().SeedLookahead {
		randaoMixSlot = state.CurrentCalculationEpoch -
			params.BeaconConfig().SeedLookahead
	}
	mix, err := helpers.RandaoMix(state, randaoMixSlot)
	if err != nil {
		return nil, fmt.Errorf("could not get randao mix: %v", err)
	}
	state.CurrentEpochSeedHash32 = mix

	return state, nil
}

// ProcessPartialValidatorRegistry processes the portion of validator registry
// fields, it doesn't set registry latest change slot. This only gets called if
// validator registry update did not happen.
//
// Spec pseudocode definition:
//	Set state.previous_calculation_epoch = state.current_calculation_epoch
//	Set state.previous_epoch = state.current_epoch
//	Let epochs_since_last_registry_change = (state.slot - state.validator_registry_latest_change_slot)
// 		EPOCH_LENGTH.
//	If epochs_since_last_registry_change is an exact power of 2,
// 		set state.current_calculation_epoch = state.slot
// 		set state.current_epoch_randao_mix = state.latest_randao_mixes[
// 			(state.current_epoch_calculation_slot - SEED_LOOKAHEAD) %
// 			LATEST_RANDAO_MIXES_LENGTH].
func ProcessPartialValidatorRegistry(state *pb.BeaconState) *pb.BeaconState {
	epochsSinceLastRegistryChange := (state.Slot - state.ValidatorRegistryUpdateEpoch) /
		params.BeaconConfig().EpochLength

	if mathutil.IsPowerOf2(epochsSinceLastRegistryChange) {
		state.CurrentCalculationEpoch = state.Slot

		var randaoIndex uint64
		if state.CurrentCalculationEpoch > params.BeaconConfig().SeedLookahead {
			randaoIndex = state.CurrentCalculationEpoch - params.BeaconConfig().SeedLookahead
		}

		randaoMix := state.LatestRandaoMixesHash32S[randaoIndex%params.BeaconConfig().LatestRandaoMixesLength]
		state.CurrentEpochSeedHash32 = randaoMix
	}
	return state
}

// CleanupAttestations removes any attestation in state's latest attestations
// such that the attestation slot is lower than state slot minus epoch length.
// Spec pseudocode definition:
// 		Remove any attestation in state.latest_attestations such
// 		that slot_to_epoch(att.data.slot) < slot_to_epoch(state) - 1
func CleanupAttestations(state *pb.BeaconState) *pb.BeaconState {
	currEpoch := helpers.CurrentEpoch(state)

	var latestAttestations []*pb.PendingAttestationRecord
	for _, attestation := range state.LatestAttestations {
		if helpers.SlotToEpoch(attestation.Data.Slot) >= currEpoch {
			latestAttestations = append(latestAttestations, attestation)
		}
	}
	state.LatestAttestations = latestAttestations
	return state
}

// UpdatePenalizedExitBalances ports over the current epoch's penalized exit balances
// into next epoch.
//
// Spec pseudocode definition:
// Let e = state.slot // EPOCH_LENGTH.
// Set state.latest_penalized_exit_balances[(e+1) % LATEST_PENALIZED_EXIT_LENGTH] =
// 		state.latest_penalized_exit_balances[e % LATEST_PENALIZED_EXIT_LENGTH]
func UpdatePenalizedExitBalances(state *pb.BeaconState) *pb.BeaconState {
	epoch := state.Slot / params.BeaconConfig().EpochLength
	nextPenalizedEpoch := (epoch + 1) % params.BeaconConfig().LatestPenalizedExitLength
	currPenalizedEpoch := (epoch) % params.BeaconConfig().LatestPenalizedExitLength
	state.LatestPenalizedBalances[nextPenalizedEpoch] =
		state.LatestPenalizedBalances[currPenalizedEpoch]
	return state
}
