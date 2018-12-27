package blocks

import (
	"fmt"
	"time"

	"github.com/prysmaticlabs/prysm/beacon-chain/utils"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

var clock utils.Clock = &utils.RealClock{}

// NewGenesisBlock returns the canonical, genesis block for the beacon chain protocol.
func NewGenesisBlock(stateRoot []byte) *pb.BeaconBlock {
	block := &pb.BeaconBlock{
		Slot:               params.BeaconConfig().InitialSlotNumber,
		ParentRootHash32:   params.BeaconConfig().ZeroHash[:],
		StateRootHash32:    stateRoot,
		RandaoRevealHash32: params.BeaconConfig().ZeroHash[:],
		Signature:          params.BeaconConfig().EmptySignature,
		Body: &pb.BeaconBlockBody{
			ProposerSlashings: []*pb.ProposerSlashing{},
			CasperSlashings:   []*pb.CasperSlashing{},
			Attestations:      []*pb.Attestation{},
			Deposits:          []*pb.Deposit{},
			Exits:             []*pb.Exit{},
		},
	}
	return block
}

// IsRandaoValid verifies the validity of randao from block by comparing it with
// the proposer's randao from the beacon state.
func IsRandaoValid(blockRandao []byte, stateRandao []byte) bool {
	var h [32]byte
	copy(h[:], stateRandao)
	return hashutil.Hash(blockRandao) == h
}

// IsSlotValid compares the slot to the system clock to determine if the block is valid.
func IsSlotValid(slot uint64, genesisTime time.Time) bool {
	slotDuration := time.Duration(slot*params.BeaconConfig().SlotDuration) * time.Second
	validTimeThreshold := genesisTime.Add(slotDuration)
	return clock.Now().After(validTimeThreshold)
}

// BlockRoot returns the block hash from input slot, the block hashes
// are stored in BeaconState.
//
// Spec pseudocode definition:
//   def get_block_root(state: BeaconState, slot: int) -> Hash32:
//     """
//     Returns the block hash at a recent ``slot``.
//     """
//     earliest_slot_in_array = state.slot - len(state.latest_block_roots)
//     assert earliest_slot_in_array <= slot < state.slot
//     return state.latest_block_roots[slot - earliest_slot_in_array]
func BlockRoot(state *pb.BeaconState, slot uint64) ([]byte, error) {
	var earliestSlot uint64

	// If the state slot is less than the length of state block root list, then
	// the earliestSlot would result in a negative number. Therefore we should
	// default earliestSlot = 0 in this case.
	if state.Slot > uint64(len(state.LatestBlockRootHash32S)) {
		earliestSlot = state.Slot - uint64(len(state.LatestBlockRootHash32S))
	}

	if slot < earliestSlot || slot >= state.Slot {
		return []byte{}, fmt.Errorf("slot %d out of bounds: %d <= slot < %d",
			slot,
			earliestSlot,
			state.Slot,
		)
	}

	return state.LatestBlockRootHash32S[slot-earliestSlot], nil
}

// ProcessBlockRoots processes the previous block root into the state, by appending it
// to the most recent block roots.
// Spec:
//  Let previous_block_root be the tree_hash_root of the previous beacon block processed in the chain.
//	Set state.latest_block_roots[(state.slot - 1) % LATEST_BLOCK_ROOTS_LENGTH] = previous_block_root.
//	If state.slot % LATEST_BLOCK_ROOTS_LENGTH == 0 append merkle_root(state.latest_block_roots) to state.batched_block_roots.
func ProcessBlockRoots(state *pb.BeaconState, prevBlockRoot [32]byte) *pb.BeaconState {
	state.LatestBlockRootHash32S[(state.Slot-1)%params.BeaconConfig().LatestBlockRootsLength] = prevBlockRoot[:]
	if state.Slot%params.BeaconConfig().LatestBlockRootsLength == 0 {
		merkleRoot := hashutil.MerkleRoot(state.LatestBlockRootHash32S)
		state.BatchedBlockRootHash32S = append(state.BatchedBlockRootHash32S, merkleRoot)
	}

	return state
}
