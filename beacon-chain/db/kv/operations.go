package kv

import (
	"context"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-ssz"
	bolt "go.etcd.io/bbolt"
	"go.opencensus.io/trace"
)

// VoluntaryExit retrieval by signing root.
func (kv *Store) VoluntaryExit(ctx context.Context, exitRoot [32]byte) (*ethpb.VoluntaryExit, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.VoluntaryExit")
	defer span.End()
	enc, err := kv.voluntaryExitBytes(ctx, exitRoot)
	if err != nil {
		return nil, err
	}
	exit := &ethpb.VoluntaryExit{}
	return exit, decode(ctx, enc, exit)
}

// HasVoluntaryExit verifies if a voluntary exit is stored in the db by its signing root.
func (kv *Store) HasVoluntaryExit(ctx context.Context, exitRoot [32]byte) bool {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.HasVoluntaryExit")
	defer span.End()
	enc, err := kv.voluntaryExitBytes(ctx, exitRoot)
	if err != nil {
		panic(err)
	}
	return len(enc) > 0
}

// SaveVoluntaryExit to the db by its signing root.
func (kv *Store) SaveVoluntaryExit(ctx context.Context, exit *ethpb.VoluntaryExit) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.SaveVoluntaryExit")
	defer span.End()
	exitRoot, err := ssz.HashTreeRoot(exit)
	if err != nil {
		return err
	}
	enc, err := encode(ctx, exit)
	if err != nil {
		return err
	}
	return kv.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(voluntaryExitsBucket)
		return bucket.Put(exitRoot[:], enc)
	})
}

func (kv *Store) voluntaryExitBytes(ctx context.Context, exitRoot [32]byte) ([]byte, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.voluntaryExitBytes")
	defer span.End()
	var dst []byte
	err := kv.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(voluntaryExitsBucket)
		dst = bkt.Get(exitRoot[:])
		return nil
	})
	return dst, err
}

// deleteVoluntaryExit clears a voluntary exit from the db by its signing root.
func (kv *Store) deleteVoluntaryExit(ctx context.Context, exitRoot [32]byte) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.deleteVoluntaryExit")
	defer span.End()
	return kv.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(voluntaryExitsBucket)
		return bucket.Delete(exitRoot[:])
	})
}
