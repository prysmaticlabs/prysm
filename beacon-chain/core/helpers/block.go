package helpers

import (
	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// BlockRootAtSlot returns the block root stored in the BeaconState for a recent slot.
// It returns an error if the requested block root is not within the slot range.
//
// Spec pseudocode definition:
//  def get_block_root_at_slot(state: BeaconState, slot: Slot) -> Hash:
//    """
//    Return the block root at a recent ``slot``.
//    """
//    assert slot < state.slot <= slot + SLOTS_PER_HISTORICAL_ROOT
//    return state.block_roots[slot % SLOTS_PER_HISTORICAL_ROOT]
func BlockRootAtSlot(state *stateTrie.BeaconState, slot uint64) ([]byte, error) {
	if slot >= state.Slot() || state.Slot() > slot+params.BeaconConfig().SlotsPerHistoricalRoot {
		return []byte{}, errors.Errorf("slot %d out of bounds", slot)
	}
	return state.BlockRootAtIndex(slot % params.BeaconConfig().SlotsPerHistoricalRoot)
}

// BlockRoot returns the block root stored in the BeaconState for epoch start slot.
//
// Spec pseudocode definition:
//  def get_block_root(state: BeaconState, epoch: Epoch) -> Hash:
//    """
//    Return the block root at the start of a recent ``epoch``.
//    """
//    return get_block_root_at_slot(state, compute_start_slot_of_epoch(epoch))
func BlockRoot(state *stateTrie.BeaconState, epoch uint64) ([]byte, error) {
	return BlockRootAtSlot(state, StartSlot(epoch))
}

// SignedBlockHeaderFromBlock converts a given signed beacon block into a signed beacon block header.
func SignedBlockHeaderFromBlock(block *ethpb.SignedBeaconBlock) (*ethpb.SignedBeaconBlockHeader, error) {
	bodyRoot, err := stateutil.BlockBodyRoot(block.Block.Body)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get body root of block")
	}
	return &ethpb.SignedBeaconBlockHeader{
		Header: &ethpb.BeaconBlockHeader{
			Slot:          block.Block.Slot,
			ProposerIndex: block.Block.ProposerIndex,
			ParentRoot:    block.Block.ParentRoot,
			StateRoot:     block.Block.StateRoot,
			BodyRoot:      bodyRoot[:],
		},
		Signature: block.Signature,
	}, nil
}
