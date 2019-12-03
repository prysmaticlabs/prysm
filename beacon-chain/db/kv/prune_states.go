package kv

import (
	"bytes"
	"context"

	"github.com/boltdb/bolt"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/db/filters"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/sirupsen/logrus"
)

var pruneStatesKey = []byte("prune-states")

func (kv *Store) pruneStates(ctx context.Context) error {
	var pruned bool

	if !featureconfig.Get().PruneEpochBoundaryStates {
		return kv.db.Update(func(tx *bolt.Tx) error {
			bkt := tx.Bucket(migrationBucket)
			return bkt.Put(pruneStatesKey, []byte{0x00})
		})
	}

	kv.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(migrationBucket)
		v := bkt.Get(pruneStatesKey)
		pruned = len(v) == 1 && v[0] == 0x01
		return nil
	})

	if pruned {
		return nil
	}

	log := logrus.WithField("prefix", "kv")
	log.Info("Pruning states before last finalized check point. This might take a while...")

	roots, err := kv.rootsToPrune(ctx)
	if err != nil {
		return err
	}

	if err := kv.DeleteStates(ctx, roots); err != nil {
		return err
	}

	return kv.db.Update(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(migrationBucket)
		return bkt.Put(pruneStatesKey, []byte{0x01})
	})
}

// This retrieves the key roots needed to prune states
// * Get last finalized check point
// * Rewind end slot until it's not finalized root
// * return roots between slot 1 and end slot
func (kv *Store) rootsToPrune(ctx context.Context) ([][32]byte, error) {
	cp, err := kv.FinalizedCheckpoint(ctx)
	if err != nil {
		return nil, err
	}
	f := filters.NewFilter().SetStartSlot(1).SetEndSlot(helpers.StartSlot(cp.Epoch))
	roots, err := kv.BlockRoots(ctx, f)
	if err != nil {
		return nil, err
	}
	// Ensure we don't delete finalized root
	i := 0
	if len(roots) > 1 {
		i = len(roots) - 1
		for bytes.Equal(roots[i][:], cp.Root) {
			i--
		}
	}

	return roots[:i], nil
}
