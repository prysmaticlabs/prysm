package kv

import (
	"context"
	"fmt"
	"os"
	"path"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/sirupsen/logrus"
	bolt "go.etcd.io/bbolt"
	"go.opencensus.io/trace"
)

const backupsDirectoryName = "backups"

// Backup the database to the datadir backup directory.
// Example for backup at slot 345: $DATADIR/backups/prysm_beacondb_at_slot_0000345.backup
func (kv *Store) Backup(ctx context.Context) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.Backup")
	defer span.End()

	backupsDir := path.Join(kv.databasePath, backupsDirectoryName)
	head, err := kv.HeadBlock(ctx)
	if err != nil {
		return err
	}
	if head == nil {
		return errors.New("no head block")
	}
	// Ensure the backups directory exists.
	if err := os.MkdirAll(backupsDir, params.BeaconIoConfig().ReadWriteExecutePermissions); err != nil {
		return err
	}
	backupPath := path.Join(backupsDir, fmt.Sprintf("prysm_beacondb_at_slot_%07d.backup", head.Block.Slot))
	logrus.WithField("prefix", "db").WithField("backup", backupPath).Info("Writing backup database.")

	copyDB, err := bolt.Open(backupPath, params.BeaconIoConfig().ReadWritePermissions, &bolt.Options{Timeout: params.BeaconIoConfig().BoltTimeout})
	if err != nil {
		panic(err)
	}
	defer func() {
		if err := copyDB.Close(); err != nil {
			logrus.WithError(err).Error("Failed to close destination database")
		}
	}()

	return kv.db.View(func(tx *bolt.Tx) error {
		return tx.ForEach(func(name []byte, b *bolt.Bucket) error {
			logrus.Debugf("Copying bucket %s\n", name)
			return copyDB.Update(func(tx2 *bolt.Tx) error {
				b2, err := tx2.CreateBucketIfNotExists(name)
				if err != nil {
					return err
				}
				return b.ForEach(b2.Put)
			})
		})
	})
}
