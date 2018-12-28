package state

import (
	"fmt"

	"github.com/gogo/protobuf/proto"
	b "github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
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
	prevBlockRoot [32]byte,
) (*pb.BeaconState, error) {

	var err error

	newState := proto.Clone(beaconState).(*pb.BeaconState)

	currentSlot := newState.GetSlot()
	newState.Slot = currentSlot + 1

	newState, err = randao.UpdateRandaoLayers(newState, newState.GetSlot())
	if err != nil {
		return nil, fmt.Errorf("unable to update randao layer %v", err)
	}
	newState = randao.UpdateRandaoMixes(newState)

	newState = b.ProcessBlockRoots(newState, prevBlockRoot)

	newHashes, err := CalculateNewBlockHashes(newState, block, currentSlot)
	if err != nil {
		return nil, fmt.Errorf("unable to calculate recent blockhashes")
	}

	newState.LatestBlockRootHash32S = newHashes

	if block != nil {
		newState, err = ProcessBlock(newState, block)
		if err != nil {
			return nil, fmt.Errorf("unable to process block: %v", err)
		}
		if newState.GetSlot()%params.BeaconConfig().EpochLength == 0 {
			newState = NewEpochTransition(newState)
		}
	}

	return newState, nil
}

// ProcessBlock creates a new, modified beacon state by applying block operation
// transformations as defined in the Ethereum Serenity specification, including processing proposer slashings,
// processing block attestations, and more.
func ProcessBlock(state *pb.BeaconState, block *pb.BeaconBlock) (*pb.BeaconState, error) {
	newState := proto.Clone(state).(*pb.BeaconState)
	if block.GetSlot() != state.GetSlot() {
		return nil, fmt.Errorf(
			"block.slot != state.slot, block.slot = %d, state.slot = %d",
			block.GetSlot(),
			newState.GetSlot(),
		)
	}
	// TODO(#781): Verify Proposer Signature.
	var err error
	newState = b.ProcessPOWReceiptRoots(newState, block)
	newState, err = b.ProcessBlockRandao(newState, block)
	if err != nil {
		return nil, fmt.Errorf("could not verify and process block randao: %v", err)
	}
	newState, err = b.ProcessProposerSlashings(newState, block)
	if err != nil {
		return nil, fmt.Errorf("could not verify block proposer slashings: %v", err)
	}
	newState, err = b.ProcessCasperSlashings(newState, block)
	if err != nil {
		return nil, fmt.Errorf("could not verify block casper slashings: %v", err)
	}
	newState, err = b.ProcessBlockAttestations(newState, block)
	if err != nil {
		return nil, fmt.Errorf("could not process block attestations: %v", err)
	}
	// TODO(#781): Process block validator deposits.
	newState, err = b.ProcessValidatorExits(newState, block)
	if err != nil {
		return nil, fmt.Errorf("could not process validator exits: %v", err)
	}
	return newState, nil
}

// NewEpochTransition describes the per epoch operations that are performed on the
// beacon state.
func NewEpochTransition(state *pb.BeaconState) *pb.BeaconState {
	// TODO(#1074): This will encompass all the related logic to epoch transitions.
	return state
}
