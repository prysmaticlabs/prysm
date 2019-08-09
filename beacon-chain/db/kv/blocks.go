package kv

import (
	"context"

	"github.com/prysmaticlabs/prysm/beacon-chain/db/filters"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
)

// Block retrival by root.
// TODO(#3164): Implement.
func (k *Store) Block(ctx context.Context, blockRoot [32]byte) (*ethpb.BeaconBlock, error) {
	return nil, nil
}

// Blocks retrieves a list of beacon blocks by filter criteria.
// TODO(#3164): Implement.
func (k *Store) Blocks(ctx context.Context, f *filters.QueryFilter) ([]*ethpb.BeaconBlock, error) {
	return nil, nil
}

// BlockRoots retrieves a list of beacon block roots by filter criteria.
// TODO(#3164): Implement.
func (k *Store) BlockRoots(ctx context.Context, f *filters.QueryFilter) ([][]byte, error) {
	return nil, nil
}

// HasBlock checks if a block by root exists in the db.
// TODO(#3164): Implement.
func (k *Store) HasBlock(ctx context.Context, blockRoot [32]byte) bool {
	return false
}

// DeleteBlock by block root.
// TODO(#3164): Implement.
func (k *Store) DeleteBlock(ctx context.Context, blockRoot [32]byte) error {
	return nil
}

// SaveBlock to the db.
// TODO(#3164): Implement.
func (k *Store) SaveBlock(ctx context.Context, block *ethpb.BeaconBlock) error {
	return nil
}

// SaveBlocks via batch updates to the db.
// TODO(#3164): Implement.
func (k *Store) SaveBlocks(ctx context.Context, blocks []*ethpb.BeaconBlock) error {
	return nil
}
