package unfinalizedblocks

import "github.com/urfave/cli/v2"

var Commands = []*cli.Command{
	{
		Name:    "unfinalized-blocks",
		Aliases: []string{"ub"},
		Usage:   "commands for managing unfinalized blocks",
		Subcommands: []*cli.Command{
			downloadCmd,
		},
	},
}
