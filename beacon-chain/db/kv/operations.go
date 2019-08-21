package kv

import (
	"context"

	"github.com/boltdb/bolt"
	"github.com/gogo/protobuf/proto"
	"github.com/prysmaticlabs/go-ssz"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"go.opencensus.io/trace"
)

func (k *Store) ProposerSlashing(ctx context.Context, slashingRoot [32]byte) (*ethpb.ProposerSlashing, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.ProposerSlashing")
	defer span.End()
	var slashing *ethpb.ProposerSlashing
	err := k.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(validatorsBucket)
		enc := bkt.Get(slashingRoot[:])
		if enc == nil {
			return nil
		}
		slashing = &ethpb.ProposerSlashing{}
		return proto.Unmarshal(enc, slashing)
	})
	return slashing, err
}

// HasProposerSlashing --
func (k *Store) HasProposerSlashing(ctx context.Context, slashingRoot [32]byte) bool {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.HasProposerSlashing")
	defer span.End()
	exists := false
	// #nosec G104. Always returns nil.
	k.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(validatorsBucket)
		exists = bkt.Get(slashingRoot[:]) != nil
		return nil
	})
	return exists
}

// SaveProposerSlashing --
func (k *Store) SaveProposerSlashing(ctx context.Context, slashing *ethpb.ProposerSlashing) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.SaveProposerSlashing")
	defer span.End()
	slashingRoot, err := ssz.HashTreeRoot(slashing)
	if err != nil {
		return err
	}
	enc, err := proto.Marshal(slashing)
	if err != nil {
		return err
	}
	return k.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(validatorsBucket)
		return bucket.Put(slashingRoot[:], enc)
	})
}
