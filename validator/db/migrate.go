package db

import (
	"context"
	"path"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/shared/cmd"
	"github.com/prysmaticlabs/prysm/shared/fileutil"
	"github.com/prysmaticlabs/prysm/validator/db/kv"
	"github.com/urfave/cli/v2"
)

func migrateUp(cliCtx *cli.Context) error {
	dataDir := cliCtx.String(cmd.DataDirFlag.Name)

	if !fileutil.FileExists(path.Join(dataDir, kv.ProtectionDbFileName)) {
		return errors.New("No validator db found at path, nothing to migrate")
	}

	ctx := context.Background()
	log.Info("Opening DB")
	validatorDB, err := kv.NewKVStore(ctx, dataDir, &kv.Config{})
	if err != nil {
		return err
	}
	log.Info("Running migrations")
	return validatorDB.RunUpMigrations(ctx)
}

func migrateDown(cliCtx *cli.Context) error {
	dataDir := cliCtx.String(cmd.DataDirFlag.Name)

	if !fileutil.FileExists(path.Join(dataDir, kv.ProtectionDbFileName)) {
		return errors.New("No validator db found at path, nothing to rollback")
	}

	ctx := context.Background()
	log.Info("Opening DB")
	validatorDB, err := kv.NewKVStore(ctx, dataDir, &kv.Config{})
	if err != nil {
		return err
	}
	log.Info("Running migrations")
	return validatorDB.RunDownMigrations(ctx)
}
