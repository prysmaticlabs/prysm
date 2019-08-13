package kv

import (
	"context"

	"github.com/boltdb/bolt"
	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

// State returns the saved state using block's signing root,
// this particular block was used to generate the state.
func (k *Store) State(ctx context.Context, blockRoot [32]byte) (*pb.BeaconState, error) {
	var s *pb.BeaconState
	err := k.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(stateBucket)
		enc := bucket.Get(blockRoot[:])
		if enc == nil {
			return nil
		}

		var err error
		s, err = createState(enc)

		return err
	})
	return s, err
}

// HeadState returns the latest canonical state in beacon chain.
func (k *Store) HeadState(ctx context.Context) (*pb.BeaconState, error) {
	var s *pb.BeaconState
	err := k.db.View(func(tx *bolt.Tx) error {
		// Retrieve head block's signing root from blocks bucket,
		// to look up what the head state is.
		bucket := tx.Bucket(blocksBucket)
		headBlkRoot := bucket.Get(headBlockRootKey)

		bucket = tx.Bucket(stateBucket)
		enc := bucket.Get(headBlkRoot)
		if enc == nil {
			return nil
		}

		var err error
		s, err = createState(enc)

		return err
	})
	return s, err
}

// SaveState stores a state to the db using block's signing root which was used to generate the state.
func (k *Store) SaveState(ctx context.Context, state *pb.BeaconState, blockRoot [32]byte) error {
	enc, err := proto.Marshal(state)
	if err != nil {
		return err
	}

	return k.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(stateBucket)
		return bucket.Put(blockRoot[:], enc)
	})
}

// creates state from marshaled proto state bytes.
func createState(enc []byte) (*pb.BeaconState, error) {
	protoState := &pb.BeaconState{}
	err := proto.Unmarshal(enc, protoState)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal encoding")
	}
	return protoState, nil
}
