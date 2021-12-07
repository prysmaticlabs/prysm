package ssz

import "github.com/urfave/cli/v2"

var Commands = []*cli.Command{
	{
		Name:  "ssz",
		Usage: "commands for interacting with ssz values",
		Subcommands: []*cli.Command{
			inspectCmd,
		},
	},
}
