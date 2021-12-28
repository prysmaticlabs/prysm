package kv

import (
	"bytes"
	"context"

	"github.com/pkg/errors"
	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/beacon-chain/db/filters"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/time/slots"
	bolt "go.etcd.io/bbolt"
	"go.opencensus.io/trace"
)

// SaveLightClientUpdate saves light client update to the database.
func (s *Store) SaveLightClientUpdate(ctx context.Context, update *ethpb.LightClientUpdate) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.SaveLightClientUpdate")
	defer span.End()

	enc, err := encode(ctx, update)
	if err != nil {
		return err
	}
	slot := bytesutil.SlotToBytesBigEndian(update.AttestedHeader.Slot)
	return s.db.Update(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(lightClientUpdateBucket)
		return bkt.Put(slot, enc)
	})
}

// SaveFinalizedLightClientUpdate saves latest finalized light client update to the database.
func (s *Store) SaveFinalizedLightClientUpdate(ctx context.Context, update *ethpb.LightClientUpdate) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.SaveFinalizedLightClientUpdate")
	defer span.End()

	enc, err := encode(ctx, update)
	if err != nil {
		return err
	}
	slot := bytesutil.SlotToBytesBigEndian(update.AttestedHeader.Slot)
	return s.db.Update(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(lightClientFinalizedUpdateBucket)
		return bkt.Put(slot, enc)
	})
}

// LightClientUpdates retrieves light client updates from the database using query filter.
func (s *Store) LightClientUpdates(ctx context.Context, f *filters.QueryFilter) ([]*ethpb.LightClientUpdate, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.LightClientUpdates")
	defer span.End()

	updates := make([]*ethpb.LightClientUpdate, 0)
	if err := s.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(lightClientUpdateBucket)
		keys, err := lightClientUpdateKeysByFilter(ctx, tx, f, tx.Bucket(lightClientUpdateBucket))
		if err != nil {
			return err
		}
		for _, key := range keys {
			enc := bkt.Get(key)
			if len(enc) == 0 {
				continue
			}
			update := &ethpb.LightClientUpdate{}
			if err := decode(ctx, enc, update); err != nil {
				continue
			}
			updates = append(updates, update)
		}

		bkt = tx.Bucket(lightClientFinalizedUpdateBucket)
		keys, err = lightClientUpdateKeysByFilter(ctx, tx, f, tx.Bucket(lightClientFinalizedUpdateBucket))
		if err != nil {
			return err
		}
		for _, key := range keys {
			enc := bkt.Get(key)
			if len(enc) == 0 {
				continue
			}
			update := &ethpb.LightClientUpdate{}
			if err := decode(ctx, enc, update); err != nil {
				continue
			}
			updates = append(updates, update)
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return updates, nil
}

// LatestLightClientUpdate retrieves the latest light client update from the database.
func (s *Store) LatestLightClientUpdate(ctx context.Context) (*ethpb.LightClientUpdate, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.LatestLightClientUpdate")
	defer span.End()

	var enc []byte
	if err := s.db.View(func(tx *bolt.Tx) error {
		_, enc = tx.Bucket(lightClientUpdateBucket).Cursor().Last()
		return nil
	}); err != nil {
		return nil, err
	}
	if len(enc) == 0 {
		return nil, errors.New("no latest light client update found")
	}

	update := &ethpb.LightClientUpdate{}
	if err := decode(ctx, enc, update); err != nil {
		return nil, err
	}
	return update, nil
}

// LatestFinalizedLightClientUpdate retrieves latest finalized light client update from the database.
func (s *Store) LatestFinalizedLightClientUpdate(ctx context.Context) (*ethpb.LightClientUpdate, error) {
	_, span := trace.StartSpan(ctx, "BeaconDB.LatestFinalizedLightClientUpdate")
	defer span.End()

	var enc []byte
	if err := s.db.View(func(tx *bolt.Tx) error {
		_, enc = tx.Bucket(lightClientFinalizedUpdateBucket).Cursor().Last()
		return nil
	}); err != nil {
		return nil, err
	}
	if len(enc) == 0 {
		return nil, errors.New("no latest finalized light client update found")
	}

	update := &ethpb.LightClientUpdate{}
	if err := decode(ctx, enc, update); err != nil {
		return nil, err
	}
	return update, nil
}

// DeleteLightClientUpdates deletes light client updates from the database using input slots.
func (s *Store) DeleteLightClientUpdates(ctx context.Context, slots []types.Slot) error {
	_, span := trace.StartSpan(ctx, "BeaconDB.DeleteLightClientUpdates")
	defer span.End()

	return s.db.Update(func(tx *bolt.Tx) error {
		for _, slot := range slots {
			slotBytes := bytesutil.SlotToBytesBigEndian(slot)
			bkt := tx.Bucket(lightClientUpdateBucket)
			if err := bkt.Delete(slotBytes); err != nil {
				continue
			}
		}
		return nil
	})
}

// DeleteLightClientFinalizedUpdates deletes light client finalized updates from the database using input slots.
func (s *Store) DeleteLightClientFinalizedUpdates(ctx context.Context, slots []types.Slot) error {
	_, span := trace.StartSpan(ctx, "BeaconDB.DeleteLightClientFinalizedUpdates")
	defer span.End()

	return s.db.Update(func(tx *bolt.Tx) error {
		for _, slot := range slots {
			slotBytes := bytesutil.SlotToBytesBigEndian(slot)
			bkt := tx.Bucket(lightClientFinalizedUpdateBucket)
			if err := bkt.Delete(slotBytes); err != nil {
				continue
			}
		}
		return nil
	})
}

// lightClientUpdateKeysByFilter returns keys of light client updates from the database using query filter.
func lightClientUpdateKeysByFilter(ctx context.Context, tx *bolt.Tx, f *filters.QueryFilter, bkt *bolt.Bucket) ([][]byte, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.lightClientUpdateKeysByFilter")
	defer span.End()

	if f == nil {
		return nil, errors.New("must specify a filter criteria for retrieving blocks")
	}
	filtersMap := f.Filters()
	updateKeys, err := keysBySlotRange(
		ctx,
		bkt,
		filtersMap[filters.StartSlot],
		filtersMap[filters.EndSlot],
		filtersMap[filters.StartEpoch],
		filtersMap[filters.EndEpoch],
	)
	if err != nil {
		return nil, err
	}
	return updateKeys, nil
}

// keysBySlotRange returns keys of light client updates from the database using slot range.
func keysBySlotRange(
	ctx context.Context,
	bkt *bolt.Bucket,
	startSlotEncoded, endSlotEncoded, startEpochEncoded, endEpochEncoded interface{},
) ([][]byte, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.keysBySlotRange")
	defer span.End()

	if startSlotEncoded == nil && endSlotEncoded == nil && startEpochEncoded == nil && endEpochEncoded == nil {
		return [][]byte{}, nil
	}

	var startSlot, endSlot types.Slot
	var ok bool
	if startSlot, ok = startSlotEncoded.(types.Slot); !ok {
		startSlot = 0
	}
	if endSlot, ok = endSlotEncoded.(types.Slot); !ok {
		endSlot = 0
	}

	startEpoch, startEpochOk := startEpochEncoded.(types.Epoch)
	endEpoch, endEpochOk := endEpochEncoded.(types.Epoch)
	var err error
	if startEpochOk && endEpochOk {
		startSlot, err = slots.EpochStart(startEpoch)
		if err != nil {
			return nil, err
		}
		endSlot, err = slots.EpochStart(endEpoch)
		if err != nil {
			return nil, err
		}
		endSlot = endSlot + params.BeaconConfig().SlotsPerEpoch - 1
	}
	if endSlot < startSlot {
		return nil, errInvalidSlotRange
	}
	updates := make([][]byte, 0, endSlot.SubSlot(startSlot))
	min := bytesutil.SlotToBytesBigEndian(startSlot)
	max := bytesutil.SlotToBytesBigEndian(endSlot)
	c := bkt.Cursor()
	conditional := func(key, max []byte) bool {
		return key != nil && bytes.Compare(key, max) <= 0
	}
	for k, _ := c.Seek(min); conditional(k, max); k, _ = c.Next() {
		updates = append(updates, k)
	}
	return updates, nil
}
