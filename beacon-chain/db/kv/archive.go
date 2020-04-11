package kv

import (
	"context"
	"encoding/binary"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	bolt "go.etcd.io/bbolt"
	"go.opencensus.io/trace"
)

// ArchivedActiveValidatorChanges retrieval by epoch.
func (k *Store) ArchivedActiveValidatorChanges(ctx context.Context, epoch uint64) (*pb.ArchivedActiveSetChanges, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.ArchivedActiveValidatorChanges")
	defer span.End()

	buf := bytesutil.Uint64ToBytes(epoch)
	var target *pb.ArchivedActiveSetChanges
	err := k.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(archivedValidatorSetChangesBucket)
		enc := bkt.Get(buf)
		if enc == nil {
			return nil
		}
		target = &pb.ArchivedActiveSetChanges{}
		return decode(enc, target)
	})
	return target, err
}

// SaveArchivedActiveValidatorChanges by epoch.
func (k *Store) SaveArchivedActiveValidatorChanges(ctx context.Context, epoch uint64, changes *pb.ArchivedActiveSetChanges) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.SaveArchivedActiveValidatorChanges")
	defer span.End()
	buf := bytesutil.Uint64ToBytes(epoch)
	enc, err := encode(changes)
	if err != nil {
		return err
	}
	return k.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(archivedValidatorSetChangesBucket)
		return bucket.Put(buf, enc)
	})
}

// ArchivedCommitteeInfo retrieval by epoch.
func (k *Store) ArchivedCommitteeInfo(ctx context.Context, epoch uint64) (*pb.ArchivedCommitteeInfo, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.ArchivedCommitteeInfo")
	defer span.End()

	buf := bytesutil.Uint64ToBytes(epoch)
	var target *pb.ArchivedCommitteeInfo
	err := k.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(archivedCommitteeInfoBucket)
		enc := bkt.Get(buf)
		if enc == nil {
			return nil
		}
		target = &pb.ArchivedCommitteeInfo{}
		return decode(enc, target)
	})
	return target, err
}

// SaveArchivedCommitteeInfo by epoch.
func (k *Store) SaveArchivedCommitteeInfo(ctx context.Context, epoch uint64, info *pb.ArchivedCommitteeInfo) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.SaveArchivedCommitteeInfo")
	defer span.End()
	buf := bytesutil.Uint64ToBytes(epoch)
	enc, err := encode(info)
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

	buf := bytesutil.Uint64ToBytes(epoch)
	var target []uint64
	err := k.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(archivedBalancesBucket)
		enc := bkt.Get(buf)
		if enc == nil {
			return nil
		}
		target = unmarshalBalances(enc)
		return nil
	})
	return target, err
}

// SaveArchivedBalances by epoch.
func (k *Store) SaveArchivedBalances(ctx context.Context, epoch uint64, balances []uint64) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.SaveArchivedBalances")
	defer span.End()
	buf := bytesutil.Uint64ToBytes(epoch)
	enc := marshalBalances(balances)
	return k.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(archivedBalancesBucket)
		return bucket.Put(buf, enc)
	})
}

// ArchivedValidatorParticipation retrieval by epoch.
func (k *Store) ArchivedValidatorParticipation(ctx context.Context, epoch uint64) (*ethpb.ValidatorParticipation, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.ArchivedValidatorParticipation")
	defer span.End()

	buf := bytesutil.Uint64ToBytes(epoch)
	var target *ethpb.ValidatorParticipation
	err := k.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(archivedValidatorParticipationBucket)
		enc := bkt.Get(buf)
		if enc == nil {
			return nil
		}
		target = &ethpb.ValidatorParticipation{}
		return decode(enc, target)
	})
	return target, err
}

// SaveArchivedValidatorParticipation by epoch.
func (k *Store) SaveArchivedValidatorParticipation(ctx context.Context, epoch uint64, part *ethpb.ValidatorParticipation) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.SaveArchivedValidatorParticipation")
	defer span.End()
	buf := bytesutil.Uint64ToBytes(epoch)
	enc, err := encode(part)
	if err != nil {
		return err
	}
	return k.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(archivedValidatorParticipationBucket)
		return bucket.Put(buf, enc)
	})
}

func marshalBalances(bals []uint64) []byte {
	res := make([]byte, len(bals)*8)
	offset := 0
	for i := 0; i < len(bals); i++ {
		binary.LittleEndian.PutUint64(res[offset:offset+8], bals[i])
		offset += 8
	}
	return res
}

func unmarshalBalances(bals []byte) []uint64 {
	numItems := len(bals) / 8
	res := make([]uint64, numItems)
	offset := 0
	for i := 0; i < numItems; i++ {
		res[i] = binary.LittleEndian.Uint64(bals[offset : offset+8])
		offset += 8
	}
	return res
}
