package kv

import (
	"context"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-ssz"
	bolt "go.etcd.io/bbolt"
	"go.opencensus.io/trace"
)

// ProposerSlashing retrieval by slashing root.
func (kv *Store) ProposerSlashing(ctx context.Context, slashingRoot [32]byte) (*ethpb.ProposerSlashing, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.ProposerSlashing")
	defer span.End()
	var slashing *ethpb.ProposerSlashing
	err := kv.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(proposerSlashingsBucket)
		enc := bkt.Get(slashingRoot[:])
		if enc == nil {
			return nil
		}
		slashing = &ethpb.ProposerSlashing{}
		return decode(enc, slashing)
	})
	return slashing, err
}

// HasProposerSlashing verifies if a slashing is stored in the db.
func (kv *Store) HasProposerSlashing(ctx context.Context, slashingRoot [32]byte) bool {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.HasProposerSlashing")
	defer span.End()
	exists := false
	if err := kv.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(proposerSlashingsBucket)
		exists = bkt.Get(slashingRoot[:]) != nil
		return nil
	}); err != nil { // This view never returns an error, but we'll handle anyway for sanity.
		panic(err)
	}
	return exists
}

// SaveProposerSlashing to the db by its hash tree root.
func (kv *Store) SaveProposerSlashing(ctx context.Context, slashing *ethpb.ProposerSlashing) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.SaveProposerSlashing")
	defer span.End()
	slashingRoot, err := ssz.HashTreeRoot(slashing)
	if err != nil {
		return err
	}
	enc, err := encode(slashing)
	if err != nil {
		return err
	}
	return kv.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(proposerSlashingsBucket)
		return bucket.Put(slashingRoot[:], enc)
	})
}

// deleteProposerSlashing clears a proposer slashing from the db by its hash tree root.
func (kv *Store) deleteProposerSlashing(ctx context.Context, slashingRoot [32]byte) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.deleteProposerSlashing")
	defer span.End()
	return kv.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(proposerSlashingsBucket)
		return bucket.Delete(slashingRoot[:])
	})
}

// AttesterSlashing retrieval by hash tree root.
func (kv *Store) AttesterSlashing(ctx context.Context, slashingRoot [32]byte) (*ethpb.AttesterSlashing, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.AttesterSlashing")
	defer span.End()
	var slashing *ethpb.AttesterSlashing
	err := kv.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(attesterSlashingsBucket)
		enc := bkt.Get(slashingRoot[:])
		if enc == nil {
			return nil
		}
		slashing = &ethpb.AttesterSlashing{}
		return decode(enc, slashing)
	})
	return slashing, err
}

// HasAttesterSlashing verifies if a slashing is stored in the db.
func (kv *Store) HasAttesterSlashing(ctx context.Context, slashingRoot [32]byte) bool {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.HasAttesterSlashing")
	defer span.End()
	exists := false
	if err := kv.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(attesterSlashingsBucket)
		exists = bkt.Get(slashingRoot[:]) != nil
		return nil
	}); err != nil { // This view never returns an error, but we'll handle anyway for sanity.
		panic(err)
	}
	return exists
}

// SaveAttesterSlashing to the db by its hash tree root.
func (kv *Store) SaveAttesterSlashing(ctx context.Context, slashing *ethpb.AttesterSlashing) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.SaveAttesterSlashing")
	defer span.End()
	slashingRoot, err := ssz.HashTreeRoot(slashing)
	if err != nil {
		return err
	}
	enc, err := encode(slashing)
	if err != nil {
		return err
	}
	return kv.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(attesterSlashingsBucket)
		return bucket.Put(slashingRoot[:], enc)
	})
}

// deleteAttesterSlashing clears an attester slashing from the db by its hash tree root.
func (kv *Store) deleteAttesterSlashing(ctx context.Context, slashingRoot [32]byte) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.deleteAttesterSlashing")
	defer span.End()
	return kv.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(attesterSlashingsBucket)
		return bucket.Delete(slashingRoot[:])
	})
}
