package slashingprotection

import (
	"fmt"
	"os"

	log "github.com/sirupsen/logrus"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/shared/fileutil"
	"github.com/prysmaticlabs/prysm/validator/accounts/prompt"
	"github.com/prysmaticlabs/prysm/validator/db"
	"github.com/prysmaticlabs/prysm/validator/flags"
	slashingProtectionFormat "github.com/prysmaticlabs/prysm/validator/slashing-protection/local/standard-protection-format"
	"github.com/urfave/cli/v2"
)

// ImportSlashingProtectionCLI is the CLI command for importing slashing protection into a JSON.
func ImportSlashingProtectionCLI(cliCtx *cli.Context, valDB db.Database) error {
	var err error
	if valDB == nil {
		return errors.New("validator database cannot be nil")
	}
	protectionFilePath, err := prompt.InputDirectory(cliCtx, prompt.SlashingProtectionJSONPromptText, flags.SlashingProtectionJSONFileFlag)
	if err != nil {
		return errors.Wrap(err, "could not get slashing protection json file")
	}
	if protectionFilePath == "" {
		return errors.Wrap(err, "invalid protection json path")
	}

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
	log.Info("Slashing protection JSON successfully imported")
	return nil
}
