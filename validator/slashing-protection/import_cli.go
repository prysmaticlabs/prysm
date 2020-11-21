package slashingprotection

import (
	"fmt"
	"os"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/shared/fileutil"
	"github.com/prysmaticlabs/prysm/validator/accounts/prompt"
	"github.com/prysmaticlabs/prysm/validator/db"
	"github.com/prysmaticlabs/prysm/validator/flags"
	slashingProtectionFormat "github.com/prysmaticlabs/prysm/validator/slashing-protection/local/standard-protection-format"
	"github.com/urfave/cli/v2"
)

func ImportSlashingProtectionCLI(cliCtx *cli.Context, valDB db.Database) error {
	protectionFilePath, err := prompt.InputDirectory(cliCtx, prompt.SlashingProtectionJSONPromptText, flags.SlashingProtectionJSONFileFlag)
	if err != nil {
		return errors.Wrap(err, "could not get slashing protection json file")
	}
	if protectionFilePath != "" {
		fullPath, err := fileutil.ExpandPath(protectionFilePath)
		if err != nil {
			return errors.Wrapf(err, "could not expand file path for %s", protectionFilePath)
		}
		if !fileutil.FileExists(fullPath) {
			return fmt.Errorf("file %s does not exist", fullPath)
		}
		protectionJSON, err := os.Open(fullPath)
		if err != nil {
			return errors.Wrapf(err, "could not read private key file at path %s", fullPath)
		}
		if err := slashingProtectionFormat.ImportStandardProtectionJSON(cliCtx.Context, valDB, protectionJSON); err != nil {
			return err
		}
	}
	return nil
}
