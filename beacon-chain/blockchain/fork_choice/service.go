package forkchoice

import (
	"bytes"
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/sirupsen/logrus"
)

var log = logrus.WithField("prefix", "forkchoice")

type Store struct {
	ctx              context.Context
	cancel           context.CancelFunc
	time             uint64
	db               *db.BeaconDB
	justifiedCheckpt *ethpb.Checkpoint
	finalizedCheckpt *ethpb.Checkpoint
}

func NewForkChoiceService(ctx context.Context, db *db.BeaconDB) *Store {
	ctx, cancel := context.WithCancel(ctx)

	return &Store{
		ctx:    ctx,
		cancel: cancel,
		db:     db,
	}
}

// GensisStore to be filled
//
// Spec pseudocode definition:
//   def get_genesis_store(genesis_state: BeaconState) -> Store:
//    genesis_block = BeaconBlock(state_root=hash_tree_root(genesis_state))
//    root = signing_root(genesis_block)
//    justified_checkpoint = Checkpoint(epoch=GENESIS_EPOCH, root=root)
//    finalized_checkpoint = Checkpoint(epoch=GENESIS_EPOCH, root=root)
//    return Store(
//        time=genesis_state.genesis_time,
//        justified_checkpoint=justified_checkpoint,
//        finalized_checkpoint=finalized_checkpoint,
//        blocks={root: genesis_block},
//        block_states={root: genesis_state.copy()},
//        checkpoint_states={justified_checkpoint: genesis_state.copy()},
//    )
func (s *Store) GensisStore(genesisState *pb.BeaconState) error {
	stateRoot, err := ssz.HashTreeRoot(genesisState)
	if err != nil {
		return errors.Wrap(err, "could not tree hash genesis state")
	}

	genesisBlk := blocks.NewGenesisBlock(stateRoot[:])

	blkRoot, err := ssz.SigningRoot(genesisBlk)
	if err != nil {
		return errors.Wrap(err, "could not tree hash genesis block")
	}

	s.time = genesisState.GenesisTime
	s.justifiedCheckpt = &ethpb.Checkpoint{Epoch: 0, Root: blkRoot[:]}
	s.finalizedCheckpt = &ethpb.Checkpoint{Epoch: 0, Root: blkRoot[:]}

	if err := s.db.SaveBlock(genesisBlk); err != nil {
		return errors.Wrap(err, "could not save genesis block")
	}
	if err := s.db.SaveForkChoiceState(s.ctx, genesisState, blkRoot[:]); err != nil {
		return errors.Wrap(err, "could not save genesis state")
	}
	if err := s.db.SaveCheckpointState(s.ctx, genesisState, s.justifiedCheckpt); err != nil {
		return errors.Wrap(err, "could not save justified checkpt")
	}

	return nil
}

// Ancestor to be filled
//
// Spec pseudocode definition:
//   def get_ancestor(store: Store, root: Hash, slot: Slot) -> Hash:
//    block = store.blocks[root]
//    assert block.slot >= slot
//    return root if block.slot == slot else get_ancestor(store, block.parent_root, slot)
func (s *Store) Ancestor(root []byte, slot uint64) ([]byte, error) {
	b, err := s.db.Block(bytesutil.ToBytes32(root))
	if err != nil {
		return nil, errors.Wrap(err, "could not get ancestor block")
	}

	// If we dont have the ancestor in the DB, simply return nil so rest of fork choice
	// operation can proceed. This is not an error condition.
	if b == nil || b.Slot < slot {
		return nil, nil
	}

	if b.Slot == slot {
		return root, nil
	}

	return s.Ancestor(b.ParentRoot, slot)
}

// LatestAttestingBalance to be filled
//
// Spec pseudocode definition:
//   def get_latest_attesting_balance(store: Store, root: Hash) -> Gwei:
//    state = store.checkpoint_states[store.justified_checkpoint]
//    active_indices = get_active_validator_indices(state, get_current_epoch(state))
//    return Gwei(sum(
//        state.validators[i].effective_balance for i in active_indices
//        if (i in store.latest_messages
//            and get_ancestor(store, store.latest_messages[i].root, store.blocks[root].slot) == root)
//    ))
func (s *Store) LatestAttestingBalance(root []byte) (uint64, error) {
	lastJustifiedState, err := s.db.CheckpointState(s.ctx, s.justifiedCheckpt)
	if err != nil {
		return 0, errors.Wrap(err, "could not get checkpoint state")
	}
	if lastJustifiedState == nil {
		return 0, errors.Wrapf(err, "could not get justified state at epoch %d", s.justifiedCheckpt.Epoch)
	}

	lastJustifiedEpoch := helpers.CurrentEpoch(lastJustifiedState)
	activeIndices, err := helpers.ActiveValidatorIndices(lastJustifiedState, lastJustifiedEpoch)
	if err != nil {
		return 0, errors.Wrap(err, "could not get active indices for last checkpoint state")
	}

	wantedBlk, err := s.db.Block(bytesutil.ToBytes32(root))
	if err != nil {
		return 0, errors.Wrap(err, "could not get slot for an ancestor block")
	}

	balances := uint64(0)
	for _, i := range activeIndices {
		if s.db.HasLatestMessage(i) {
			msg, err := s.db.LatestMessage(i)
			if err != nil {
				return 0, errors.Wrapf(err, "could not get validator %d's latest msg", i)
			}

			wantedRoot, err := s.Ancestor(msg.Root, wantedBlk.Slot)
			if err != nil {
				return 0, errors.Wrapf(err, "could not get ancestor root for slot %d", wantedBlk.Slot)
			}
			if bytes.Equal(wantedRoot, root) {
				balances += lastJustifiedState.Validators[i].EffectiveBalance
			}
		}
	}
	return balances, nil
}

// Head to be filled
//
// Spec pseudocode definition:
//   def get_head(store: Store) -> Hash:
//    # Execute the LMD-GHOST fork choice
//    head = store.justified_checkpoint.root
//    justified_slot = compute_start_slot_of_epoch(store.justified_checkpoint.epoch)
//    while True:
//        children = [
//            root for root in store.blocks.keys()
//            if store.blocks[root].parent_root == head and store.blocks[root].slot > justified_slot
//        ]
//        if len(children) == 0:
//            return head
//        # Sort by latest attesting balance with ties broken lexicographically
//        head = max(children, key=lambda root: (get_latest_attesting_balance(store, root), root))
func (s *Store) Head() ([]byte, error) {
	head := s.justifiedCheckpt.Root
	justifiedSlot := helpers.StartSlot(s.justifiedCheckpt.Epoch)

	for {
		children, err := s.db.ChildrenBlocksFromParent(head, justifiedSlot)
		if err != nil {
			return nil, errors.Wrap(err, "could not retrieve children info")
		}

		if len(children) == 0 {
			return head, nil
		}

		head = children[0]
		highest, err := s.LatestAttestingBalance(head)
		if err != nil {
			return nil, errors.Wrap(err, "could not get latest balance")
		}
		for _, child := range children[1:] {
			balance, err := s.LatestAttestingBalance(child)
			if err != nil {
				return nil, errors.Wrap(err, "could not get latest balance")
			}
			if balance > highest {
				highest = balance
				head = child
			}
		}
	}
}

// OnTick to be filled
//
// Spec pseudocode definition:
//   def on_tick(store: Store, time: uint64) -> None:
//    store.time = time
func (s *Store) OnTick(t uint64) {
	s.time = t
}

// OnBlock to be filled
//
// Spec pseudocode definition:
//   def on_block(store: Store, block: BeaconBlock) -> None:
//    # Make a copy of the state to avoid mutability issues
//    assert block.parent_root in store.block_states
//    pre_state = store.block_states[block.parent_root].copy()
//    # Blocks cannot be in the future. If they are, their consideration must be delayed until the are in the past.
//    assert store.time >= pre_state.genesis_time + block.slot * SECONDS_PER_SLOT
//    # Add new block to the store
//    store.blocks[signing_root(block)] = block
//    # Check block is a descendant of the finalized block
//    assert (
//        get_ancestor(store, signing_root(block), store.blocks[store.finalized_checkpoint.root].slot) ==
//        store.finalized_checkpoint.root
//    )
//    # Check that block is later than the finalized epoch slot
//    assert block.slot > compute_start_slot_of_epoch(store.finalized_checkpoint.epoch)
//    # Check the block is valid and compute the post-state
//    state = state_transition(pre_state, block)
//    # Add new state for this block to the store
//    store.block_states[signing_root(block)] = state
//
//    # Update justified checkpoint
//    if state.current_justified_checkpoint.epoch > store.justified_checkpoint.epoch:
//        store.justified_checkpoint = state.current_justified_checkpoint
//
//    # Update finalized checkpoint
//    if state.finalized_checkpoint.epoch > store.finalized_checkpoint.epoch:
//        store.finalized_checkpoint = state.finalized_checkpoint
func (s *Store) OnBlock(b *ethpb.BeaconBlock) error {
	preState, err := s.db.ForkChoiceState(s.ctx, b.ParentRoot)
	if err != nil {
		return errors.Wrapf(err, "could not get pre state for slot %d", b.Slot)
	}
	if preState == nil {
		return fmt.Errorf("pre state of slot %d does not exist", b.Slot)
	}

	// Blocks cannot be in the future. If they are, their consideration must be delayed until the are in the past.
	slotTime := preState.GenesisTime + b.Slot*params.BeaconConfig().SecondsPerSlot
	if slotTime > s.time {
		return fmt.Errorf("could not process block from the future, slot time %d > current time %d", slotTime, s.time)
	}

	if err := s.db.SaveBlock(b); err != nil {
		return errors.Wrapf(err, "could not save block from slot %d", b.Slot)
	}

	// Verify block is a descendent of a finalized block.
	finalizedBlk, err := s.db.Block(bytesutil.ToBytes32(s.finalizedCheckpt.Root))
	if err != nil || finalizedBlk == nil {
		return errors.Wrap(err, "could not get finalized block")
	}
	root, err := ssz.SigningRoot(b)
	if err != nil {
		return errors.Wrapf(err, "could not get sign root of block %d", b.Slot)
	}

	bFinalizedRoot, err := s.Ancestor(root[:], finalizedBlk.Slot)
	if !bytes.Equal(bFinalizedRoot, s.finalizedCheckpt.Root) {
		return fmt.Errorf("block from slot %d is not a descendent of the current finalized block", b.Slot)
	}

	// Verify block is later than the finalized epoch slot.
	finalizedSlot := helpers.StartSlot(s.finalizedCheckpt.Epoch)
	if finalizedSlot >= b.Slot {
		return fmt.Errorf("block is equal or earlier than finalized block, slot %d < slot %d", b.Slot, finalizedSlot)
	}

	// Apply new state transition for the block to the store.
	// Make block root as bad to reject in sync.
	postState, err := state.ExecuteStateTransition(s.ctx, preState, b)
	if err != nil {
		if err := s.db.DeleteBlock(b); err != nil {
			return errors.Wrap(err, "could not delete bad block from db")
		}
		return errors.Wrap(err, "could not execute state transition")
	}

	if err := s.db.SaveForkChoiceState(s.ctx, postState, root[:]); err != nil {
		return errors.Wrap(err, "could not save state")
	}

	// Update justified check point.
	if postState.CurrentJustifiedCheckpoint.Epoch > s.justifiedCheckpt.Epoch {
		s.justifiedCheckpt = postState.CurrentJustifiedCheckpoint
	}

	// Update finalized check point.
	// Prune the block cache and helper caches on every new finalized epoch.
	if postState.FinalizedCheckpoint.Epoch > s.finalizedCheckpt.Epoch {
		helpers.ClearAllCaches()
		s.db.ClearBlockCache()
		s.finalizedCheckpt.Epoch = postState.FinalizedCheckpoint.Epoch
	}

	if helpers.IsEpochStart(postState.Slot) {
		if err := saveValidatorIdx(postState, s.db); err != nil {
			return errors.Wrap(err, "could not save validator index")
		}
		if err := deleteValidatorIdx(postState, s.db); err != nil {
			return errors.Wrap(err, "could not delete validator index")
		}
		logEpochData(postState)
	}

	return nil
}

// OnAttestation to be filled
//
// Spec pseudocode definition:
//   def on_attestation(store: Store, attestation: Attestation) -> None:
//    target = attestation.data.target
//
//    # Cannot calculate the current shuffling if have not seen the target
//    assert target.root in store.blocks
//
//    # Attestations cannot be from future epochs. If they are, delay consideration until the epoch arrives
//    base_state = store.block_states[target.root].copy()
//    assert store.time >= base_state.genesis_time + compute_start_slot_of_epoch(target.epoch) * SECONDS_PER_SLOT
//
//    # Store target checkpoint state if not yet seen
//    if target not in store.checkpoint_states:
//        process_slots(base_state, compute_start_slot_of_epoch(target.epoch))
//        store.checkpoint_states[target] = base_state
//    target_state = store.checkpoint_states[target]
//
//    # Attestations can only affect the fork choice of subsequent slots.
//    # Delay consideration in the fork choice until their slot is in the past.
//    attestation_slot = get_attestation_data_slot(target_state, attestation.data)
//    assert store.time >= (attestation_slot + 1) * SECONDS_PER_SLOT
//
//    # Get state at the `target` to validate attestation and calculate the committees
//    indexed_attestation = get_indexed_attestation(target_state, attestation)
//    assert is_valid_indexed_attestation(target_state, indexed_attestation)
//
//    # Update latest messages
//    for i in indexed_attestation.custody_bit_0_indices + indexed_attestation.custody_bit_1_indices:
//        if i not in store.latest_messages or target.epoch > store.latest_messages[i].epoch:
//            store.latest_messages[i] = LatestMessage(epoch=target.epoch, root=attestation.data.beacon_block_root)
func (s *Store) OnAttestation(a *ethpb.Attestation) error {
	tgt := a.Data.Target

	// Verify beacon node has seen the target block before.
	if !s.db.HasBlock(bytesutil.ToBytes32(tgt.Root)) {
		return fmt.Errorf("target root %#x does not exist in db", bytesutil.Trunc(tgt.Root))
	}

	// Verify Attestations cannot be from future epochs.
	// If they are, delay consideration until the epoch arrives
	tgtSlot := helpers.StartSlot(tgt.Epoch)
	baseState, err := s.db.ForkChoiceState(s.ctx, tgt.Root)
	if err != nil {
		return errors.Wrapf(err, "could not get pre state for slot %d", tgtSlot)
	}
	if baseState == nil {
		return fmt.Errorf("pre state of target block %d does not exist", tgtSlot)
	}

	slotTime := baseState.GenesisTime + tgtSlot*params.BeaconConfig().SecondsPerSlot
	if slotTime > s.time {
		return fmt.Errorf("could not process attestation from the future epoch, time %d > time %d", slotTime, s.time)
	}

	// Store target checkpoint state if not yet seen.
	exists, err := s.db.HasCheckpoint(tgt)
	if err != nil {
		return errors.Wrap(err, "could not get check point state")
	}
	if !exists {
		baseState, err = state.ProcessSlots(s.ctx, baseState, tgtSlot)
		if err != nil {
			return errors.Wrapf(err, "could not process slots up to %d", tgtSlot)
		}
		if err := s.db.SaveCheckpointState(s.ctx, baseState, tgt); err != nil {
			return errors.Wrap(err, "could not save check point state")
		}
	}

	// Verify attestations can only affect the fork choice of subsequent slots.
	aSlot, err := helpers.AttestationDataSlot(baseState, a.Data)
	if err != nil {
		return errors.Wrap(err, "could not get attestation slot")
	}
	slotTime = baseState.GenesisTime + (aSlot+1)*params.BeaconConfig().SecondsPerSlot
	if slotTime > s.time {
		return fmt.Errorf("could not process attestation for fork choice until inclusion delay, time %d > time %d", slotTime, s.time)
	}

	// Use the target state to to validate attestation and calculate the committees.
	indexedAtt, err := blocks.ConvertToIndexed(baseState, a)
	if err != nil {
		return errors.Wrap(err, "could not convert attestation to indexed attestation")
	}

	if err := blocks.VerifyIndexedAttestation(baseState, indexedAtt); err != nil {
		return errors.New("could not verify indexed attestation")
	}

	// Update every validator's latest message.
	for _, i := range append(indexedAtt.CustodyBit_0Indices, indexedAtt.CustodyBit_1Indices...) {
		s.db.HasLatestMessage(i)
		msg, err := s.db.LatestMessage(i)
		if err != nil {
			return errors.Wrapf(err, "could not get latest msg for validator %d", i)
		}
		if !s.db.HasLatestMessage(i) || tgt.Epoch > msg.Epoch {
			if err := s.db.SaveLatestMessage(s.ctx, i, &pb.LatestMessage{
				Epoch: tgt.Epoch,
				Root:  a.Data.BeaconBlockRoot,
			}); err != nil {
				return errors.Wrapf(err, "could not save latest msg for validator %d", i)
			}
		}
	}
	return nil
}

// FinalizedCheckpt returns the latest finalized check point from fork choice store.
func (s *Store) FinalizedCheckpt() *ethpb.Checkpoint {
	return s.finalizedCheckpt
}

// JustifiedCheckpt returns the latest justified check point from fork choice store.
func (s *Store) JustifiedCheckpt() *ethpb.Checkpoint {
	return s.justifiedCheckpt
}
