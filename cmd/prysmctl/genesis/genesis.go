package genesis

import "github.com/urfave/cli/v2"

var Commands = []*cli.Command{
	{
		Name:  "genesis",
		Usage: "commands for dealing with Ethereum beacon chain genesis data",
		Subcommands: []*cli.Command{
			{
				Name:        "generate",
				Usage:       "commands for generating beacon chain genesis items",
				Subcommands: []*cli.Command{generateGenesisStateCmd},
			},
		},
	},
}
