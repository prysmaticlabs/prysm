package db

import (
	"fmt"

	"github.com/boltdb/bolt"
	"github.com/prysmaticlabs/prysm/beacon-chain/utils"
)

// ReadBlockVoteCache read block vote cache from DB.
func (db *BeaconDB) ReadBlockVoteCache(blockHashes [][32]byte) (utils.BlockVoteCache, error) {
	blockVoteCache := utils.NewBlockVoteCache()
	err := db.view(func(tx *bolt.Tx) error {
		blockVoteCacheInfo := tx.Bucket(blockVoteCacheBucket)
		for _, h := range blockHashes {
			blob := blockVoteCacheInfo.Get(h[:])
			if blob == nil {
				continue
			}
			vote := new(utils.BlockVote)
			if err := vote.Unmarshal(blob); err != nil {
				return fmt.Errorf("failed to decode block vote cache for block hash %x", h)
			}
			blockVoteCache[h] = vote
		}
		return nil
	})
	return blockVoteCache, err
}

// DeleteBlockVoteCache removes vote cache for specified blocks from DB.
func (db *BeaconDB) DeleteBlockVoteCache(blockHashes [][32]byte) error {
	err := db.update(func(tx *bolt.Tx) error {
		blockVoteCacheInfo := tx.Bucket(blockVoteCacheBucket)
		for _, h := range blockHashes {
			if err := blockVoteCacheInfo.Delete(h[:]); err != nil {
				return fmt.Errorf("failed to delete block vote cache for block hash %x: %v", h, err)
			}
		}
		return nil
	})
	return err
}

// WriteBlockVoteCache write block vote cache object into DB.
func (db *BeaconDB) WriteBlockVoteCache(blockVoteCache utils.BlockVoteCache) error {
	err := db.update(func(tx *bolt.Tx) error {
		blockVoteCacheInfo := tx.Bucket(blockVoteCacheBucket)
		for h := range blockVoteCache {
			blob, err := blockVoteCache[h].Marshal()
			if err != nil {
				return fmt.Errorf("failed to encode block vote cache for block hash %x", h)
			}
			if err = blockVoteCacheInfo.Put(h[:], blob); err != nil {
				return fmt.Errorf("failed to store block vote cache into DB")
			}
		}
		return nil
	})
	return err
}
