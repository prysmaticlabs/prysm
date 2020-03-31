package kv

import (
	"errors"

	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	bolt "go.etcd.io/bbolt"
)

var historicalStateDeletedKey = []byte("historical-states-deleted")

func (kv *Store) ensureNewStateServiceCompatible() error {
	if !featureconfig.Get().NewStateMgmt {
		return kv.db.Update(func(tx *bolt.Tx) error {
			bkt := tx.Bucket(newStateServiceCompatibleBucket)
			return bkt.Put(historicalStateDeletedKey, []byte{0x01})
		})
	}

	var historicalStateDeleted bool
	kv.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(newStateServiceCompatibleBucket)
		v := bkt.Get(historicalStateDeletedKey)
		historicalStateDeleted = len(v) == 1 && v[0] == 0x01
		return nil
	})

	if historicalStateDeleted {
		return errors.New("historical states were pruned in db, do not run with flag --new-state-mgmt")
	}

	return nil
}
