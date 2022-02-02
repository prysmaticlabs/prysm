package blocks

import (
	"bytes"
	"context"
	"fmt"

	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/config/params"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/block"
)

// ProcessBlockHeader validates a block by its header.
//
// Spec pseudocode definition:
//
//  def process_block_header(state: BeaconState, block: BeaconBlock) -> None:
//    # Verify that the slots match
//    assert block.slot == state.slot
//    # Verify that the block is newer than latest block header
//    assert block.slot > state.latest_block_header.slot
//    # Verify that proposer index is the correct index
//    assert block.proposer_index == get_beacon_proposer_index(state)
//    # Verify that the parent matches
//    assert block.parent_root == hash_tree_root(state.latest_block_header)
//    # Cache current block as the new latest block
//    state.latest_block_header = BeaconBlockHeader(
//        slot=block.slot,
//        proposer_index=block.proposer_index,
//        parent_root=block.parent_root,
//        state_root=Bytes32(),  # Overwritten in the next process_slot call
//        body_root=hash_tree_root(block.body),
//    )
//
//    # Verify proposer is not slashed
//    proposer = state.validators[block.proposer_index]
//    assert not proposer.slashed
func ProcessBlockHeader(
	ctx context.Context,
	beaconState state.BeaconState,
	block block.SignedBeaconBlock,
) (state.BeaconState, error) {
	if err := helpers.BeaconBlockIsNil(block); err != nil {
		return nil, err
	}
	bodyRoot, err := block.Block().Body().HashTreeRoot()
	if err != nil {
		return nil, err
	}
	beaconState, err = ProcessBlockHeaderNoVerify(ctx, beaconState, block.Block().Slot(), block.Block().ProposerIndex(), block.Block().ParentRoot(), bodyRoot[:])
	if err != nil {
		return nil, err
	}

	// Verify proposer signature.
	if err := VerifyBlockSignature(beaconState, block.Block().ProposerIndex(), block.Signature(), block.Block().HashTreeRoot); err != nil {
		return nil, err
	}

	return beaconState, nil
}

// ProcessBlockHeaderNoVerify validates a block by its header but skips proposer
// signature verification.
//
// WARNING: This method does not verify proposer signature. This is used for proposer to compute state root
// using a unsigned block.
//
// Spec pseudocode definition:
//  def process_block_header(state: BeaconState, block: BeaconBlock) -> None:
//    # Verify that the slots match
//    assert block.slot == state.slot
//    # Verify that the block is newer than latest block header
//    assert block.slot > state.latest_block_header.slot
//    # Verify that proposer index is the correct index
//    assert block.proposer_index == get_beacon_proposer_index(state)
//    # Verify that the parent matches
//    assert block.parent_root == hash_tree_root(state.latest_block_header)
//    # Cache current block as the new latest block
//    state.latest_block_header = BeaconBlockHeader(
//        slot=block.slot,
//        proposer_index=block.proposer_index,
//        parent_root=block.parent_root,
//        state_root=Bytes32(),  # Overwritten in the next process_slot call
//        body_root=hash_tree_root(block.body),
//    )
//
//    # Verify proposer is not slashed
//    proposer = state.validators[block.proposer_index]
//    assert not proposer.slashed
func ProcessBlockHeaderNoVerify(
	ctx context.Context,
	beaconState state.BeaconState,
	slot types.Slot, proposerIndex types.ValidatorIndex,
	parentRoot, bodyRoot []byte,
) (state.BeaconState, error) {
	if beaconState.Slot() != slot {
		return nil, fmt.Errorf("state slot: %d is different than block slot: %d", beaconState.Slot(), slot)
	}
	idx, err := helpers.BeaconProposerIndex(ctx, beaconState)
	if err != nil {
		return nil, err
	}
	if proposerIndex != idx {
		return nil, fmt.Errorf("proposer index: %d is different than calculated: %d", proposerIndex, idx)
	}
	parentHeader := beaconState.LatestBlockHeader()
	if parentHeader.Slot >= slot {
		return nil, fmt.Errorf("block.Slot %d must be greater than state.LatestBlockHeader.Slot %d", slot, parentHeader.Slot)
	}
	parentHeaderRoot, err := parentHeader.HashTreeRoot()
	if err != nil {
		return nil, err
	}

	if !bytes.Equal(parentRoot, parentHeaderRoot[:]) {
		return nil, fmt.Errorf(
			"parent root %#x does not match the latest block header signing root in state %#x",
			parentRoot, parentHeaderRoot[:])
	}

	proposer, err := beaconState.ValidatorAtIndexReadOnly(idx)
	if err != nil {
		return nil, err
	}
	if proposer.Slashed() {
		return nil, fmt.Errorf("proposer at index %d was previously slashed", idx)
	}

	if err := beaconState.SetLatestBlockHeader(&ethpb.BeaconBlockHeader{
		Slot:          slot,
		ProposerIndex: proposerIndex,
		ParentRoot:    parentRoot,
		StateRoot:     params.BeaconConfig().ZeroHash[:],
		BodyRoot:      bodyRoot,
	}); err != nil {
		return nil, err
	}
	return beaconState, nil
}
