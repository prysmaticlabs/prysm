package slashingprotection

import (
	"bytes"
	"fmt"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/cmd/validator/flags"
	"github.com/prysmaticlabs/prysm/shared/cmd"
	"github.com/prysmaticlabs/prysm/shared/fileutil"
	"github.com/prysmaticlabs/prysm/validator/accounts/prompt"
	"github.com/prysmaticlabs/prysm/validator/db/kv"
	slashingProtectionFormat "github.com/prysmaticlabs/prysm/validator/slashing-protection/local/standard-protection-format"
	"github.com/urfave/cli/v2"
)

// ImportSlashingProtectionCLI reads an input slashing protection EIP-3076
// standard JSON file and attempts to insert its data into our validator DB.
//
// Steps:
// 1. Parse a path to the validator's datadir from the CLI context.
// 2. Open the validator database.
// 3. Read the JSON file from user input.
// 4. Call the function which actually imports the data from
// from the standard slashing protection JSON file into our database.
func ImportSlashingProtectionCLI(cliCtx *cli.Context) error {
	var err error
	dataDir := cliCtx.String(cmd.DataDirFlag.Name)
	if !cliCtx.IsSet(cmd.DataDirFlag.Name) {
		dataDir, err = prompt.InputDirectory(cliCtx, prompt.DataDirDirPromptText, cmd.DataDirFlag)
		if err != nil {
			return errors.Wrapf(err, "could not read directory value from input")
		}
	}
	// ensure that the validator.db is found under the specified dir or its subdirectories
	found, _, err := fileutil.RecursiveFileFind(kv.ProtectionDbFileName, dataDir)
	if err != nil {
		return errors.Wrapf(err, "error finding validator database at path %s", dataDir)
	}
	if !found {
		log.Infof(
			"Did not find existing validator.db inside of %s, creating a new one",
			dataDir,
		)
	} else {
		log.Infof("Found existing validator.db inside of %s", dataDir)
	}
	valDB, err := kv.NewKVStore(cliCtx.Context, dataDir, &kv.Config{})
	if err != nil {
		return errors.Wrapf(err, "could not access validator database at path: %s", dataDir)
	}
	defer func() {
		if err := valDB.Close(); err != nil {
			log.WithError(err).Errorf("Could not close validator DB")
		}
	}()
	protectionFilePath, err := prompt.InputDirectory(cliCtx, prompt.SlashingProtectionJSONPromptText, flags.SlashingProtectionJSONFileFlag)
	if err != nil {
		return errors.Wrap(err, "could not get slashing protection json file")
	}
	if protectionFilePath == "" {
		return fmt.Errorf(
			"no path to a slashing_protection.json file specified, please retry or "+
				"you can also specify it with the %s flag",
			flags.SlashingProtectionJSONFileFlag.Name,
		)
	}
	enc, err := fileutil.ReadFileAsBytes(protectionFilePath)
	if err != nil {
		return err
	}
	log.Infof("Starting import of slashing protection file %s", protectionFilePath)
	buf := bytes.NewBuffer(enc)
	if err := slashingProtectionFormat.ImportStandardProtectionJSON(
		cliCtx.Context, valDB, buf,
	); err != nil {
		return err
	}
	log.Infof("Slashing protection JSON successfully imported into %s", dataDir)
	return nil
}
