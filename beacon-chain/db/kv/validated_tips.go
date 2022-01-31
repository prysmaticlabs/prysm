package kv

import (
	"context"

	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	bolt "go.etcd.io/bbolt"
	"go.opencensus.io/trace"
)

// ValidatedTips returns all the validated_tips that are present in the DB.
func (s *Store) ValidatedTips(ctx context.Context) (map[[32]byte]types.Slot, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.ValidatedTips")
	defer span.End()

	valTips := make(map[[32]byte]types.Slot, 1)
	err := s.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(validatedTips)

		c := bkt.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			valTips[bytesutil.ToBytes32(k)] = bytesutil.BytesToSlotBigEndian(v)
		}
		return nil
	})
	return valTips, err
}

// UpdateValidatedTips clears off all the old validated_tips from the DB and
// adds the new tips that are provided.
func (s *Store) UpdateValidatedTips(ctx context.Context, newVals map[[32]byte]types.Slot) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.UpdateValidatedTips")
	defer span.End()

	// Get the already existing tips.
	oldVals, err := s.ValidatedTips(ctx)
	if err != nil {
		return err
	}

	updateErr := s.db.Update(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(validatedTips)

		// Delete keys that are present and not in the new set.
		for k := range oldVals {
			if _, ok := newVals[k]; !ok {
				deleteErr := bkt.Delete(k[:])
				if deleteErr != nil {
					return deleteErr
				}

			}
		}

		// Add keys not present already.
		for k, v := range newVals {
			if _, ok := oldVals[k]; !ok {
				putErr := bkt.Put(k[:], bytesutil.SlotToBytesBigEndian(v))
				if putErr != nil {
					return putErr
				}
			}
		}
		return nil
	})
	return updateErr
}
