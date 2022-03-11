package kv

import (
	"context"
	"encoding/binary"

	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	bolt "go.etcd.io/bbolt"
	"go.opencensus.io/trace"
)

func (s *Store) LastValidatedCheckpoint(ctx context.Context) ([32]byte, types.Slot, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.LastValidatedCheckpoint")
	defer span.End()

	var lastChkPoint [32]byte
	var slot types.Slot
	err := s.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(lastValidatedCheckpoint)
		val := bkt.Get([]byte("lastChkPoint"))
		if len(val) != 40 { // 32 for checkpoint root + 8 bytes for slot
			return errInvalidCheckpointSize
		}
		lastChkPoint = bytesutil.ToBytes32(val[:32])
		slot = types.Slot(binary.LittleEndian.Uint64(val[32:]))
		return nil
	})
	return lastChkPoint, slot, err
}

func (s *Store) SaveLastValidatedCheckpoint(ctx context.Context, checkPoint [32]byte, slot types.Slot) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.saveLastValidatedCheckpoint")
	defer span.End()

	updateErr := s.db.Update(func(tx *bolt.Tx) error {
		value := make([]byte, 40)
		copy(value[:32], checkPoint[:])
		binary.LittleEndian.PutUint64(value[32:], uint64(slot))
		bkt := tx.Bucket(lastValidatedCheckpoint)
		err := bkt.Put([]byte("lastChkPoint"), value)
		return err
	})

	return updateErr
}
