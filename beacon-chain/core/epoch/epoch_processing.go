package epoch

import (
	"github.com/gogo/protobuf/proto"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
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

// ProcessReceipt processes PoW receipt roots by checking its vote count.
// With sufficient votes (>2*POW_RECEIPT_ROOT_VOTING_PERIOD), it then
// assigns root hash to processed receipt vote in state.
func ProcessReceipt(state *pb.BeaconState) *pb.BeaconState {
	newState := proto.Clone(state).(*pb.BeaconState)
	for _, receiptRoot := range state.CandidatePowReceiptRoots {
		if receiptRoot.VoteCount*2 > params.BeaconConfig().PowReceiptRootVotingPeriod {
			newState.ProcessedPowReceiptRootHash32 = receiptRoot.CandidatePowReceiptRootHash32
		}
	}
	newState.CandidatePowReceiptRoots = make([]*pb.CandidatePoWReceiptRootRecord, 0)
	return newState
}

// ProcessJustification processes for justified slot by comparing
// epoch boundary balance and total balance.
//
// Spec pseudocode definition:
//    Set state.previous_justified_slot = state.justified_slot.
//    Set state.justification_bitfield = (state.justification_bitfield * 2) % 2**64.
//    Set state.justification_bitfield |= 2 and state.justified_slot =
//    state.slot - 2 * EPOCH_LENGTH if 3 * previous_epoch_boundary_attesting_balance >= 2 * total_balance.
//    Set state.justification_bitfield |= 1 and state.justified_slot =
//    state.slot - 1 * EPOCH_LENGTH if 3 * this_epoch_boundary_attesting_balance >= 2 * total_balance.
func ProcessJustification(
	state *pb.BeaconState,
	thisEpochBoundaryAttestingBalance uint64,
	prevEpochBoundaryAttestingBalance uint64,
	totalBalance uint64) *pb.BeaconState {

	newState := proto.Clone(state).(*pb.BeaconState)
	newState.PreviousJustifiedSlot = state.JustifiedSlot
	// Shifts all the bits over one to create a new bit for the recent epoch.
	newState.JustificationBitfield = state.JustificationBitfield * 2

	// If prev prev epoch was justified then we ensure the 2nd bit in the bitfield is set,
	// assign new justified slot to 2 * EPOCH_LENGTH before.
	if 3*prevEpochBoundaryAttestingBalance >= 2*totalBalance {
		newState.JustificationBitfield |= 2
		newState.JustifiedSlot = state.Slot - 2*params.BeaconConfig().EpochLength
	}

	// If this epoch was justified then we ensure the 1st bit in the bitfield is set,
	// assign new justified slot to 1 * EPOCH_LENGTH before.
	if 3*thisEpochBoundaryAttestingBalance >= 2*totalBalance {
		newState.JustificationBitfield |= 1
		newState.JustifiedSlot = state.Slot - 1*params.BeaconConfig().EpochLength
	}
	return newState
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

	newState := proto.Clone(state).(*pb.BeaconState)
	if state.PreviousJustifiedSlot == state.Slot-2*epochLength &&
		state.JustificationBitfield%4 == 3 {
		newState.FinalizedSlot = state.JustifiedSlot
		return newState
	}
	if state.PreviousJustifiedSlot == state.Slot-3*epochLength &&
		state.JustificationBitfield%8 == 7 {
		newState.FinalizedSlot = state.JustifiedSlot
		return newState
	}
	if state.PreviousJustifiedSlot == state.Slot-4*epochLength &&
		(state.JustificationBitfield%16 == 15 ||
			state.JustificationBitfield%16 == 14) {
		newState.FinalizedSlot = state.JustifiedSlot
		return newState
	}
	return newState
}
