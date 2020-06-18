package accounts

import (
	"github.com/prysmaticlabs/prysm/shared/cmd"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/validator/flags"
	cli "github.com/urfave/cli/v2"
)

// Commands for all v2 validator accounts usage.
var Commands = &cli.Command{
	Name:     "accounts",
	Category: "accounts",
	Usage:    "defines useful functions for interacting with the validator client's account",
	Subcommands: []*cli.Command{
		{
			Name: "create",
			Description: `creates a new validator account keystore containing private keys for Ethereum 2.0 -
this command outputs a deposit data string which can be used to deposit Ether into the ETH1.0 deposit
contract in order to activate the validator client`,
			Flags: append(featureconfig.ActiveFlags(featureconfig.ValidatorFlags),
				[]cli.Flag{
					flags.KeystorePathFlag,
					flags.PasswordFlag,
					cmd.ChainConfigFileFlag,
				}...),
			Action: func(cliCtx *cli.Context) error {
				return nil
			},
		},
	},
}
