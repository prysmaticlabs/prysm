package kv

import (
	"context"
	"encoding/binary"
	"fmt"
	"github.com/prysmaticlabs/prysm/v5/encoding/bytesutil"
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
		return bkt.Put(bytesutil.Uint64ToBytesBigEndian(period), updateMarshalled)
	})
}

func (s *Store) LightClientUpdates(ctx context.Context, startPeriod, endPeriod uint64) ([]*ethpbv2.LightClientUpdateWithVersion, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.LightClientUpdates")
	defer span.End()

	if startPeriod > endPeriod {
		return nil, fmt.Errorf("start period %d is greater than end period %d", startPeriod, endPeriod)
	}

	updates := make([]*ethpbv2.LightClientUpdateWithVersion, 0, endPeriod-startPeriod+1)
	err := s.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(lightClientUpdatesBucket)
		c := bkt.Cursor()

		// first available period in DB
		firstPeriodInDB, _ := c.First()
		if firstPeriodInDB == nil {
			updates = nil
			return fmt.Errorf("no light client updates in the database")
		}
		// last available period in DB
		lastPeriodInDB, _ := c.Last()

		// first available period in the requested range
		firstPeriodInRange, _ := c.Seek(bytesutil.Uint64ToBytesBigEndian(startPeriod))
		if firstPeriodInRange == nil {
			updates = nil
			return fmt.Errorf("no light client updates in this range")
		}
		// last available period in the requested range
		lastPeriodInRange, _ := c.Seek(bytesutil.Uint64ToBytesBigEndian(endPeriod))
		if lastPeriodInRange == nil {
			lastPeriodInRange = lastPeriodInDB
		} else if binary.BigEndian.Uint64(lastPeriodInRange) > endPeriod {
			lastPeriodInRange, _ = c.Prev()
		}

		// check for missing periods at the beginning of the range
		if binary.BigEndian.Uint64(firstPeriodInRange) > startPeriod && startPeriod > binary.BigEndian.Uint64(firstPeriodInDB) {
			updates = nil
			return fmt.Errorf("missing light client updates for some periods in this range")
		}

		// check for missing periods at the end of the range
		if binary.BigEndian.Uint64(lastPeriodInRange) < endPeriod && endPeriod < binary.BigEndian.Uint64(lastPeriodInDB) {
			updates = nil
			return fmt.Errorf("missing light client updates for some periods in this range")
		}

		// check for missing periods in the middle of the range - need to go through all periods in the range
		expectedStartPeriod := binary.BigEndian.Uint64(firstPeriodInRange)
		expectedEndPeriod := binary.BigEndian.Uint64(lastPeriodInRange)

		expectedPeriod := expectedStartPeriod
		for k, v := c.Seek(bytesutil.Uint64ToBytesBigEndian(startPeriod)); k != nil && binary.BigEndian.Uint64(k) <= endPeriod; k, v = c.Next() {
			// check if there was a gap by matching the current period with the expected period
			currentPeriod := binary.BigEndian.Uint64(k)
			if currentPeriod != expectedPeriod {
				updates = nil
				return fmt.Errorf("missing light client updates for some periods in this range")
			}

			var update ethpbv2.LightClientUpdateWithVersion
			if err := decode(ctx, v, &update); err != nil {
				return err
			}
			expectedPeriod++
			updates = append(updates, &update)
		}
		// check if the last period in the range is the expected end period and if all updates were found
		if expectedPeriod-1 != expectedEndPeriod || len(updates) != int(expectedEndPeriod-expectedStartPeriod+1) {
			updates = nil
			return fmt.Errorf("missing light client updates for some periods in this range")
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
		updateBytes := bkt.Get(bytesutil.Uint64ToBytesBigEndian(period))
		if updateBytes == nil {
			return nil
		}
		return decode(ctx, updateBytes, &update)
	})
	return &update, err
}
