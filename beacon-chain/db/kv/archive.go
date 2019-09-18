package kv

import (
	"context"

	"github.com/boltdb/bolt"
	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"go.opencensus.io/trace"

	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
)

// ArchivedActiveValidatorChanges retrieval by epoch.
func (k *Store) ArchivedActiveValidatorChanges(ctx context.Context, epoch uint64) (*ethpb.ArchivedActiveSetChanges, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.ArchivedActiveValidatorChanges")
	defer span.End()

	buf := uint64ToBytes(epoch)
	var target *ethpb.ArchivedActiveSetChanges
	err := k.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(archivedValidatorSetChangesBucket)
		enc := bkt.Get(buf)
		if enc == nil {
			return nil
		}
		target = &ethpb.ArchivedActiveSetChanges{}
		return proto.Unmarshal(enc, target)
	})
	return target, err
}

// SaveArchivedActiveValidatorChanges by epoch.
func (k *Store) SaveArchivedActiveValidatorChanges(ctx context.Context, epoch uint64, changes *ethpb.ArchivedActiveSetChanges) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.SaveArchivedActiveValidatorChanges")
	defer span.End()
	buf := uint64ToBytes(epoch)
	enc, err := proto.Marshal(changes)
	if err != nil {
		return err
	}
	return k.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(archivedValidatorSetChangesBucket)
		return bucket.Put(buf, enc)
	})
}

// ArchivedCommitteeInfo retrieval by epoch.
func (k *Store) ArchivedCommitteeInfo(ctx context.Context, epoch uint64) (*ethpb.ArchivedCommitteeInfo, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.ArchivedCommitteeInfo")
	defer span.End()

	buf := uint64ToBytes(epoch)
	var target *ethpb.ArchivedCommitteeInfo
	err := k.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(archivedValidatorSetChangesBucket)
		enc := bkt.Get(buf)
		if enc == nil {
			return nil
		}
		target = &ethpb.ArchivedCommitteeInfo{}
		return proto.Unmarshal(enc, target)
	})
	return target, err
}

// SaveArchivedCommitteeInfo by epoch.
func (k *Store) SaveArchivedCommitteeInfo(ctx context.Context, epoch uint64, info *ethpb.ArchivedCommitteeInfo) error {
	return errors.New("unimplemented")
}

// ArchivedBalances retrieval by epoch.
func (k *Store) ArchivedBalances(ctx context.Context, epoch uint64) ([]uint64, error) {
	return nil, errors.New("unimplemented")
}

// SaveArchivedBalances by epoch.
func (k *Store) SaveArchivedBalances(ctx context.Context, epoch uint64, balances []uint64) error {
	return errors.New("unimplemented")
}

// ArchivedActiveIndices retrieval by epoch.
func (k *Store) ArchivedActiveIndices(ctx context.Context, epoch uint64) ([]uint64, error) {
	return nil, errors.New("unimplemented")
}

// SaveArchivedActiveIndices by epoch.
func (k *Store) SaveArchivedActiveIndices(ctx context.Context, epoch uint64, indices []uint64) error {
	return errors.New("unimplemented")
}

// ArchivedValidatorParticipation retrieval by epoch.
func (k *Store) ArchivedValidatorParticipation(ctx context.Context, epoch uint64) (*ethpb.ValidatorParticipation, error) {
	return nil, errors.New("unimplemented")
}

// SaveArchivedValidatorParticipation by epoch.
func (k *Store) SaveArchivedValidatorParticipation(ctx context.Context, epoch uint64, part *ethpb.ValidatorParticipation) error {
	return errors.New("unimplemented")
}
