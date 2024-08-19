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

func getFirstAndLastPeriodInDB(c *bolt.Cursor) (uint64, uint64, error) {
	firstPeriod, _ := c.First()
	lastPeriod, _ := c.Last()
	if firstPeriod == nil || lastPeriod == nil {
		return 0, 0, fmt.Errorf("no light client updates in the database")
	}
	return binary.BigEndian.Uint64(firstPeriod), binary.BigEndian.Uint64(lastPeriod), nil
}

func getFirstAndLastPeriodInRequestedRange(c *bolt.Cursor, startPeriod, endPeriod uint64) (uint64, uint64, error) {
	firstPeriodInRange, _ := c.Seek(bytesutil.Uint64ToBytesBigEndian(startPeriod))
	if firstPeriodInRange == nil {
		return 0, 0, fmt.Errorf("no light client updates in this range")
	}
	lastPeriodInRange, _ := c.Seek(bytesutil.Uint64ToBytesBigEndian(endPeriod))
	if lastPeriodInRange == nil {
		lastPeriodInRange, _ = c.Last()
	} else if binary.BigEndian.Uint64(lastPeriodInRange) > endPeriod {
		lastPeriodInRange, _ = c.Prev()
	}
	return binary.BigEndian.Uint64(firstPeriodInRange), binary.BigEndian.Uint64(lastPeriodInRange), nil
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

		// get first and last available periods in the database
		firstPeriodInDB, lastPeriodInDB, err := getFirstAndLastPeriodInDB(c)
		if err != nil {
			return err
		}

		// get first and last available periods in the requested range
		firstPeriodInRange, lastPeriodInRange, err := getFirstAndLastPeriodInRequestedRange(c, startPeriod, endPeriod)
		if err != nil {
			return err
		}

		// check for missing periods at the beginning of the range
		if firstPeriodInRange > startPeriod && startPeriod > firstPeriodInDB {
			return fmt.Errorf("missing light client updates for some periods in this range")
		}

		// check for missing periods at the end of the range
		if lastPeriodInRange < endPeriod && endPeriod < lastPeriodInDB {
			return fmt.Errorf("missing light client updates for some periods in this range")
		}

		// check for missing periods in the middle of the range - need to go through all periods in the range
		expectedStartPeriod := firstPeriodInRange
		expectedEndPeriod := lastPeriodInRange

		expectedPeriod := expectedStartPeriod
		for k, v := c.Seek(bytesutil.Uint64ToBytesBigEndian(startPeriod)); k != nil && binary.BigEndian.Uint64(k) <= endPeriod; k, v = c.Next() {
			// check if there was a gap by matching the current period with the expected period
			currentPeriod := binary.BigEndian.Uint64(k)
			if currentPeriod != expectedPeriod {
				return fmt.Errorf("missing light client updates for some periods in this range")
			}

			var update ethpbv2.LightClientUpdateWithVersion
			if err := decode(ctx, v, &update); err != nil {
				return err
			}
			updates = append(updates, &update)
			expectedPeriod++
		}

		// check if the last period in the range is the expected end period and if all updates were found
		if expectedPeriod-1 != expectedEndPeriod || len(updates) != int(expectedEndPeriod-expectedStartPeriod+1) {
			return fmt.Errorf("missing light client updates for some periods in this range")
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
