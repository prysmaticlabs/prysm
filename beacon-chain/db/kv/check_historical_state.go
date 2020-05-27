package kv

import (
	"context"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	log "github.com/sirupsen/logrus"
	bolt "go.etcd.io/bbolt"
)

var historicalStateDeletedKey = []byte("historical-states-deleted")

// HistoricalStatesDeleted verifies historical states exist in DB.
func (kv *Store) HistoricalStatesDeleted(ctx context.Context) error {
	if !featureconfig.Get().NewStateMgmt {
		return kv.db.Update(func(tx *bolt.Tx) error {
			bkt := tx.Bucket(newStateServiceCompatibleBucket)
			return bkt.Put(historicalStateDeletedKey, []byte{0x01})
		})
	}

	var historicalStateDeleted bool
	if err := kv.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(newStateServiceCompatibleBucket)
		v := bkt.Get(historicalStateDeletedKey)
		historicalStateDeleted = len(v) == 1 && v[0] == 0x01
		return nil
	}); err != nil {
		return err
	}

	if historicalStateDeleted {
		log.Warn("Regenerating and saving historical states. This may take a while. Skip this with --skip-regen-historical-states")
		if err := kv.regenHistoricalStates(ctx); err != nil {
			return errors.Wrap(err, "could not regenerate historical states, please retry")
		}
	}

	return kv.db.Update(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(newStateServiceCompatibleBucket)
		return bkt.Put(historicalStateDeletedKey, []byte{0x00})
	})
}
