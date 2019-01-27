// Package epoch contains epoch processing libraries. These libraries
// process new balance for the validators, justify and finalize new
// check points, shuffle and reassign validators to different slots and
// shards.
package epoch

import (
	"fmt"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/validators"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/mathutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

var config = params.BeaconConfig()

// CanProcessEpoch checks the eligibility to process epoch.
// The epoch can be processed every EPOCH_LENGTH.
//
// Spec pseudocode definition:
//    If state.slot % EPOCH_LENGTH == 0:
func CanProcessEpoch(state *pb.BeaconState) bool {
	return state.Slot%config.EpochLength == 0
}

// CanProcessDepositRoots checks the eligibility to process deposit root.
// The deposit root can be processed every DEPOSIT_ROOT_VOTING_PERIOD.
//
// Spec pseudocode definition:
//    If state.slot % DEPOSIT_ROOT_VOTING_PERIOD == 0:
func CanProcessDepositRoots(state *pb.BeaconState) bool {
	return state.Slot%config.Eth1DataVotingPeriod == 0
}

// CanProcessValidatorRegistry checks the eligibility to process validator registry.
// It checks crosslink committees last changed slot and finalized slot against
// latest change slot.
//
// Spec pseudocode definition:
//    If the following are satisfied:
//		* state.finalized_slot > state.validator_registry_latest_change_slot
//		* state.latest_crosslinks[shard].slot > state.validator_registry_latest_change_slot
// 			for every shard number shard in [(state.current_epoch_start_shard + i) %
//	 			SHARD_COUNT for i in range(get_current_epoch_committees_per_slot(state) *
//	 			EPOCH_LENGTH)] (that is, for every shard in the current committees)
func CanProcessValidatorRegistry(state *pb.BeaconState) bool {
	if state.FinalizedSlot <= state.ValidatorRegistryUpdateSlot {
		return false
	}
	shardsProcessed := validators.CurrCommitteesCountPerSlot(state) * config.EpochLength
	fmt.Println(shardsProcessed)
	startShard := state.CurrentEpochStartShard
	for i := startShard; i < shardsProcessed; i++ {
		if state.LatestCrosslinks[i%config.ShardCount].Slot <=
			state.ValidatorRegistryUpdateSlot {
			return false
		}
	}
	return true
}

// ProcessEth1Data processes eth1 block deposit roots by checking its vote count.
// With sufficient votes (>2*ETH1_DATA_VOTING_PERIOD), it then
// marks the voted Eth1 data as the latest data set.
func ProcessEth1Data(state *pb.BeaconState) *pb.BeaconState {
	for _, eth1DataVote := range state.Eth1DataVotes {
		if eth1DataVote.VoteCount*2 > config.Eth1DataVotingPeriod {
			state.LatestEth1Data.DepositRootHash32 = eth1DataVote.Eth1Data.DepositRootHash32
			state.LatestEth1Data.BlockHash32 = eth1DataVote.Eth1Data.BlockHash32
		}
	}
	state.Eth1DataVotes = make([]*pb.Eth1DataVote, 0)
	return state
}

// ProcessJustification processes for justified slot by comparing
// epoch boundary balance and total balance.
//
// Spec pseudocode definition:
//    Set state.previous_justified_slot = state.justified_slot.
//    Set state.justification_bitfield = (state.justification_bitfield * 2) % 2**64.
//    Set state.justification_bitfield |= 2 and state.justified_slot =
//    state.slot - 2 * EPOCH_LENGTH if 3 * previous_epoch_boundary_attesting_balance >= 2 * total_balance
//    Set state.justification_bitfield |= 1 and state.justified_slot =
//    state.slot - 1 * EPOCH_LENGTH if 3 * this_epoch_boundary_attesting_balance >= 2 * total_balance
func ProcessJustification(
	state *pb.BeaconState,
	thisEpochBoundaryAttestingBalance uint64,
	prevEpochBoundaryAttestingBalance uint64,
	totalBalance uint64) *pb.BeaconState {

	state.PreviousJustifiedSlot = state.JustifiedSlot
	// Shifts all the bits over one to create a new bit for the recent epoch.
	state.JustificationBitfield = state.JustificationBitfield * 2

	// If prev prev epoch was justified then we ensure the 2nd bit in the bitfield is set,
	// assign new justified slot to 2 * EPOCH_LENGTH before.
	if 3*prevEpochBoundaryAttestingBalance >= 2*totalBalance {
		state.JustificationBitfield |= 2
		state.JustifiedSlot = state.Slot - 2*config.EpochLength
	}

	// If this epoch was justified then we ensure the 1st bit in the bitfield is set,
	// assign new justified slot to 1 * EPOCH_LENGTH before.
	if 3*thisEpochBoundaryAttestingBalance >= 2*totalBalance {
		state.JustificationBitfield |= 1
		state.JustifiedSlot = state.Slot - 1*config.EpochLength
	}
	return state
}

// ProcessFinalization processes for finalized slot by checking
// consecutive justified slots.
//
// Spec pseudocode definition:
//   Set state.finalized_slot = state.previous_justified_slot if any of the following are true:
//		state.previous_justified_slot == state.slot - 2 * EPOCH_LENGTH and state.justification_bitfield % 4 == 3
//		state.previous_justified_slot == state.slot - 3 * EPOCH_LENGTH and state.justification_bitfield % 8 == 7
//		state.previous_justified_slot == state.slot - 4 * EPOCH_LENGTH and state.justification_bitfield % 16 in (15, 14)
func ProcessFinalization(state *pb.BeaconState) *pb.BeaconState {
	epochLength := config.EpochLength

	if state.PreviousJustifiedSlot == state.Slot-2*epochLength &&
		state.JustificationBitfield%4 == 3 {
		state.FinalizedSlot = state.JustifiedSlot
		return state
	}
	if state.PreviousJustifiedSlot == state.Slot-3*epochLength &&
		state.JustificationBitfield%8 == 7 {
		state.FinalizedSlot = state.JustifiedSlot
		return state
	}
	if state.PreviousJustifiedSlot == state.Slot-4*epochLength &&
		(state.JustificationBitfield%16 == 15 ||
			state.JustificationBitfield%16 == 14) {
		state.FinalizedSlot = state.JustifiedSlot
		return state
	}
	return state
}

// ProcessCrosslinks goes through each crosslink committee and check
// crosslink committee's attested balance * 3 is greater than total balance *2.
// If it's greater then beacon node updates crosslink committee with
// the state slot and wining root.
//
// Spec pseudocode definition:
//	For every `slot in range(state.slot - 2 * EPOCH_LENGTH, state.slot)`,
// 	let `crosslink_committees_at_slot = get_crosslink_committees_at_slot(state, slot)`.
// 		For every `(crosslink_committee, shard)` in `crosslink_committees_at_slot`, compute:
// 			Set state.latest_crosslinks[shard] = Crosslink(
// 			slot=state.slot, shard_block_root=winning_root(crosslink_committee))
// 			if 3 * total_attesting_balance(crosslink_committee) >= 2 * total_balance(crosslink_committee)
func ProcessCrosslinks(
	state *pb.BeaconState,
	thisEpochAttestations []*pb.PendingAttestationRecord,
	prevEpochAttestations []*pb.PendingAttestationRecord) (*pb.BeaconState, error) {

	var startSlot uint64
	if state.Slot > 2*config.EpochLength {
		startSlot = state.Slot - 2*config.EpochLength
	}

	for i := startSlot; i < state.Slot; i++ {
		crosslinkCommittees, err := validators.CrosslinkCommitteesAtSlot(state, i)
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
					Slot:                 state.Slot,
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
	activeValidatorIndices := validators.ActiveValidatorIndices(state.ValidatorRegistry, state.Slot)
	for _, index := range activeValidatorIndices {
		if state.ValidatorBalances[index] < config.EjectionBalance {
			state, err = validators.ExitValidator(state, index)
			if err != nil {
				return nil, fmt.Errorf("could not exit validator %d: %v", index, err)
			}
		}
	}
	return state, nil
}

// ProcessPrevSlotShard computes and sets current epoch's calculation slot
// and start shard to previous epoch. Then it returns the updated state.
//
// Spec pseudocode definition:
//	Set state.previous_epoch_randao_mix = state.current_epoch_randao_mix
//	Set state.current_epoch_calculation_slot = state.slot
func ProcessPrevSlotShard(state *pb.BeaconState) *pb.BeaconState {
	state.PreviousEpochCalculationSlot = state.CurrentEpochCalculationSlot
	state.PreviousEpochStartShard = state.CurrentEpochStartShard
	return state
}

// ProcessValidatorRegistry computes and sets new validator registry fields,
// reshuffles shard committees and returns the recomputed state with the updated registry.
//
// Spec pseudocode definition:
//	Set state.previous_epoch_randao_mix = state.current_epoch_randao_mix
//	Set state.current_epoch_calculation_slot = state.slot
//	Set state.current_epoch_start_shard = (state.current_epoch_start_shard + get_current_epoch_committees_per_slot(state) * EPOCH_LENGTH) % SHARD_COUNT
//	Set state.current_epoch_randao_mix = get_randao_mix(state, state.current_epoch_calculation_slot - SEED_LOOKAHEAD)
func ProcessValidatorRegistry(
	state *pb.BeaconState) (*pb.BeaconState, error) {
	state.PreviousEpochRandaoMixHash32 = state.CurrentEpochRandaoMixHash32
	state.CurrentEpochCalculationSlot = state.Slot

	nextStartShard := (state.CurrentEpochStartShard +
		validators.CurrCommitteesCountPerSlot(state)*config.EpochLength) %
		config.EpochLength
	state.CurrentEpochStartShard = nextStartShard

	var randaoMixSlot uint64
	if state.CurrentEpochCalculationSlot > config.SeedLookahead {
		randaoMixSlot = state.CurrentEpochCalculationSlot -
			config.SeedLookahead
	}
	mix, err := randaoMix(state, randaoMixSlot)
	if err != nil {
		return nil, fmt.Errorf("could not get randao mix: %v", err)
	}
	state.CurrentEpochRandaoMixHash32 = mix

	return state, nil
}

// ProcessPartialValidatorRegistry processes the portion of validator registry
// fields, it doesn't set registry latest change slot. This only gets called if
// validator registry update did not happen.
//
// Spec pseudocode definition:
//	Set state.previous_epoch_calculation_slot = state.current_epoch_calculation_slot
//	Set state.previous_epoch_start_shard = state.current_epoch_start_shard
//	Let epochs_since_last_registry_change = (state.slot - state.validator_registry_latest_change_slot)
// 		EPOCH_LENGTH.
//	If epochs_since_last_registry_change is an exact power of 2,
// 		set state.current_epoch_calculation_slot = state.slot
// 		set state.current_epoch_randao_mix = state.latest_randao_mixes[
// 			(state.current_epoch_calculation_slot - SEED_LOOKAHEAD) %
// 			LATEST_RANDAO_MIXES_LENGTH].
func ProcessPartialValidatorRegistry(state *pb.BeaconState) *pb.BeaconState {
	epochsSinceLastRegistryChange := (state.Slot - state.ValidatorRegistryUpdateSlot) /
		config.EpochLength

	if mathutil.IsPowerOf2(epochsSinceLastRegistryChange) {
		state.CurrentEpochCalculationSlot = state.Slot

		var randaoIndex uint64
		if state.CurrentEpochCalculationSlot > config.SeedLookahead {
			randaoIndex = state.CurrentEpochCalculationSlot - config.SeedLookahead
		}

		randaoMix := state.LatestRandaoMixesHash32S[randaoIndex%config.LatestRandaoMixesLength]
		state.CurrentEpochRandaoMixHash32 = randaoMix
	}
	return state
}

// CleanupAttestations removes any attestation in state's latest attestations
// such that the attestation slot is lower than state slot minus epoch length.
// Spec pseudocode definition:
// 		Remove any attestation in state.latest_attestations such
// 		that attestation.data.slot < state.slot - EPOCH_LENGTH
func CleanupAttestations(state *pb.BeaconState) *pb.BeaconState {
	epochLength := config.EpochLength
	var earliestSlot uint64

	// If the state slot is less than epochLength, then the earliestSlot would
	// result in a negative number. Therefore we should default to
	// earliestSlot = 0 in this case.
	if state.Slot > epochLength {
		earliestSlot = state.Slot - epochLength
	}

	var latestAttestations []*pb.PendingAttestationRecord
	for _, attestation := range state.LatestAttestations {
		if attestation.Data.Slot >= earliestSlot {
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
	epoch := state.Slot / config.EpochLength
	nextPenalizedEpoch := (epoch + 1) % config.LatestPenalizedExitLength
	currPenalizedEpoch := (epoch) % config.LatestPenalizedExitLength
	state.LatestPenalizedBalances[nextPenalizedEpoch] =
		state.LatestPenalizedBalances[currPenalizedEpoch]
	return state
}
