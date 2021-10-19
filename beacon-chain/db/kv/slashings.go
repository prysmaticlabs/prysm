package kv

import (
	"context"

	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	bolt "go.etcd.io/bbolt"
	"go.opencensus.io/trace"
)

// ProposerSlashing retrieval by slashing root.
func (s *Store) ProposerSlashing(ctx context.Context, slashingRoot [32]byte) (*ethpb.ProposerSlashing, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.ProposerSlashing")
	defer span.End()
	enc, err := s.proposerSlashingBytes(ctx, slashingRoot)
	if err != nil {
		return nil, err
	}
	if len(enc) == 0 {
		return nil, nil
	}
	proposerSlashing := &ethpb.ProposerSlashing{}
	if err := decode(ctx, enc, proposerSlashing); err != nil {
		return nil, err
	}
	return proposerSlashing, nil
}

// HasProposerSlashing verifies if a slashing is stored in the db.
func (s *Store) HasProposerSlashing(ctx context.Context, slashingRoot [32]byte) bool {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.HasProposerSlashing")
	defer span.End()
	enc, err := s.proposerSlashingBytes(ctx, slashingRoot)
	if err != nil {
		panic(err)
	}
	return len(enc) > 0
}

// SaveProposerSlashing to the db by its hash tree root.
func (s *Store) SaveProposerSlashing(ctx context.Context, slashing *ethpb.ProposerSlashing) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.SaveProposerSlashing")
	defer span.End()
	slashingRoot, err := slashing.HashTreeRoot()
	if err != nil {
		return err
	}
	enc, err := encode(ctx, slashing)
	if err != nil {
		return err
	}
	return s.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(proposerSlashingsBucket)
		return bucket.Put(slashingRoot[:], enc)
	})
}

func (s *Store) proposerSlashingBytes(ctx context.Context, slashingRoot [32]byte) ([]byte, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.proposerSlashingBytes")
	defer span.End()
	var dst []byte
	err := s.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(proposerSlashingsBucket)
		dst = bkt.Get(slashingRoot[:])
		return nil
	})
	return dst, err
}

// deleteProposerSlashing clears a proposer slashing from the db by its hash tree root.
func (s *Store) deleteProposerSlashing(ctx context.Context, slashingRoot [32]byte) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.deleteProposerSlashing")
	defer span.End()
	return s.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(proposerSlashingsBucket)
		return bucket.Delete(slashingRoot[:])
	})
}

// AttesterSlashing retrieval by hash tree root.
func (s *Store) AttesterSlashing(ctx context.Context, slashingRoot [32]byte) (*ethpb.AttesterSlashing, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.AttesterSlashing")
	defer span.End()
	enc, err := s.attesterSlashingBytes(ctx, slashingRoot)
	if err != nil {
		return nil, err
	}
	if len(enc) == 0 {
		return nil, nil
	}
	attSlashing := &ethpb.AttesterSlashing{}
	if err := decode(ctx, enc, attSlashing); err != nil {
		return nil, err
	}
	return attSlashing, nil
}

// HasAttesterSlashing verifies if a slashing is stored in the db.
func (s *Store) HasAttesterSlashing(ctx context.Context, slashingRoot [32]byte) bool {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.HasAttesterSlashing")
	defer span.End()
	enc, err := s.attesterSlashingBytes(ctx, slashingRoot)
	if err != nil {
		panic(err)
	}
	return len(enc) > 0
}

// SaveAttesterSlashing to the db by its hash tree root.
func (s *Store) SaveAttesterSlashing(ctx context.Context, slashing *ethpb.AttesterSlashing) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.SaveAttesterSlashing")
	defer span.End()
	slashingRoot, err := slashing.HashTreeRoot()
	if err != nil {
		return err
	}
	enc, err := encode(ctx, slashing)
	if err != nil {
		return err
	}
	return s.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(attesterSlashingsBucket)
		return bucket.Put(slashingRoot[:], enc)
	})
}

func (s *Store) attesterSlashingBytes(ctx context.Context, slashingRoot [32]byte) ([]byte, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.attesterSlashingBytes")
	defer span.End()
	var dst []byte
	err := s.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(attesterSlashingsBucket)
		dst = bkt.Get(slashingRoot[:])
		return nil
	})
	return dst, err
}

// deleteAttesterSlashing clears an attester slashing from the db by its hash tree root.
func (s *Store) deleteAttesterSlashing(ctx context.Context, slashingRoot [32]byte) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.deleteAttesterSlashing")
	defer span.End()
	return s.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(attesterSlashingsBucket)
		return bucket.Delete(slashingRoot[:])
	})
}
