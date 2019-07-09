package forkchoice

import (
	"bytes"
	"context"
	"fmt"

	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
)

type Store struct {
	ctx              context.Context
	cancel           context.CancelFunc
	time             uint64
	justifiedCheckpt *pb.Checkpoint
	finalizedCheckpt *pb.Checkpoint
	db               *db.BeaconDB
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
func (s *Store) GensisStore(state *pb.BeaconState) error {

	stateRoot, err := ssz.HashTreeRoot(state)
	if err != nil {
		return fmt.Errorf("could not tree hash genesis state: %v", err)
	}

	genesisBlk := &pb.BeaconBlock{StateRoot: stateRoot[:]}

	blkRoot, err := ssz.HashTreeRoot(genesisBlk)
	if err != nil {
		return fmt.Errorf("could not tree hash genesis block: %v", err)
	}

	s.time = state.GenesisTime
	s.justifiedCheckpt = &pb.Checkpoint{Epoch: 0, Root: blkRoot[:]}
	s.finalizedCheckpt = &pb.Checkpoint{Epoch: 0, Root: blkRoot[:]}

	if err := s.db.SaveBlock(genesisBlk); err != nil {
		return fmt.Errorf("could not save genesis block: %v", err)
	}
	if err := s.db.SaveState(s.ctx, state); err != nil {
		return fmt.Errorf("could not save genesis state: %v", err)
	}
	if err := s.db.SaveCheckpointState(s.ctx, s.justifiedCheckpt, state); err != nil {
		return fmt.Errorf("could not save justified checkpt: %v", err)
	}
	if err := s.db.SaveCheckpointState(s.ctx, s.finalizedCheckpt, state); err != nil {
		return fmt.Errorf("could not save finalized checkpt: %v", err)
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
		return nil, fmt.Errorf("could not get ancestor block: %v", err)
	}

	if b.Slot < slot {
		return nil, fmt.Errorf("ancestor slot %d reacched below wanted slot %d", b.Slot, slot)
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
		return 0, fmt.Errorf("could not get checkpoint state: %v", err)
	}
	lastJustifiedEpoch := helpers.CurrentEpoch(lastJustifiedState)
	activeIndices, err := helpers.ActiveValidatorIndices(lastJustifiedState, lastJustifiedEpoch)
	if err != nil {
		return 0, fmt.Errorf("could not get active indices for last checkpoint state: %v", err)
	}

	wantedBlk, err := s.db.Block(bytesutil.ToBytes32(root))
	if err != nil {
		return 0, fmt.Errorf("could not get slot for an ancestor block: %v", err)
	}
	balances := uint64(0)

	for _, i := range activeIndices {
		if s.db.HasLatestMessage(i) {
			msg, err := s.db.LatestMessage(i)
			if err != nil {
				return 0, fmt.Errorf("could not get validator %d's latest msg: %v", i, err)
			}
			wantedRoot, err := s.Ancestor(msg.Root, wantedBlk.Slot)
			if err != nil {
				return 0, fmt.Errorf("could not get ancestor root for slot %d: %v", wantedBlk.Slot, err)
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

	}
}
