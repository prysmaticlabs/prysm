package slashingprotection

import (
	"github.com/prysmaticlabs/prysm/shared/cmd"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/tos"
	"github.com/prysmaticlabs/prysm/validator/flags"
	"github.com/urfave/cli/v2"
)

// Commands for slashing protection.
var Commands = &cli.Command{
	Name:     "slashing-protection",
	Category: "slashing-protection",
	Usage:    "defines commands for interacting your validator's slashing protection history",
	Subcommands: []*cli.Command{
		{
			Name:        "export",
			Description: `exports your validator slashing protection history into an EIP-3076 compliant JSON`,
			Flags: cmd.WrapFlags([]cli.Flag{
				cmd.DataDirFlag,
				flags.SlashingProtectionExportDirFlag,
			}),
			Before: func(cliCtx *cli.Context) error {
				if err := cmd.LoadFlagsFromConfig(cliCtx, cliCtx.Command.Flags); err != nil {
					return err
				}
				return tos.VerifyTosAcceptedOrPrompt(cliCtx)
			},
			Action: func(cliCtx *cli.Context) error {
				featureconfig.ConfigureValidator(cliCtx)
				return ExportSlashingProtectionJSONCli(cliCtx)
			},
		},
		{
			Name:        "import",
			Description: `imports a selected EIP-3076 compliant slashing protection JSON to the validator database`,
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
				return ImportSlashingProtectionCLI(cliCtx)
			},
		},
	},
}
