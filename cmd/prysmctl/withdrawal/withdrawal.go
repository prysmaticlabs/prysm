package withdrawal

import (
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/api/client/beacon"
	"github.com/urfave/cli/v2"
	"gopkg.in/yaml.v2"
)

var withdrawalFlags = struct {
	BeaconNodeHost string
	File           string
}{}

var Commands = []*cli.Command{
	{
		Name:    "set-withdrawal-address",
		Aliases: []string{"swa"},
		Usage:   "command for setting the withdrawal ethereum address to the associated validator key",
		Action:  cliActionLatest,
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
		},
	},
}

func cliActionLatest(_ *cli.Context) error {
	f := withdrawalFlags

	cleanpath := filepath.Clean(f.File)
	b, err := os.ReadFile(cleanpath)
	if err != nil {
		return errors.Wrap(err, "failed to open file")
	}
	var to BlsToExecutionEngineFile
	if err := yaml.Unmarshal(b, to); err != nil {
		return errors.Wrap(err, "failed to unmarshal file")
	}
	if to.Message == nil {
		return errors.New("the message field in file is empty")
	}

	_, err = beacon.NewClient(withdrawalFlags.BeaconNodeHost)
	if err != nil {
		return err
	}

	return nil
}
