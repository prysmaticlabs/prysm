package kv

import (
	"bytes"
	"context"
	"reflect"

	"github.com/boltdb/bolt"
	"github.com/gogo/protobuf/proto"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/db/filters"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
)

// Block retrieval by root.
func (k *Store) Block(ctx context.Context, blockRoot [32]byte) (*ethpb.BeaconBlock, error) {
	block := &ethpb.BeaconBlock{}
	err := k.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(blocksBucket)
		enc := bkt.Get(blockRoot[:])
		if enc == nil {
			return nil
		}
		return proto.Unmarshal(enc, block)
	})
	return block, err
}

// HeadBlock returns the latest canonical block in eth2.
func (k *Store) HeadBlock(ctx context.Context) (*ethpb.BeaconBlock, error) {
	headBlock := &ethpb.BeaconBlock{}
	err := k.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(blocksBucket)
		headRoot := bkt.Get(headBlockRootKey)
		if headRoot == nil {
			return nil
		}
		enc := bkt.Get(headRoot)
		if enc == nil {
			return nil
		}
		return proto.Unmarshal(enc, headBlock)
	})
	return headBlock, err
}

// Blocks retrieves a list of beacon blocks by filter criteria.
func (k *Store) Blocks(ctx context.Context, f *filters.QueryFilter) ([]*ethpb.BeaconBlock, error) {
	blocks := make([]*ethpb.BeaconBlock, 0)
	hasFilterSpecified := !reflect.DeepEqual(f, &filters.QueryFilter{}) && f != nil
	err := k.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(blocksBucket)
		c := bkt.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			if v != nil {
				block := &ethpb.BeaconBlock{}
				if err := proto.Unmarshal(v, block); err != nil {
					return err
				}
				if !hasFilterSpecified || ensureBlockFilterCriteria(k, block, f) {
					blocks = append(blocks, block)
				}
			}
		}
		return nil
	})
	return blocks, err
}

// BlockRoots retrieves a list of beacon block roots by filter criteria.
func (k *Store) BlockRoots(ctx context.Context, f *filters.QueryFilter) ([][]byte, error) {
	roots := make([][]byte, 0)
	hasFilterSpecified := !reflect.DeepEqual(f, &filters.QueryFilter{}) && f != nil
	err := k.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(blocksBucket)
		c := bkt.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			if v != nil {
				block := &ethpb.BeaconBlock{}
				if err := proto.Unmarshal(v, block); err != nil {
					return err
				}
				if !hasFilterSpecified || ensureBlockFilterCriteria(k, block, f) {
					roots = append(roots, k)
				}
			}
		}
		return nil
	})
	return roots, err
}

// HasBlock checks if a block by root exists in the db.
func (k *Store) HasBlock(ctx context.Context, blockRoot [32]byte) bool {
	exists := false
	// #nosec G104. Always returns nil.
	k.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(blocksBucket)
		exists = bkt.Get(blockRoot[:]) != nil
		return nil
	})
	return exists
}

// DeleteBlock by block root.
func (k *Store) DeleteBlock(ctx context.Context, blockRoot [32]byte) error {
	return k.db.Update(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(blocksBucket)
		return bkt.Delete(blockRoot[:])
	})
}

// SaveBlock to the db.
func (k *Store) SaveBlock(ctx context.Context, block *ethpb.BeaconBlock) error {
	key, err := generateBlockKey(block)
	if err != nil {
		return err
	}
	enc, err := proto.Marshal(block)
	if err != nil {
		return err
	}
	return k.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(blocksBucket)
		return bucket.Put(key, enc)
	})
}

// SaveBlocks via batch updates to the db.
func (k *Store) SaveBlocks(ctx context.Context, blocks []*ethpb.BeaconBlock) error {
	encodedValues := make([][]byte, len(blocks))
	keys := make([][]byte, len(blocks))
	for i := 0; i < len(blocks); i++ {
		enc, err := proto.Marshal(blocks[i])
		if err != nil {
			return err
		}
		key, err := generateBlockKey(blocks[i])
		if err != nil {
			return err
		}
		encodedValues[i] = enc
		keys[i] = key
	}
	return k.db.Batch(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(blocksBucket)
		for i := 0; i < len(blocks); i++ {
			if err := bucket.Put(keys[i], encodedValues[i]); err != nil {
				return err
			}
		}
		return nil
	})
}

// SaveHeadBlockRoot to the db.
func (k *Store) SaveHeadBlockRoot(ctx context.Context, blockRoot [32]byte) error {
	return k.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(blocksBucket)
		return bucket.Put(headBlockRootKey, blockRoot[:])
	})
}

func generateBlockKey(block *ethpb.BeaconBlock) ([]byte, error) {
	blockRoot, err := ssz.HashTreeRoot(block)
	if err != nil {
		return nil, err
	}
	return blockRoot[:], nil
}

// ensureBlockFilterCriteria uses a set of specified filters
// to ensure the byte key used for db lookups contains the correct values
// requested by the filter. For example, if a key looks like:
// root-0x23923-parent-root-0x49349 and our filter criteria wants the key to
// contain parent root 0x49349 and root 0x99283, the key will NOT meet all the filter
// criteria and the function will return false.
func ensureBlockFilterCriteria(key []byte, block *ethpb.BeaconBlock, f *filters.QueryFilter) bool {
	numCriteriaMet := 0
	for k, v := range f.Filters() {
		switch k {
		case filters.Root:
			root := v.([]byte)
			if bytes.Contains(key, append([]byte("root"), root[:]...)) {
				numCriteriaMet++
			}
		case filters.ParentRoot:
			if bytes.Equal(block.ParentRoot, v.([]byte)) {
				numCriteriaMet++
			}
		case filters.StartSlot:
			if block.Slot >= v.(uint64) {
				numCriteriaMet++
			}
		case filters.EndSlot:
			if block.Slot <= v.(uint64) {
				numCriteriaMet++
			}
		case filters.StartEpoch:
			if helpers.SlotToEpoch(block.Slot) >= v.(uint64) {
				numCriteriaMet++
			}
		case filters.EndEpoch:
			if helpers.SlotToEpoch(block.Slot) <= v.(uint64) {
				numCriteriaMet++
			}
		}
	}
	return numCriteriaMet == len(f.Filters())
}
