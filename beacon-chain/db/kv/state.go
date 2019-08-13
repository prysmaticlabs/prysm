package kv

import (
	"context"

	"github.com/prysmaticlabs/prysm/beacon-chain/db/filters"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

// State retrieval by filter criteria.
// TODO(#3164): Implement.
func (k *Store) State(ctx context.Context, f *filters.QueryFilter) (*pb.BeaconState, error) {
	return nil, nil
}

// HeadState returns the latest canonical state in eth2.
// TODO(#3164): Implement.
func (k *Store) HeadState(ctx context.Context) (*pb.BeaconState, error) {
	return nil, nil
}

// SaveState stores a state to the db by the block root which triggered it.
// TODO(#3164): Implement.
func (k *Store) SaveState(ctx context.Context, state *pb.BeaconState, blockRoot [32]byte) error {
	return nil
}
