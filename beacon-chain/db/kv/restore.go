package kv

import (
	"context"
	"errors"
	"path"

	"github.com/prysmaticlabs/prysm/shared/fileutil"
	"go.opencensus.io/trace"
)

// Restore restores the database using fromFilePath as the restore point.
func Restore(ctx context.Context, fromFilePath string, datadir string) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.Restore")
	defer span.End()

	restoreDir := path.Join(datadir, BeaconNodeDbDirName)
	if err := fileutil.MkdirAll(restoreDir); err != nil {
		return err
	}
	if fileutil.FileExists(path.Join(restoreDir, databaseFileName)) {
		return errors.New("database file already exists in target directory")
	}
	if err := fileutil.CopyFile(fromFilePath, path.Join(datadir, BeaconNodeDbDirName, databaseFileName)); err != nil {
		return err
	}

	return nil
}
