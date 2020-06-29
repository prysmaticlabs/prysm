package v2

import (
	"github.com/prysmaticlabs/prysm/shared/cmd"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/validator/flags"
	"github.com/urfave/cli/v2"
)

var Commands = &cli.Command{
	Name:     "accounts-v2",
	Category: "accounts-v2",
	Usage:    "defines useful functions for interacting with the validator client's account",
	Subcommands: []*cli.Command{
		{
			Name: "new",
			Description: `creates a new validator account keystore containing private keys for Ethereum 2.0 -
this command outputs a deposit data string which can be used to deposit Ether into the ETH1.0 deposit
contract in order to activate the validator client`,
			Flags: append(featureconfig.ActiveFlags(featureconfig.ValidatorFlags),
				[]cli.Flag{
					flags.KeystorePathFlag,
					flags.PasswordFlag,
					cmd.DataDirFlag, // TODO: Replace by wallet path
				}...),
			Action: New,
		},
	},
}
