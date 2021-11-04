package kv

import (
	"context"

	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	bolt "go.etcd.io/bbolt"
	"go.opencensus.io/trace"
)

// VoluntaryExit retrieval by signing root.
func (s *Store) VoluntaryExit(ctx context.Context, exitRoot [32]byte) (*ethpb.VoluntaryExit, error) {
	ctx, span := trace.StartSpan(ctx, "beaconDB.VoluntaryExit")
	defer span.End()
	enc, err := s.voluntaryExitBytes(ctx, exitRoot)
	if err != nil {
		return nil, err
	}
	if len(enc) == 0 {
		return nil, nil
	}
	exit := &ethpb.VoluntaryExit{}
	if err := decode(ctx, enc, exit); err != nil {
		return nil, err
	}
	return exit, nil
}

// HasVoluntaryExit verifies if a voluntary exit is stored in the db by its signing root.
func (s *Store) HasVoluntaryExit(ctx context.Context, exitRoot [32]byte) bool {
	ctx, span := trace.StartSpan(ctx, "beaconDB.HasVoluntaryExit")
	defer span.End()
	enc, err := s.voluntaryExitBytes(ctx, exitRoot)
	if err != nil {
		panic(err)
	}
	return len(enc) > 0
}

// SaveVoluntaryExit to the db by its signing root.
func (s *Store) SaveVoluntaryExit(ctx context.Context, exit *ethpb.VoluntaryExit) error {
	ctx, span := trace.StartSpan(ctx, "beaconDB.SaveVoluntaryExit")
	defer span.End()
	exitRoot, err := exit.HashTreeRoot()
	if err != nil {
		return err
	}
	enc, err := encode(ctx, exit)
	if err != nil {
		return err
	}
	return s.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(voluntaryExitsBucket)
		return bucket.Put(exitRoot[:], enc)
	})
}

func (s *Store) voluntaryExitBytes(ctx context.Context, exitRoot [32]byte) ([]byte, error) {
	ctx, span := trace.StartSpan(ctx, "beaconDB.voluntaryExitBytes")
	defer span.End()
	var dst []byte
	err := s.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(voluntaryExitsBucket)
		dst = bkt.Get(exitRoot[:])
		return nil
	})
	return dst, err
}

// deleteVoluntaryExit clears a voluntary exit from the db by its signing root.
func (s *Store) deleteVoluntaryExit(ctx context.Context, exitRoot [32]byte) error {
	ctx, span := trace.StartSpan(ctx, "beaconDB.deleteVoluntaryExit")
	defer span.End()
	return s.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(voluntaryExitsBucket)
		return bucket.Delete(exitRoot[:])
	})
}
