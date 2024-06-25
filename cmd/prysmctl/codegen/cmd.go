package codegen

import (
	"github.com/OffchainLabs/methodical-ssz/cmd/ssz/commands"
	"github.com/urfave/cli/v2"
)

var Commands = []*cli.Command{
	{
		Name:        "ssz",
		Usage:       "ssz code generation utilities",
		Subcommands: commands.All,
	},
}
