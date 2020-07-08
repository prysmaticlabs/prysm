package blockchain

import (
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
)

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
