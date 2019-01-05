package epoch

import (
	"fmt"

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

// CanProcessReceiptRoots checks the eligibility to process PoW receipt root.
// The receipt root can be processed every POW_RECEIPT_ROOT_VOTING_PERIOD.
//
// Spec pseudocode definition:
//    If state.slot % POW_RECEIPT_ROOT_VOTING_PERIOD == 0:
func CanProcessReceiptRoots(state *pb.BeaconState) bool {
	return state.Slot%params.BeaconConfig().PowReceiptRootVotingPeriod == 0
}

// CanProcessValidatorRegistry checks the eligibility to process validator registry.
// It checks shard committees last changed slot and finalized slot against
// latest change slot.
//
// Spec pseudocode definition:
//    If the following are satisfied:
//		* state.finalized_slot > state.validator_registry_latest_change_slot
//		* state.latest_crosslinks[shard].slot > state.validator_registry_latest_change_slot
// 			for every shard number shard in state.shard_committees_at_slots
func CanProcessValidatorRegistry(state *pb.BeaconState) bool {
	if state.FinalizedSlot <= state.ValidatorRegistryLastChangeSlot {
		return false
	}
	for _, shardCommitteesAtSlot := range state.ShardAndCommitteesAtSlots {
		for _, shardCommittee := range shardCommitteesAtSlot.ArrayShardAndCommittee {
			if state.LatestCrosslinks[shardCommittee.Shard].Slot <= state.ValidatorRegistryLastChangeSlot {
				return false
			}
		}
	}
	return true
}

// ProcessReceipt processes PoW receipt roots by checking its vote count.
// With sufficient votes (>2*POW_RECEIPT_ROOT_VOTING_PERIOD), it then
// assigns root hash to processed receipt vote in state.
func ProcessReceipt(state *pb.BeaconState) *pb.BeaconState {

	for _, receiptRoot := range state.CandidatePowReceiptRoots {
		if receiptRoot.VoteCount*2 > params.BeaconConfig().PowReceiptRootVotingPeriod {
			state.ProcessedPowReceiptRootHash32 = receiptRoot.CandidatePowReceiptRootHash32
		}
	}
	state.CandidatePowReceiptRoots = make([]*pb.CandidatePoWReceiptRootRecord, 0)
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
		state.JustifiedSlot = state.Slot - 2*params.BeaconConfig().EpochLength
	}

	// If this epoch was justified then we ensure the 1st bit in the bitfield is set,
	// assign new justified slot to 1 * EPOCH_LENGTH before.
	if 3*thisEpochBoundaryAttestingBalance >= 2*totalBalance {
		state.JustificationBitfield |= 1
		state.JustifiedSlot = state.Slot - 1*params.BeaconConfig().EpochLength
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
	epochLength := params.BeaconConfig().EpochLength

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

// ProcessCrosslinks goes through each shard committee and check
// shard committee's attested balance * 3 was greater than total balance *2.
// If it's greater then beacon node updates shard committee with
// the latest state slot and wining root.
//
// Spec pseudocode definition:
//	For every shard_committee in state.shard_committees_at_slots:
// 		Set state.latest_crosslinks[shard] = CrosslinkRecord(
// 		slot=state.slot, block_root=winning_root(shard_committee))
// 		if 3 * total_attesting_balance(shard_committee) >= 2 * total_balance(shard_committee)
func ProcessCrosslinks(
	state *pb.BeaconState,
	thisEpochAttestations []*pb.PendingAttestationRecord,
	prevEpochAttestations []*pb.PendingAttestationRecord) (*pb.BeaconState, error) {

	for _, shardCommitteesAtSlot := range state.ShardAndCommitteesAtSlots {
		for _, shardCommittee := range shardCommitteesAtSlot.ArrayShardAndCommittee {
			attestingBalance, err := TotalAttestingBalance(state, shardCommittee, thisEpochAttestations, prevEpochAttestations)
			if err != nil {
				return nil, fmt.Errorf("could not get attesting balance for shard committee %d: %v", shardCommittee.Shard, err)
			}
			totalBalance := TotalBalance(state, shardCommittee)
			if attestingBalance*3 > totalBalance*2 {
				winningRoot, err := WinningRoot(state, shardCommittee, thisEpochAttestations, prevEpochAttestations)
				if err != nil {
					return nil, fmt.Errorf("could not get winning root: %v", err)
				}
				state.LatestCrosslinks[shardCommittee.Shard] = &pb.CrosslinkRecord{
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
		if state.ValidatorBalances[index] < params.BeaconConfig().EjectionBalanceInGwei {
			state, err = validators.ExitValidator(state, index)
			if err != nil {
				return nil, fmt.Errorf("could not exit validator %d: %v", index, err)
			}
		}
	}
	return state, nil
}

// ProcessValidatorRegistry computes and sets new validator registry fields,
// reshuffles shard committees and returns the recomputed state.
//
// Spec pseudocode definition:
//	Set state.shard_committees_at_slots[:EPOCH_LENGTH] = state.shard_committees_at_slots[EPOCH_LENGTH:].
//	Set state.shard_committees_at_slots[EPOCH_LENGTH:] =
//  		get_new_shuffling(state.latest_randao_mixes[(state.slot - SEED_LOOKAHEAD) %
//  		LATEST_RANDAO_MIXES_LENGTH], state.validator_registry, next_start_shard, state.slot)
//  		where next_start_shard = (state.shard_committees_at_slots[-1][-1].shard + 1) % SHARD_COUNT
func ProcessValidatorRegistry(
	state *pb.BeaconState) (*pb.BeaconState, error) {

	epochLength := int(params.BeaconConfig().EpochLength)
	randaoMixesLength := params.BeaconConfig().LatestRandaoMixesLength
	seedLookahead := params.BeaconConfig().SeedLookahead
	shardCount := params.BeaconConfig().ShardCount

	shardCommittees := state.ShardAndCommitteesAtSlots
	lastSlot := len(shardCommittees) - 1
	lastCommittee := len(shardCommittees[lastSlot].ArrayShardAndCommittee) - 1
	nextStartShard := (shardCommittees[lastSlot].ArrayShardAndCommittee[lastCommittee].Shard + 1) %
		shardCount

	var randaoHash32 [32]byte
	copy(randaoHash32[:], state.LatestRandaoMixesHash32S[(state.Slot-
		uint64(seedLookahead))%randaoMixesLength])

	for i := 0; i < epochLength; i++ {
		state.ShardAndCommitteesAtSlots[i] = state.ShardAndCommitteesAtSlots[epochLength+i]
	}
	newShuffledCommittees, err := validators.ShuffleValidatorRegistryToCommittees(
		randaoHash32,
		state.ValidatorRegistry,
		nextStartShard,
		state.Slot,
	)
	if err != nil {
		return nil, fmt.Errorf("could not shuffle validator registry for commtitees: %v", err)
	}

	for i := 0; i < epochLength; i++ {
		state.ShardAndCommitteesAtSlots[epochLength+i] = newShuffledCommittees[i]
	}
	return state, nil
}

// ProcessPartialValidatorRegistry processes the portion of validator registry
// fields, it doesn't set registry latest change slot. This only gets called if
// validator registry update did not happen.
//
// Spec pseudocode definition:
//	Set state.shard_committees_at_slots[:EPOCH_LENGTH] = state.shard_committees_at_slots[EPOCH_LENGTH:]
//  Let epochs_since_last_registry_change =
//  	(state.slot - state.validator_registry_latest_change_slot) // EPOCH_LENGTH
//	If epochs_since_last_registry_change is an exact power of 2:
// 		state.shard_committees_at_slots[EPOCH_LENGTH:] =
// 			get_shuffling(state.latest_randao_mixes[(state.slot - SEED_LOOKAHEAD)
// 			% LATEST_RANDAO_MIXES_LENGTH], state.validator_registry, start_shard, state.slot)
func ProcessPartialValidatorRegistry(
	state *pb.BeaconState) (*pb.BeaconState, error) {

	epochLength := int(params.BeaconConfig().EpochLength)
	randaoMixesLength := params.BeaconConfig().LatestRandaoMixesLength
	seedLookahead := params.BeaconConfig().SeedLookahead
	var randaoHash32 [32]byte
	copy(randaoHash32[:], state.LatestRandaoMixesHash32S[(state.Slot-uint64(seedLookahead))%randaoMixesLength])

	for i := 0; i < epochLength; i++ {
		state.ShardAndCommitteesAtSlots[i] = state.ShardAndCommitteesAtSlots[epochLength+i]
	}
	epochsSinceLastRegistryChange := (state.Slot - state.ValidatorRegistryLastChangeSlot) / uint64(epochLength)
	startShard := state.ShardAndCommitteesAtSlots[0].ArrayShardAndCommittee[0].Shard
	if mathutil.IsPowerOf2(epochsSinceLastRegistryChange) {
		newShuffledCommittees, err := validators.ShuffleValidatorRegistryToCommittees(
			randaoHash32,
			state.ValidatorRegistry,
			startShard,
			state.Slot,
		)
		if err != nil {
			return nil, fmt.Errorf("could not shuffle validator registry for commtitees: %v", err)
		}
		for i := 0; i < epochLength; i++ {
			state.ShardAndCommitteesAtSlots[epochLength+i] = newShuffledCommittees[i]
		}
	}
	return state, nil
}

// CleanupAttestations removes any attestation in state's latest attestations
// such that the attestation slot is lower than state slot minus epoch length.
// Spec pseudocode definition:
// 		Remove any attestation in state.latest_attestations such
// 		that attestation.data.slot < state.slot - EPOCH_LENGTH
func CleanupAttestations(state *pb.BeaconState) *pb.BeaconState {
	epochLength := params.BeaconConfig().EpochLength
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
