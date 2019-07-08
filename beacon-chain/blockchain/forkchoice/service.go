package forkchoice

import (
	"context"
	"fmt"
	"github.com/prysmaticlabs/go-ssz"

	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

type Store struct {
	ctx context.Context
	cancel              context.CancelFunc
	time uint64
	justifiedCheckpt *pb.Checkpoint
	finalizedCheckpt *pb.Checkpoint
	db         *db.BeaconDB
}

func NewForkChoiceService(ctx context.Context, db *db.BeaconDB) *Store {
	ctx, cancel := context.WithCancel(ctx)

	return &Store{
		ctx: ctx,
		cancel: cancel,
		db: db,
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
	if err := s.db.SaveCheckpoint(s.ctx, s.justifiedCheckpt); err != nil {
		return fmt.Errorf("could not save justified checkpt: %v", err)
	}
	if err := s.db.SaveCheckpoint(s.ctx, s.finalizedCheckpt); err != nil {
		return fmt.Errorf("could not save finalized checkpt: %v", err)
	}

	return nil
}
