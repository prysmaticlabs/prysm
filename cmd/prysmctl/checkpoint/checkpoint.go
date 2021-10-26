package checkpoint

import "github.com/urfave/cli/v2"

var Commands = []*cli.Command{
	{
		Name:    "checkpoint",
		Aliases: []string{"cpt"},
		Usage:   "commands for managing checkpoint syncing",
		Subcommands: []*cli.Command{
			latestCmd,
			saveCmd,
		},
	},
}
