package get

import "github.com/urfave/cli/v2"

var Commands = []*cli.Command{
	{
		Name:  "get",
		Usage: "commands for retrieving objects from a running beacon node",
		Subcommands: []*cli.Command{
			getBlockCmd,
			getStateCmd,
		},
	},
}
