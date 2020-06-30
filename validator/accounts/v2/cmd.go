package v2

import (
	"github.com/prysmaticlabs/prysm/shared/cmd"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/validator/flags"
	"github.com/urfave/cli/v2"
)

// Commands for accounts-v2 for Prysm validator clients.
var Commands = &cli.Command{
	Name:     "accounts-v2",
	Category: "accounts-v2",
	Usage:    "defines commands for interacting with eth2 validator accounts (work in progress)",
	Subcommands: []*cli.Command{
		{
			Name: "new",
			Description: `creates a new validator account for eth2. creates a new wallet for a user based on
specified input, capable of creating a direct, derived, or remote wallet.
this command outputs a deposit data string which can be used to deposit Ether into the ETH1.0 deposit
contract in order to activate the validator client`,
			Flags: append(featureconfig.ActiveFlags(featureconfig.ValidatorFlags),
				[]cli.Flag{
					flags.WalletDirFlag,
					flags.WalletPasswordsDirFlag,
					cmd.DataDirFlag, // TODO: Replace by wallet path
				}...),
			Action: New,
		},
	},
}
