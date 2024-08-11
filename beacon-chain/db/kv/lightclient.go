package kv

import (
	"context"
	"encoding/binary"
	ethpbv2 "github.com/prysmaticlabs/prysm/v5/proto/eth/v2"
	bolt "go.etcd.io/bbolt"
	"go.opencensus.io/trace"
)

func (s *Store) SaveLightClientUpdate(ctx context.Context, period uint64, update *ethpbv2.LightClientUpdateWithVersion) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.saveLightClientUpdate")
	defer span.End()

	return s.db.Update(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(lightClientUpdatesBucket)
		updateMarshalled, err := encode(ctx, update)
		if err != nil {
			return err
		}
		return bkt.Put(uint64ToBytes(period), updateMarshalled)
	})
}

func (s *Store) LightClientUpdates(ctx context.Context, startPeriod, endPeriod uint64) ([]*ethpbv2.LightClientUpdateWithVersion, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.LightClientUpdates")
	defer span.End()

	var updates []*ethpbv2.LightClientUpdateWithVersion
	err := s.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(lightClientUpdatesBucket)
		c := bkt.Cursor()
		for k, v := c.Seek(uint64ToBytes(startPeriod)); k != nil && binary.BigEndian.Uint64(k) <= endPeriod; k, v = c.Next() {
			var update ethpbv2.LightClientUpdateWithVersion
			if err := decode(ctx, v, &update); err != nil {
				return err
			}
			updates = append(updates, &update)
		}
		return nil
	})
	return updates, err
}

func (s *Store) LightClientUpdate(ctx context.Context, period uint64) (*ethpbv2.LightClientUpdateWithVersion, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.LightClientUpdate")
	defer span.End()

	var update ethpbv2.LightClientUpdateWithVersion
	err := s.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(lightClientUpdatesBucket)
		updateBytes := bkt.Get(uint64ToBytes(period))
		if updateBytes == nil {
			return nil
		}
		return decode(ctx, updateBytes, &update)
	})
	return &update, err
}

func uint64ToBytes(period uint64) []byte {
	var b [8]byte
	binary.BigEndian.PutUint64(b[:], period)
	return b[:]
}
