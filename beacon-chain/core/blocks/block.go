package blocks

import (
	"fmt"
	"time"

	ptypes "github.com/gogo/protobuf/types"
	"github.com/prysmaticlabs/prysm/beacon-chain/utils"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

var clock utils.Clock = &utils.RealClock{}

// NewGenesisBlock returns the canonical, genesis block for the beacon chain protocol.
func NewGenesisBlock(stateRoot []byte) *pb.BeaconBlock {
	block := &pb.BeaconBlock{
		ParentRootHash32:              []byte{0},
		RandaoRevealHash32:            []byte{0},
		CandidatePowReceiptRootHash32: []byte{0},
		StateRootHash32:               []byte{0},
	}
	// Genesis time here is static so error can be safely ignored.
	// #nosec G104
	protoGenesis, _ := ptypes.TimestampProto(params.BeaconConfig().GenesisTime)
	block.Timestamp = protoGenesis
	block.StateRootHash32 = stateRoot
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
