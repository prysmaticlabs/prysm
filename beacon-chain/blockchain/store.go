package blockchain

import (
	"bytes"
	"context"

	"github.com/pkg/errors"
	types "github.com/prysmaticlabs/eth2-types"
	coreTime "github.com/prysmaticlabs/prysm/beacon-chain/core/time"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/block"
	"github.com/prysmaticlabs/prysm/time/slots"
)

// store is the store defend in the fork choice consensus spec:
//
// class Store(object):
//    time: uint64
//    genesis_time: uint64
//    justified_checkpoint: Checkpoint
//    finalized_checkpoint: Checkpoint
//    best_justified_checkpoint: Checkpoint
//    proposerBoostRoot: Root
type store struct {
	time                 uint64
	genesisTime          uint64
	justifiedCheckpt     *ethpb.Checkpoint
	finalizedCheckpt     *ethpb.Checkpoint
	bestJustifiedCheckpt *ethpb.Checkpoint
	proposerBoostRoot    [32]byte
}

// InitializeStore initializes the fork choice store.
//
// def get_forkchoice_store(anchor_state: BeaconState, anchor_block: BeaconBlock) -> Store:
//    assert anchor_block.state_root == hash_tree_root(anchor_state)
//    anchor_root = hash_tree_root(anchor_block)
//    anchor_epoch = get_current_epoch(anchor_state)
//    justified_checkpoint = Checkpoint(epoch=anchor_epoch, root=anchor_root)
//    finalized_checkpoint = Checkpoint(epoch=anchor_epoch, root=anchor_root)
//    proposer_boost_root = Root()
//    return Store(
//        time=uint64(anchor_state.genesis_time + SECONDS_PER_SLOT * anchor_state.slot),
//        genesis_time=anchor_state.genesis_time,
//        justified_checkpoint=justified_checkpoint,
//        finalized_checkpoint=finalized_checkpoint,
//        best_justified_checkpoint=justified_checkpoint,
//        proposer_boost_root=proposer_boost_root,
//        blocks={anchor_root: copy(anchor_block)},
//        block_states={anchor_root: copy(anchor_state)},
//        checkpoint_states={justified_checkpoint: copy(anchor_state)},
//    )
func (s *Service) InitializeStore(ctx context.Context, anchorState state.BeaconState, anchorBlock block.SignedBeaconBlock) error {
	r, err := anchorState.HashTreeRoot(ctx)
	if err != nil {
		return err
	}
	if r != bytesutil.ToBytes32(anchorBlock.Block().StateRoot()) {
		return errors.New("anchor state root does not match anchor block state root")
	}
	r, err = anchorBlock.Block().HashTreeRoot()
	if err != nil {
		return err
	}
	cp := &ethpb.Checkpoint{
		Epoch: coreTime.CurrentEpoch(anchorState),
		Root:  r[:],
	}
	s.store = &store{
		time:                 anchorState.GenesisTime() + uint64(anchorState.Slot().Mul(params.BeaconConfig().SecondsPerSlot)),
		genesisTime:          anchorState.GenesisTime(),
		justifiedCheckpt:     cp,
		finalizedCheckpt:     cp,
		bestJustifiedCheckpt: cp,
		proposerBoostRoot:    [32]byte{},
	}
	return nil
}

// OnTick updates the store with the current time.
// def on_tick(store: Store, time: uint64) -> None:
//    previous_slot = get_current_slot(store)
//
//    # update store time
//    store.time = time
//
//    current_slot = get_current_slot(store)
//
//    # Reset store.proposer_boost_root if this is a new slot
//    if current_slot > previous_slot:
//        store.proposer_boost_root = Root()
//
//    # Not a new epoch, return
//    if not (current_slot > previous_slot and compute_slots_since_epoch_start(current_slot) == 0):
//        return
//
//    # Update store.justified_checkpoint if a better checkpoint on the store.finalized_checkpoint chain
//    if store.best_justified_checkpoint.epoch > store.justified_checkpoint.epoch:
//        finalized_slot = compute_start_slot_at_epoch(store.finalized_checkpoint.epoch)
//        ancestor_at_finalized_slot = get_ancestor(store, store.best_justified_checkpoint.root, finalized_slot)
//        if ancestor_at_finalized_slot == store.finalized_checkpoint.root:
//            store.justified_checkpoint = store.best_justified_checkpoint
func (s *Service) OnTick(ctx context.Context, time uint64) error {
	prevSlot := s.slotInStore()
	s.store.time = time
	currentSlot := s.slotInStore()
	if currentSlot > prevSlot {
		s.store.proposerBoostRoot = [32]byte{}
	}
	if !(currentSlot > prevSlot && slots.IsEpochStart(currentSlot)) {
		return nil
	}
	// Update store.justified_checkpoint if a better checkpoint on the store.finalized_checkpoint chain
	if s.store.bestJustifiedCheckpt.Epoch > s.store.justifiedCheckpt.Epoch {
		finalizedSlot, err := slots.EpochStart(s.store.finalizedCheckpt.Epoch)
		if err != nil {
			return err
		}
		r, err := s.ancestor(ctx, s.store.bestJustifiedCheckpt.Root, finalizedSlot)
		if err != nil {
			return err
		}
		if bytes.Equal(r, s.store.finalizedCheckpt.Root) {
			s.store.justifiedCheckpt = s.store.bestJustifiedCheckpt
		}
	}
	return nil
}

// slotInStore returns the slot in the store.
// def get_current_slot(store: Store) -> Slot:
//    return Slot(GENESIS_SLOT + get_slots_since_genesis(store))
func (s *Service) slotInStore() types.Slot {
	return types.Slot((s.store.time - s.store.genesisTime) / params.BeaconConfig().SecondsPerSlot)
}

// StoreTime returns the time in the store.
func (s *Service) StoreTime() uint64 {
	return s.store.time
}

// JustifiedCheckpoint returns the justified checkpoint in the store.
func (s *Service) JustifiedCheckpoint() *ethpb.Checkpoint {
	return s.store.justifiedCheckpt
}

// BestJustifiedCheckpoint returns the best justified checkpoint in the store.
func (s *Service) BestJustifiedCheckpoint() *ethpb.Checkpoint {
	return s.store.bestJustifiedCheckpt
}

// FinalizedCheckpoint returns the finalized checkpoint in the store.
func (s *Service) FinalizedCheckpoint() *ethpb.Checkpoint {
	return s.store.finalizedCheckpt
}