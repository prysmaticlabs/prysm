package db

import (
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/beacon-chain/db/kv"
	"github.com/prysmaticlabs/prysm/shared/cmd"
	"github.com/prysmaticlabs/prysm/shared/fileutil"
	"github.com/prysmaticlabs/prysm/shared/promptutil"
	"github.com/urfave/cli/v2"
)

const dbExistsYesNoPrompt = "A database file already exists in the target directory. " +
	"Are you sure that you want to overwrite it? [y/n]"

// Restore a beacon chain database.
func Restore(cliCtx *cli.Context) error {
	sourceFile := cliCtx.String(cmd.RestoreSourceFileFlag.Name)
	targetDir := cliCtx.String(cmd.RestoreTargetDirFlag.Name)
	// Sanitize user input paths
	sourceFile = filepath.Clean(sourceFile)
	targetDir = filepath.Clean(targetDir)

	restoreDir := path.Join(targetDir, kv.BeaconNodeDbDirName)
	if fileutil.FileExists(path.Join(restoreDir, kv.DatabaseFileName)) {
		resp, err := promptutil.ValidatePrompt(
			os.Stdin, dbExistsYesNoPrompt, promptutil.ValidateYesOrNo,
		)
		if err != nil {
			return errors.Wrap(err, "could not validate choice")
		}
		if strings.EqualFold(resp, "n") {
			log.Info("Restore aborted")
			return nil
		}
	}
	if err := fileutil.MkdirAll(restoreDir); err != nil {
		return err
	}
	if err := fileutil.CopyFile(sourceFile, path.Join(restoreDir, kv.DatabaseFileName)); err != nil {
		return err
	}

	log.Info("Restore completed successfully")
	return nil
}
