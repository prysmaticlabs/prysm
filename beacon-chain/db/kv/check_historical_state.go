package kv

import (
	"context"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/shared/cmd"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	bolt "go.etcd.io/bbolt"
)

var historicalStateDeletedKey = []byte("historical-states-deleted")

func (kv *Store) ensureNewStateServiceCompatible(ctx context.Context) error {
	if featureconfig.Get().NoNewStateMgmt {
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

	regenHistoricalStatesConfirmed := false
	var err error
	if historicalStateDeleted {
		actionText := "--no-new-state-mgmt was used. To proceed without the flag, the db will need " +
			"to generate and save historical states. This process may take a while, - do you want to proceed? (Y/N)"
		deniedText := "Historical states will not be generated. Please continue use --no-new-state-mgmt"

		regenHistoricalStatesConfirmed, err = cmd.ConfirmAction(actionText, deniedText)
		if err != nil {
			return err
		}

		if !regenHistoricalStatesConfirmed {
			return errors.New("exiting... please use --no-new-state-mgmt")
		}

		if err := kv.regenHistoricalStates(ctx); err != nil {
			return errors.Wrap(err, "could not regenerate historical states, please retry")
		}
	}

	return kv.db.Update(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(newStateServiceCompatibleBucket)
		return bkt.Put(historicalStateDeletedKey, []byte{0x00})
	})
}
