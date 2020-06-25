package blockchain

import (
	"context"

	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
)

func (s *Service) generateState(ctx context.Context, startRoot [32]byte, endRoot [32]byte) (*stateTrie.BeaconState, error) {
	preState, err := s.beaconDB.State(ctx, startRoot)
	if err != nil {
		return nil, err
	}
	if preState == nil {
		if !s.stateGen.HasState(ctx, startRoot) {
			if err := s.beaconDB.SaveBlocks(ctx, s.getInitSyncBlocks()); err != nil {
				return nil, errors.Wrap(err, "could not save initial sync blocks")
			}
			s.clearInitSyncBlocks()
		}
		preState, err = s.stateGen.StateByRoot(ctx, startRoot)
		if err != nil {
			return nil, err
		}
		if preState == nil {
			return nil, errors.New("finalized state does not exist in db")
		}
	}

	if err := s.beaconDB.SaveBlocks(ctx, s.getInitSyncBlocks()); err != nil {
		return nil, err
	}
	var endBlock *ethpb.SignedBeaconBlock
	if s.hasInitSyncBlock(endRoot) {
		endBlock = s.getInitSyncBlock(endRoot)
		s.clearInitSyncBlocks()
	} else {
		endBlock, err = s.beaconDB.Block(ctx, endRoot)
		if err != nil {
			return nil, err
		}
	}

	if endBlock == nil {
		return nil, errors.New("provided block root does not have block saved in the db")
	}
	log.Warnf("Generating missing state of slot %d and root %#x", endBlock.Block.Slot, endRoot)

	blocks, err := s.stateGen.LoadBlocks(ctx, preState.Slot()+1, endBlock.Block.Slot, endRoot)
	if err != nil {
		return nil, errors.Wrap(err, "could not load the required blocks")
	}
	postState, err := s.stateGen.ReplayBlocks(ctx, preState, blocks, endBlock.Block.Slot)
	if err != nil {
		return nil, errors.Wrap(err, "could not replay the blocks to generate the resultant state")
	}
	return postState, nil
}

// This saves a beacon block to the initial sync blocks cache.
func (s *Service) saveInitSyncBlock(r [32]byte, b *ethpb.SignedBeaconBlock) {
	s.initSyncBlocksLock.Lock()
	defer s.initSyncBlocksLock.Unlock()
	s.initSyncBlocks[r] = b
}

// This checks if a beacon block exists in the initial sync blocks cache using the root
// of the block.
func (s *Service) hasInitSyncBlock(r [32]byte) bool {
	s.initSyncBlocksLock.RLock()
	defer s.initSyncBlocksLock.RUnlock()
	_, ok := s.initSyncBlocks[r]
	return ok
}

// This retrieves a beacon block from the initial sync blocks cache using the root of
// the block.
func (s *Service) getInitSyncBlock(r [32]byte) *ethpb.SignedBeaconBlock {
	s.initSyncBlocksLock.RLock()
	defer s.initSyncBlocksLock.RUnlock()
	b := s.initSyncBlocks[r]
	return b
}

// This retrieves all the beacon blocks from the initial sync blocks cache, the returned
// blocks are unordered.
func (s *Service) getInitSyncBlocks() []*ethpb.SignedBeaconBlock {
	s.initSyncBlocksLock.RLock()
	defer s.initSyncBlocksLock.RUnlock()

	blks := make([]*ethpb.SignedBeaconBlock, 0, len(s.initSyncBlocks))
	for _, b := range s.initSyncBlocks {
		blks = append(blks, b)
	}
	return blks
}

// This clears out the initial sync blocks cache.
func (s *Service) clearInitSyncBlocks() {
	s.initSyncBlocksLock.Lock()
	defer s.initSyncBlocksLock.Unlock()
	s.initSyncBlocks = make(map[[32]byte]*ethpb.SignedBeaconBlock)
}
