package kv

import (
	"context"
	"fmt"
	"os"
	"path"

	"github.com/boltdb/bolt"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
)

const backupsDirectoryName = "backups"

// Backup the database to the datadir backup directory.
// Example for backup at slot 345: $DATADIR/backups/prysm_beacondb_at_slot_0000345.backup
func (k *Store) Backup(ctx context.Context) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.Backup")
	defer span.End()

	backupsDir := path.Join(k.databasePath, backupsDirectoryName)
	head, err := k.HeadBlock(ctx)
	if err != nil {
		return err
	}
	if head == nil {
		return errors.New("no head block")
	}
	// Ensure the backups directory exists.
	if err := os.MkdirAll(backupsDir, os.ModePerm); err != nil {
		return err
	}
	backupPath := path.Join(backupsDir, fmt.Sprintf("prysm_beacondb_at_slot_%07d.backup", head.Slot))
	logrus.WithField("prefix", "db").WithField("backup", backupPath).Info("Writing backup database.")
	return k.db.View(func(tx *bolt.Tx) error {
		return tx.CopyFile(backupPath, 0666)
	})
}
