// Package blocks contains block processing libraries. These libraries
// process and verify block specific messages such as PoW receipt root,
// RANDAO, validator deposits, exits and slashing proofs.
package blocks

import (
	"fmt"

	"github.com/prysmaticlabs/prysm/beacon-chain/utils"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

var clock utils.Clock = &utils.RealClock{}

// NewGenesisBlock returns the canonical, genesis block for the beacon chain protocol.
func NewGenesisBlock(stateRoot []byte) *pb.BeaconBlock {
	block := &pb.BeaconBlock{
		Slot:             params.BeaconConfig().GenesisSlot,
		ParentRootHash32: params.BeaconConfig().ZeroHash[:],
		StateRootHash32:  stateRoot,
		RandaoReveal:     params.BeaconConfig().ZeroHash[:],
		Signature:        params.BeaconConfig().EmptySignature[:],
		Eth1Data: &pb.Eth1Data{
			DepositRootHash32: params.BeaconConfig().ZeroHash[:],
			BlockHash32:       params.BeaconConfig().ZeroHash[:],
		},
		Body: &pb.BeaconBlockBody{
			ProposerSlashings: []*pb.ProposerSlashing{},
			AttesterSlashings: []*pb.AttesterSlashing{},
			Attestations:      []*pb.Attestation{},
			Deposits:          []*pb.Deposit{},
			VoluntaryExits:    []*pb.VoluntaryExit{},
		},
	}
	return block
}

// BlockRoot returns the block root stored in the BeaconState for a given slot.
// It returns an error if the requested block root is not within the BeaconState.
// Spec pseudocode definition:
// 	def get_block_root(state: BeaconState, slot: int) -> Hash32:
//		"""
//		returns the block root at a recent ``slot``.
//		"""
//		assert state.slot <= slot + LATEST_BLOCK_ROOTS_LENGTH
//		assert slot < state.slot
//		return state.latest_block_roots[slot % LATEST_BLOCK_ROOTS_LENGTH]
func BlockRoot(state *pb.BeaconState, slot uint64) ([]byte, error) {
	earliestSlot := state.Slot - params.BeaconConfig().LatestBlockRootsLength

	if slot < earliestSlot || slot >= state.Slot {
		if earliestSlot < params.BeaconConfig().GenesisSlot {
			earliestSlot = params.BeaconConfig().GenesisSlot
		}
		return []byte{}, fmt.Errorf("slot %d is not within expected range of %d to %d",
			slot-params.BeaconConfig().GenesisSlot,
			earliestSlot-params.BeaconConfig().GenesisSlot,
			state.Slot-params.BeaconConfig().GenesisSlot,
		)
	}

	return state.LatestBlockRootHash32S[slot%params.BeaconConfig().LatestBlockRootsLength], nil
}

// ProcessBlockRoots processes the previous block root into the state, by appending it
// to the most recent block roots.
// Spec:
//  Let previous_block_root be the tree_hash_root of the previous beacon block processed in the chain.
//	Set state.latest_block_roots[(state.slot - 1) % LATEST_BLOCK_ROOTS_LENGTH] = previous_block_root.
//	If state.slot % LATEST_BLOCK_ROOTS_LENGTH == 0 append merkle_root(state.latest_block_roots) to state.batched_block_roots.
func ProcessBlockRoots(state *pb.BeaconState, parentRoot [32]byte) *pb.BeaconState {
	state.LatestBlockRootHash32S[(state.Slot-1)%params.BeaconConfig().LatestBlockRootsLength] = parentRoot[:]
	if state.Slot%params.BeaconConfig().LatestBlockRootsLength == 0 {
		merkleRoot := hashutil.MerkleRoot(state.LatestBlockRootHash32S)
		state.BatchedBlockRootHash32S = append(state.BatchedBlockRootHash32S, merkleRoot)
	}
	return state
}
