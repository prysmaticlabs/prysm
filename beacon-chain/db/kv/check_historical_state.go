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

	regenHistoricalStatesConfirmed := false
	var err error
	if historicalStateDeleted {
		actionText := "Looks like you stopped using --new-state-mgmt. To reuse it, the node will need " +
			"to generate and save historical states. The process may take a while, - do you want to proceed? (Y/N)"
		deniedText := "Historical states will not be generated. Please remove usage --new-state-mgmt"

		regenHistoricalStatesConfirmed, err = cmd.ConfirmAction(actionText, deniedText)
		if err != nil {
			return err
		}

		if !regenHistoricalStatesConfirmed {
			return errors.New("exiting... please do not run with flag --new-state-mgmt")
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
