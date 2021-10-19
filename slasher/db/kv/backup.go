package kv

import (
	"context"
	"fmt"
	"path"
	"time"

	"github.com/prysmaticlabs/prysm/shared/fileutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	bolt "go.etcd.io/bbolt"
	"go.opencensus.io/trace"
)

const backupsDirectoryName = "backups"

// Backup the database to the datadir backup directory.
// Example for backup: $DATADIR/backups/prysm_slasherdb_10291092.backup
func (s *Store) Backup(ctx context.Context, outputDir string, overridePermission bool) error {
	ctx, span := trace.StartSpan(ctx, "SlasherDB.Backup")
	defer span.End()

	var backupsDir string
	var err error
	if outputDir != "" {
		backupsDir, err = fileutil.ExpandPath(outputDir)
		if err != nil {
			return err
		}
	} else {
		backupsDir = path.Join(s.databasePath, backupsDirectoryName)
	}
	// Ensure the backups directory exists.
	if err := fileutil.HandleBackupDir(backupsDir, overridePermission); err != nil {
		return err
	}
	backupPath := path.Join(backupsDir, fmt.Sprintf("prysm_slasherdb_%d.backup", time.Now().Unix()))
	log.WithField("backup", backupPath).Info("Writing backup database")

	copyDB, err := bolt.Open(
		backupPath,
		params.BeaconIoConfig().ReadWritePermissions,
		&bolt.Options{Timeout: params.BeaconIoConfig().BoltTimeout},
	)
	if err != nil {
		return err
	}
	defer func() {
		if err := copyDB.Close(); err != nil {
			log.WithError(err).Error("Failed to close backup database")
		}
	}()

	return s.db.View(func(tx *bolt.Tx) error {
		return tx.ForEach(func(name []byte, b *bolt.Bucket) error {
			log.Debugf("Copying bucket %s\n", name)
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
