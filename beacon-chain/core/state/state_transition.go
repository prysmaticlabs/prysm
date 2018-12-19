package state

import (
	"fmt"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/randao"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/types"
	v "github.com/prysmaticlabs/prysm/beacon-chain/core/validators"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// ExecuteStateTransition defines the procedure for a state transition function.
// Spec:
//  We now define the state transition function. At a high level the state transition is made up of two parts:
//  - The per-slot transitions, which happens every slot, and only affects a parts of the state.
//  - The per-epoch transitions, which happens at every epoch boundary (i.e. state.slot % EPOCH_LENGTH == 0), and affects the entire state.
//  The per-slot transitions generally focus on verifying aggregate signatures and saving temporary records relating to the per-slot
//  activity in the BeaconState. The per-epoch transitions focus on the validator registry, including adjusting balances and activating
//  and exiting validators, as well as processing crosslinks and managing block justification/finalization.
func ExecuteStateTransition(
	beaconState *types.BeaconState,
	block *types.Block) (*types.BeaconState, error) {

	var err error

	newState := beaconState.CopyState()

	currentSlot := newState.Slot()
	newState.SetSlot(currentSlot + 1)

	newState, err = randao.UpdateRandaoLayers(newState, newState.Slot())
	if err != nil {
		return nil, fmt.Errorf("unable to update randao layer %v", err)
	}

	newhashes, err := newState.CalculateNewBlockHashes(block, currentSlot)
	if err != nil {
		return nil, fmt.Errorf("unable to calculate recent blockhashes")
	}

	newState.SetLatestBlockHashes(newhashes)

	if block != nil {
		newState = ProcessBlock(newState, block)

		if newState.Slot()%params.BeaconConfig().EpochLength == 0 {
			newState = NewEpochTransition(newState)
		}

	}

	return newState, nil
}

// ProcessBlock describes the per block operations that happen on every slot.
func ProcessBlock(state *types.BeaconState, block *types.Block) *types.BeaconState {
	_ = block
	// TODO(#1073): This function will encompass all the per block slot transition functions, this will
	// contain checks for randao,proposer validity and block operations.
	return state
}

// NewEpochTransition describes the per epoch operations that are performed on the
// beacon state.
func NewEpochTransition(state *types.BeaconState) *types.BeaconState {
	// TODO(#1074): This will encompass all the related logic to epoch transitions.
	return state
}

// validatorSetRecalculation recomputes the validator set.
func validatorSetRecalculations(
	shardAndCommittesForSlots []*pb.ShardAndCommitteeArray,
	validators []*pb.ValidatorRecord,
	seed [32]byte,
) ([]*pb.ShardAndCommitteeArray, error) {
	lastSlot := len(shardAndCommittesForSlots) - 1
	lastCommitteeFromLastSlot := len(shardAndCommittesForSlots[lastSlot].ArrayShardAndCommittee) - 1
	crosslinkLastShard := shardAndCommittesForSlots[lastSlot].ArrayShardAndCommittee[lastCommitteeFromLastSlot].Shard
	crosslinkNextShard := (crosslinkLastShard + 1) % params.BeaconConfig().ShardCount

	newShardCommitteeArray, err := v.ShuffleValidatorRegistryToCommittees(
		seed,
		validators,
		crosslinkNextShard,
	)
	if err != nil {
		return nil, err
	}

	return append(shardAndCommittesForSlots[params.BeaconConfig().CycleLength:], newShardCommitteeArray...), nil
}

// createRandaoMix sets the block randao seed into a beacon state randao. This function
// XOR's the current state randao with the block's randao value added by the
// proposer.
func createRandaoMix(blockRandao [32]byte, beaconStateRandao [32]byte) [32]byte {
	for i, b := range blockRandao {
		beaconStateRandao[i] ^= b
	}
	return beaconStateRandao
}
