package state

import (
	"fmt"

	"github.com/golang/protobuf/proto"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/randao"
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
	beaconState *pb.BeaconState,
	block *pb.BeaconBlock,
) (*pb.BeaconState, error) {

	var err error

	newState := proto.Clone(beaconState).(*pb.BeaconState)

	currentSlot := newState.GetSlot()
	newState.Slot = currentSlot + 1

	newState, err = randao.UpdateRandaoLayers(newState, newState.GetSlot())
	if err != nil {
		return nil, fmt.Errorf("unable to update randao layer %v", err)
	}

	newHashes, err := CalculateNewBlockHashes(newState, block, currentSlot)
	if err != nil {
		return nil, fmt.Errorf("unable to calculate recent blockhashes")
	}

	newState.LatestBlockRootHash32S = newHashes

	if block != nil {
		newState = ProcessBlock(newState, block)
		if newState.GetSlot()%params.BeaconConfig().EpochLength == 0 {
			newState = NewEpochTransition(newState)
		}
	}

	return newState, nil
}

// ProcessBlock describes the per block operations that happen on every slot.
func ProcessBlock(state *pb.BeaconState, block *pb.BeaconBlock) *pb.BeaconState {
	// TODO(#1073): This function will encompass all the per block slot transition functions, this will
	// contain checks for randao,proposer validity and block operations.
	newState := proto.Clone(state).(*pb.BeaconState)
	fmt.Printf("%v %v", newState, block)
	return state
}

// NewEpochTransition describes the per epoch operations that are performed on the
// beacon state.
func NewEpochTransition(state *pb.BeaconState) *pb.BeaconState {
	// TODO(#1074): This will encompass all the related logic to epoch transitions.
	return state
}
