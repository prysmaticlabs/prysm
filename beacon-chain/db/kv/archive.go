package kv

import (
	"context"

	"github.com/boltdb/bolt"
	"github.com/gogo/protobuf/proto"
	"github.com/prysmaticlabs/go-ssz"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"go.opencensus.io/trace"
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
		bkt := tx.Bucket(archivedCommitteeInfoBucket)
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
	ctx, span := trace.StartSpan(ctx, "BeaconDB.SaveArchivedCommitteeInfo")
	defer span.End()
	buf := uint64ToBytes(epoch)
	enc, err := proto.Marshal(info)
	if err != nil {
		return err
	}
	return k.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(archivedCommitteeInfoBucket)
		return bucket.Put(buf, enc)
	})
}

// ArchivedBalances retrieval by epoch.
func (k *Store) ArchivedBalances(ctx context.Context, epoch uint64) ([]uint64, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.ArchivedBalances")
	defer span.End()

	buf := uint64ToBytes(epoch)
	var target []uint64
	err := k.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(archivedBalancesBucket)
		enc := bkt.Get(buf)
		if enc == nil {
			return nil
		}
		target = make([]uint64, 0)
		return ssz.Unmarshal(enc, &target)
	})
	return target, err
}

// SaveArchivedBalances by epoch.
func (k *Store) SaveArchivedBalances(ctx context.Context, epoch uint64, balances []uint64) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.SaveArchivedBalances")
	defer span.End()
	buf := uint64ToBytes(epoch)
	enc, err := ssz.Marshal(balances)
	if err != nil {
		return err
	}
	return k.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(archivedBalancesBucket)
		return bucket.Put(buf, enc)
	})
}

// ArchivedActiveIndices retrieval by epoch.
func (k *Store) ArchivedActiveIndices(ctx context.Context, epoch uint64) ([]uint64, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.ArchivedActiveIndices")
	defer span.End()

	buf := uint64ToBytes(epoch)
	var target []uint64
	err := k.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(archivedActiveIndicesBucket)
		enc := bkt.Get(buf)
		if enc == nil {
			return nil
		}
		target = make([]uint64, 0)
		return ssz.Unmarshal(enc, &target)
	})
	return target, err
}

// SaveArchivedActiveIndices by epoch.
func (k *Store) SaveArchivedActiveIndices(ctx context.Context, epoch uint64, indices []uint64) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.SaveArchivedActiveIndices")
	defer span.End()
	buf := uint64ToBytes(epoch)
	enc, err := ssz.Marshal(indices)
	if err != nil {
		return err
	}
	return k.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(archivedActiveIndicesBucket)
		return bucket.Put(buf, enc)
	})
}

// ArchivedValidatorParticipation retrieval by epoch.
func (k *Store) ArchivedValidatorParticipation(ctx context.Context, epoch uint64) (*ethpb.ValidatorParticipation, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.ArchivedValidatorParticipation")
	defer span.End()

	buf := uint64ToBytes(epoch)
	var target *ethpb.ValidatorParticipation
	err := k.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(archivedValidatorParticipationBucket)
		enc := bkt.Get(buf)
		if enc == nil {
			return nil
		}
		target = &ethpb.ValidatorParticipation{}
		return proto.Unmarshal(enc, target)
	})
	return target, err
}

// SaveArchivedValidatorParticipation by epoch.
func (k *Store) SaveArchivedValidatorParticipation(ctx context.Context, epoch uint64, part *ethpb.ValidatorParticipation) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.SaveArchivedValidatorParticipation")
	defer span.End()
	buf := uint64ToBytes(epoch)
	enc, err := proto.Marshal(part)
	if err != nil {
		return err
	}
	return k.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(archivedValidatorParticipationBucket)
		return bucket.Put(buf, enc)
	})
}
