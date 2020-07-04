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
	enc, err := kv.proposerSlashingBytes(ctx, slashingRoot)
	if err != nil {
		return nil, err
	}
	proposerSlashing := &ethpb.ProposerSlashing{}
	if len(enc) == 0 {
		return nil, nil
	}
	if err := decode(ctx, enc, proposerSlashing); err != nil {
		return nil, err
	}
	return proposerSlashing, nil
}

// HasProposerSlashing verifies if a slashing is stored in the db.
func (kv *Store) HasProposerSlashing(ctx context.Context, slashingRoot [32]byte) bool {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.HasProposerSlashing")
	defer span.End()
	enc, err := kv.proposerSlashingBytes(ctx, slashingRoot)
	if err != nil {
		panic(err)
	}
	return len(enc) > 0
}

// SaveProposerSlashing to the db by its hash tree root.
func (kv *Store) SaveProposerSlashing(ctx context.Context, slashing *ethpb.ProposerSlashing) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.SaveProposerSlashing")
	defer span.End()
	slashingRoot, err := ssz.HashTreeRoot(slashing)
	if err != nil {
		return err
	}
	enc, err := encode(ctx, slashing)
	if err != nil {
		return err
	}
	return kv.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(proposerSlashingsBucket)
		return bucket.Put(slashingRoot[:], enc)
	})
}

func (kv *Store) proposerSlashingBytes(ctx context.Context, slashingRoot [32]byte) ([]byte, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.proposerSlashingBytes")
	defer span.End()
	var dst []byte
	err := kv.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(proposerSlashingsBucket)
		dst = bkt.Get(slashingRoot[:])
		return nil
	})
	return dst, err
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
	enc, err := kv.attesterSlashingBytes(ctx, slashingRoot)
	if err != nil {
		return nil, err
	}
	attSlashing := &ethpb.AttesterSlashing{}
	if len(enc) == 0 {
		return nil, nil
	}
	if err := decode(ctx, enc, attSlashing); err != nil {
		return nil, err
	}
	return attSlashing, nil
}

// HasAttesterSlashing verifies if a slashing is stored in the db.
func (kv *Store) HasAttesterSlashing(ctx context.Context, slashingRoot [32]byte) bool {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.HasAttesterSlashing")
	defer span.End()
	enc, err := kv.attesterSlashingBytes(ctx, slashingRoot)
	if err != nil {
		panic(err)
	}
	return len(enc) > 0
}

// SaveAttesterSlashing to the db by its hash tree root.
func (kv *Store) SaveAttesterSlashing(ctx context.Context, slashing *ethpb.AttesterSlashing) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.SaveAttesterSlashing")
	defer span.End()
	slashingRoot, err := ssz.HashTreeRoot(slashing)
	if err != nil {
		return err
	}
	enc, err := encode(ctx, slashing)
	if err != nil {
		return err
	}
	return kv.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(attesterSlashingsBucket)
		return bucket.Put(slashingRoot[:], enc)
	})
}

func (kv *Store) attesterSlashingBytes(ctx context.Context, slashingRoot [32]byte) ([]byte, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.attesterSlashingBytes")
	defer span.End()
	var dst []byte
	err := kv.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(attesterSlashingsBucket)
		dst = bkt.Get(slashingRoot[:])
		return nil
	})
	return dst, err
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
