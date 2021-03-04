package db

import (
	"os"
	"path"
	"strings"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/shared/cmd"
	"github.com/prysmaticlabs/prysm/shared/fileutil"
	"github.com/prysmaticlabs/prysm/shared/promptutil"
	"github.com/prysmaticlabs/prysm/validator/db/kv"
	"github.com/urfave/cli/v2"
)

const dbExistsYesNoPrompt = "A database file already exists in the target directory. " +
	"Are you sure that you want to overwrite it? [y/n]"

// Restore a Prysm validator database.
func Restore(cliCtx *cli.Context) error {
	sourceFile := cliCtx.String(cmd.RestoreSourceFileFlag.Name)
	targetDir := cliCtx.String(cmd.RestoreTargetDirFlag.Name)

	if fileutil.FileExists(path.Join(targetDir, kv.ProtectionDbFileName)) {
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
	if err := fileutil.MkdirAll(targetDir); err != nil {
		return err
	}
	if err := fileutil.CopyFile(sourceFile, path.Join(targetDir, kv.ProtectionDbFileName)); err != nil {
		return err
	}

	log.Info("Restore completed successfully")
	return nil
}
