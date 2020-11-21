package slashingprotection

import (
	"log"

	"github.com/prysmaticlabs/prysm/validator/accounts/prompt"

	"github.com/prysmaticlabs/prysm/shared/cmd"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/tos"
	"github.com/prysmaticlabs/prysm/validator/db/kv"
	"github.com/prysmaticlabs/prysm/validator/flags"
	"github.com/urfave/cli/v2"
)

// ProtectionCommands for managing EIP-3076 slashing interchange JSON.
var ProtectionCommands = &cli.Command{
	Name:     "slashing-protection",
	Category: "slashing protection",
	Usage:    "defines commands for managing the EIP-3076 compliant slashing protection data",
	Subcommands: []*cli.Command{
		{
			Name:        "import",
			Description: `imported a selected EIP-3076 compliant slashing protection JSON to the validator database`,
			Flags: cmd.WrapFlags([]cli.Flag{
				cmd.DataDirFlag,
				flags.SlashingProtectionJSONFileFlag,
				featureconfig.Mainnet,
				featureconfig.PyrmontTestnet,
				featureconfig.ToledoTestnet,
				cmd.AcceptTosFlag,
			}),
			Before: func(cliCtx *cli.Context) error {
				if err := cmd.LoadFlagsFromConfig(cliCtx, cliCtx.Command.Flags); err != nil {
					return err
				}
				return tos.VerifyTosAcceptedOrPrompt(cliCtx)
			},
			Action: func(cliCtx *cli.Context) error {
				featureconfig.ConfigureValidator(cliCtx)
				var err error
				dataDir := cliCtx.String(cmd.DataDirFlag.Name)
				if !cliCtx.IsSet(cmd.DataDirFlag.Name) {
					var err error
					dataDir, err = prompt.InputDirectory(cliCtx, prompt.DataDirDirPromptText, cmd.DataDirFlag)
					if err != nil {
						log.Fatalf("Could not get database directory: %v", err)
					}
				}
				valDB, err := kv.NewKVStore(dataDir, make([][48]byte, 0))
				if err != nil {
					log.Fatalf("Could not access validator database at path: %s: %v", dataDir, err)
				}
				if err := ImportSlashingProtectionCLI(cliCtx, valDB); err != nil {
					log.Fatalf("Could not import slashing protection JSON: %v", err)
				}
				return nil
			},
		},
	},
}
