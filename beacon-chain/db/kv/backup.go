package kv

import (
	"context"
	"fmt"
	"path"

	"github.com/boltdb/bolt"
	"go.opencensus.io/trace"
)

const backupsDirectoryName = "backups"

// Backup the database to the datadir backup directory.
// Example for backup at slot 345: $DATADIR/backups/slot_0000345.backup
func (k *Store) Backup(ctx context.Context) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.Backup")
	defer span.End()

	backupsDir := path.Join(k.databasePath, backupsDirectoryName)
	head, err := k.HeadBlock(ctx)
	if err != nil {
		return err
	}
	backupPath := path.Join(backupsDir, fmt.Sprintf("slot_%7d.backup", head.Slot))
	return k.db.View(func(tx *bolt.Tx) error {
		return tx.CopyFile(backupPath, 0666)
	})
}
