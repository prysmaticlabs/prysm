package state

import (
	"fmt"

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

	currentSlot := beaconState.Slot
	beaconState.Slot = currentSlot + 1

	beaconState, err = randao.UpdateRandaoLayers(beaconState, beaconState.Slot)
	if err != nil {
		return nil, fmt.Errorf("unable to update randao layer %v", err)
	}
	beaconState = randao.UpdateRandaoMixes(beaconState)
	beaconState = b.ProcessBlockRoots(beaconState, prevBlockRoot)
	if block != nil {
		beaconState, err = ProcessBlock(beaconState, block)
		if err != nil {
			return nil, fmt.Errorf("unable to process block: %v", err)
		}
		if beaconState.Slot%params.BeaconConfig().EpochLength == 0 {
			beaconState = NewEpochTransition(beaconState)
		}
	}

	return beaconState, nil
}

// ProcessBlock creates a new, modified beacon state by applying block operation
// transformations as defined in the Ethereum Serenity specification, including processing proposer slashings,
// processing block attestations, and more.
func ProcessBlock(state *pb.BeaconState, block *pb.BeaconBlock) (*pb.BeaconState, error) {
	if block.Slot != state.Slot {
		return nil, fmt.Errorf(
			"block.slot != state.slot, block.slot = %d, state.slot = %d",
			block.GetSlot(),
			state.GetSlot(),
		)
	}
	// TODO(#781): Verify Proposer Signature.
	var err error
	state = b.ProcessPOWReceiptRoots(state, block)
	state, err = b.ProcessBlockRandao(state, block)
	if err != nil {
		return nil, fmt.Errorf("could not verify and process block randao: %v", err)
	}
	state, err = b.ProcessProposerSlashings(state, block)
	if err != nil {
		return nil, fmt.Errorf("could not verify block proposer slashings: %v", err)
	}
	state, err = b.ProcessCasperSlashings(state, block)
	if err != nil {
		return nil, fmt.Errorf("could not verify block casper slashings: %v", err)
	}
	state, err = b.ProcessBlockAttestations(state, block)
	if err != nil {
		return nil, fmt.Errorf("could not process block attestations: %v", err)
	}
	// TODO(#781): Process block validator deposits.
	state, err = b.ProcessValidatorExits(state, block)
	if err != nil {
		return nil, fmt.Errorf("could not process validator exits: %v", err)
	}
	return state, nil
}

// NewEpochTransition describes the per epoch operations that are performed on the
// beacon state.
func NewEpochTransition(state *pb.BeaconState) *pb.BeaconState {
	// TODO(#1074): This will encompass all the related logic to epoch transitions.
	return state
}
