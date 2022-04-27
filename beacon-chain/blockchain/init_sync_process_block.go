package blockchain

import (
	"context"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/block"
)

var errBlockNotFoundInCacheOrDB = errors.New("block not found in cache or db")

// This saves a beacon block to the initial sync blocks cache.
func (s *Service) saveInitSyncBlock(r [32]byte, b block.SignedBeaconBlock) {
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

// Returns block for a given root `r` from either the initial sync blocks cache or the DB.
// Error is returned if the block is not found in either cache or DB.
func (s *Service) getBlock(ctx context.Context, r [32]byte) (block.SignedBeaconBlock, error) {
	s.initSyncBlocksLock.RLock()
	defer s.initSyncBlocksLock.RUnlock()

	// Check cache first because it's faster.
	b, ok := s.initSyncBlocks[r]
	var err error
	if !ok {
		b, err = s.cfg.BeaconDB.Block(ctx, r)
		if err != nil {
			return nil, errors.Wrap(err, "could not retrieve block from db")
		}
	}
	if err := helpers.BeaconBlockIsNil(b); err != nil {
		return nil, errBlockNotFoundInCacheOrDB
	}
	return b, nil
}

// This retrieves all the beacon blocks from the initial sync blocks cache, the returned
// blocks are unordered.
func (s *Service) getInitSyncBlocks() []block.SignedBeaconBlock {
	s.initSyncBlocksLock.RLock()
	defer s.initSyncBlocksLock.RUnlock()

	blks := make([]block.SignedBeaconBlock, 0, len(s.initSyncBlocks))
	for _, b := range s.initSyncBlocks {
		blks = append(blks, b)
	}
	return blks
}

// This clears out the initial sync blocks cache.
func (s *Service) clearInitSyncBlocks() {
	s.initSyncBlocksLock.Lock()
	defer s.initSyncBlocksLock.Unlock()
	s.initSyncBlocks = make(map[[32]byte]block.SignedBeaconBlock)
}
