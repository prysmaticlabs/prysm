package withdrawal

import (
	"os"

	"github.com/prysmaticlabs/prysm/v3/cmd"
	"github.com/prysmaticlabs/prysm/v3/config/features"
	"github.com/prysmaticlabs/prysm/v3/runtime/tos"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

var Commands = []*cli.Command{
	{
		Name:    "set-withdrawal-address",
		Aliases: []string{"swa"},
		Usage:   "command for setting the withdrawal ethereum address to the associated validator key",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "beacon-node-host",
				Usage:       "host:port for beacon node to query",
				Destination: &withdrawalFlags.BeaconNodeHost,
				Value:       "http://localhost:3500",
			},
			&cli.StringFlag{
				Name:        "file",
				Usage:       "file location for for the blsToExecutionAddress JSON or Yaml",
				Destination: &withdrawalFlags.File,
				Value:       "./blsToExecutionAddress.json",
			},
			features.Mainnet,
			features.PraterTestnet,
			features.RopstenTestnet,
			features.SepoliaTestnet,
			cmd.AcceptTosFlag,
		},
		Before: func(cliCtx *cli.Context) error {
			if err := cmd.LoadFlagsFromConfig(cliCtx, cliCtx.Command.Flags); err != nil {
				return err
			}
			return tos.VerifyTosAcceptedOrPrompt(cliCtx)
		},
		Action: func(cliCtx *cli.Context) error {
			if err := setWithdrawlAddress(cliCtx, os.Stdin); err != nil {
				log.WithError(err).Fatal("Could not set withdrawal address")
			}
			return nil
		},
	},
}
