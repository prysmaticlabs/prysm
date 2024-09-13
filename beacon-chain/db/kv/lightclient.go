package kv

import (
	"context"
	"encoding/binary"
	"fmt"

	"github.com/prysmaticlabs/prysm/v5/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/v5/monitoring/tracing/trace"
	ethpbv2 "github.com/prysmaticlabs/prysm/v5/proto/eth/v2"
	bolt "go.etcd.io/bbolt"
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
		return bkt.Put(bytesutil.Uint64ToBytesBigEndian(period), updateMarshalled)
	})
}

func (s *Store) LightClientUpdates(ctx context.Context, startPeriod, endPeriod uint64) (map[uint64]*ethpbv2.LightClientUpdateWithVersion, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.LightClientUpdates")
	defer span.End()

	if startPeriod > endPeriod {
		return nil, fmt.Errorf("start period %d is greater than end period %d", startPeriod, endPeriod)
	}

	updates := make(map[uint64]*ethpbv2.LightClientUpdateWithVersion)
	err := s.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(lightClientUpdatesBucket)
		c := bkt.Cursor()

		firstPeriodInDb, _ := c.First()
		if firstPeriodInDb == nil {
			return nil
		}

		for k, v := c.Seek(bytesutil.Uint64ToBytesBigEndian(startPeriod)); k != nil && binary.BigEndian.Uint64(k) <= endPeriod; k, v = c.Next() {
			currentPeriod := binary.BigEndian.Uint64(k)

			var update ethpbv2.LightClientUpdateWithVersion
			if err := decode(ctx, v, &update); err != nil {
				return err
			}
			updates[currentPeriod] = &update
		}

		return nil
	})

	if err != nil {
		return nil, err
	}
	return updates, err
}

func (s *Store) LightClientUpdate(ctx context.Context, period uint64) (*ethpbv2.LightClientUpdateWithVersion, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.LightClientUpdate")
	defer span.End()

	var update ethpbv2.LightClientUpdateWithVersion
	err := s.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(lightClientUpdatesBucket)
		updateBytes := bkt.Get(bytesutil.Uint64ToBytesBigEndian(period))
		if updateBytes == nil {
			return nil
		}
		return decode(ctx, updateBytes, &update)
	})
	return &update, err
}
