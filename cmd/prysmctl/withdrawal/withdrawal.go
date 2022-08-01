package withdrawal

import (
	"github.com/urfave/cli/v2"
)

var withdrawalFlags = struct {
	File string
}{}

var Commands = []*cli.Command{
	{
		Name:    "set-withdrawal-address",
		Aliases: []string{"swa"},
		Usage:   "command for setting the withdrawal ethereum address to the associated validator key",
		Action:  cliActionLatest,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "file",
				Usage:       "file location for for the blsToExecutionAddress JSON or Yaml",
				Destination: &withdrawalFlags.File,
				Value:       "",
			},
		},
	},
}

func cliActionLatest(_ *cli.Context) error {
	return nil
}
