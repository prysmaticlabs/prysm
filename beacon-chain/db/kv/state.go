package kv

import (
	"context"

	"github.com/prysmaticlabs/prysm/beacon-chain/db/filters"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

// State retrieval by filter criteria.
func (k *KVStore) State(ctx context.Context, f *filters.QueryFilter) (*pb.BeaconState, error) {
	return nil, nil
}

// HeadState returns the latest canonical state in eth2.
func (k *KVStore) HeadState(ctx context.Context) (*pb.BeaconState, error) {
	return nil, nil
}

// SaveState stores a state to the db by the block root which triggered it.
func (k *KVStore) SaveState(ctx context.Context, state *pb.BeaconState, blockRoot [32]byte) error {
	return nil
}
