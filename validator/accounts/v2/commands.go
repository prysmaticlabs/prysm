package accounts

import (
	"errors"

	"github.com/prysmaticlabs/prysm/shared/cmd"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/validator/flags"
	"github.com/urfave/cli/v2"
)

// Commands for all v2 validator accounts usage.
var Commands = &cli.Command{
	Name:     "accounts-v2",
	Category: "accounts-v2",
	Usage:    "v2 defines useful functions for interacting with the validator client's account",
	Subcommands: []*cli.Command{
		{
			Name: "create",
			Description: `creates a new validator account containing private keys for Ethereum 2.0 -
this command outputs a deposit data string which can be used to deposit Ether into the ETH1.0 deposit
contract in order to activate the validator client`,
			Flags: append(featureconfig.ActiveFlags(featureconfig.ValidatorFlags),
				[]cli.Flag{
					flags.KeystorePathFlag,
					flags.PasswordFlag,
					cmd.ChainConfigFileFlag,
				}...),
			Action: func(cliCtx *cli.Context) error {
				return errors.New("unimplemented - use v1 instead")
			},
		},
		{
			Name:        "list",
			Description: "lists all accounts available at a keystore path",
			Flags: append(featureconfig.ActiveFlags(featureconfig.ValidatorFlags),
				[]cli.Flag{
					flags.KeystorePathFlag,
					flags.PasswordFlag,
					cmd.ChainConfigFileFlag,
				}...),
			Action: func(cliCtx *cli.Context) error {
				return errors.New("unimplemented - use v1 instead")
			},
		},
		{
			Name:        "import",
			Description: "imports an account from another keystore into this keystore",
			Flags: append(featureconfig.ActiveFlags(featureconfig.ValidatorFlags),
				[]cli.Flag{
					flags.KeystorePathFlag,
					flags.PasswordFlag,
					cmd.ChainConfigFileFlag,
				}...),
			Action: func(cliCtx *cli.Context) error {
				return errors.New("unimplemented - use v1 instead")
			},
		},
		{
			Name:        "export",
			Description: "exports an account from the target keystore into a zip file",
			Flags: append(featureconfig.ActiveFlags(featureconfig.ValidatorFlags),
				[]cli.Flag{
					flags.KeystorePathFlag,
					flags.PasswordFlag,
					cmd.ChainConfigFileFlag,
				}...),
			Action: func(cliCtx *cli.Context) error {
				return errors.New("unimplemented - use v1 instead")
			},
		},
		{
			Name:        "move",
			Description: "moves an account from this keystore into a target keystore",
			Flags: append(featureconfig.ActiveFlags(featureconfig.ValidatorFlags),
				[]cli.Flag{
					flags.KeystorePathFlag,
					flags.PasswordFlag,
					cmd.ChainConfigFileFlag,
				}...),
			Action: func(cliCtx *cli.Context) error {
				return errors.New("unimplemented - use v1 instead")
			},
		},
	},
}
