package kv

import (
	"context"

	ethpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	bolt "go.etcd.io/bbolt"
	"go.opencensus.io/trace"
)

// SaveSignedExecutionPayloadEnvelopeBlind saves a signed execution payload envelope blind in the database.
func (s *Store) SaveSignedExecutionPayloadEnvelopeBlind(ctx context.Context, env *ethpb.SignedExecutionPayloadEnvelopeBlind) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.SaveSignedExecutionPayloadEnvelopeBlind")
	defer span.End()

	enc, err := encode(ctx, env)
	if err != nil {
		return err
	}

	r := env.Message.BeaconBlockRoot
	err = s.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(executionPayloadEnvelopeBucket)
		return bucket.Put(r, enc)
	})

	return err
}

// SignedExecutionPayloadEnvelopeBlind retrieves a signed execution payload envelope blind from the database.
func (s *Store) SignedExecutionPayloadEnvelopeBlind(ctx context.Context, blockRoot []byte) (*ethpb.SignedExecutionPayloadEnvelopeBlind, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.SignedExecutionPayloadEnvelopeBlind")
	defer span.End()

	env := &ethpb.SignedExecutionPayloadEnvelopeBlind{}
	err := s.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(executionPayloadEnvelopeBucket)
		enc := bkt.Get(blockRoot)
		if enc == nil {
			return ErrNotFound
		}
		return decode(ctx, enc, env)
	})
	return env, err
}
