package kv

import (
	"bytes"
	"context"

	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	bolt "go.etcd.io/bbolt"
	"go.opencensus.io/trace"
)

func (s *Store) ValidatedTips(ctx context.Context) ([][32]byte, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.ValidatedTips")
	defer span.End()

	var valTips [][32]byte
	err := s.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(validatedTips)

		c := bkt.Cursor()
		for k, _ := c.First(); k != nil; k, _ = c.Next() {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			key := make([]byte, 32)
			copy(key, k)
			valTips = append(valTips, bytesutil.ToBytes32(key))
		}
		return nil
	})
	return valTips, err
}

func (s *Store) UpdateValidatedTips(ctx context.Context, newVals [][32]byte) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.UpdateValidatedTips")
	defer span.End()

	// Get the already existing tips.
	oldVals, err := s.ValidatedTips(ctx)
	if err != nil {
		return err
	}

	// find the entries to add and entries to delete
	var tipsToAdd [][32]byte
	var tipsToDelete [][32]byte
	tipsToAdd, tipsToDelete = tipDiff(oldVals, newVals)

	updateErr := s.db.Update(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(validatedTips)

		// Add the new tips.
		for i := 0; i < len(tipsToAdd); i++ {
			putEerr := bkt.Put(tipsToAdd[i][:], []byte{0})
			if putEerr != nil {
				return putEerr
			}
		}

		// Delete the marked tips.
		for i := 0; i < len(tipsToDelete); i++ {
			deleteErr := bkt.Delete(tipsToDelete[i][:])
			if deleteErr != nil {
				return deleteErr
			}
		}

		return nil
	})
	return updateErr
}

func tipDiff(oldKeys [][32]byte, newKeys [][32]byte) ([][32]byte, [][32]byte) {

	var keysToAdd [][32]byte
	var keysToDel [][32]byte

	// There is no diff, so return the same keys back.
	if len(oldKeys) == 0 {
		return newKeys, oldKeys
	}

	// Go through the old keys to see if anything needs to be deleted.
	var deleteKey bool
	for i := 0; i < len(oldKeys); i++ {
		deleteKey = true
		for j := 0; j < len(newKeys); j++ {
			if bytes.Equal(oldKeys[i][:], newKeys[j][:]) {
				deleteKey = false
				break
			}
		}

		if deleteKey {
			keysToDel = append(keysToDel, oldKeys[i])
		}
	}

	// Go through the new keys to see if anything needs to be added.
	var addKey bool
	for i := 0; i < len(newKeys); i++ {
		addKey = true
		for j := 0; j < len(oldKeys); j++ {
			if bytes.Equal(newKeys[i][:], oldKeys[j][:]) {
				addKey = false
				break
			}
		}
		if addKey {
			keysToAdd = append(keysToAdd, newKeys[i])
		}
	}

	return keysToAdd, keysToDel
}
