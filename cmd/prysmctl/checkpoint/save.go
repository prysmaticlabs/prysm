package checkpoint

import (
	"context"
	"os"
	"time"

	"github.com/prysmaticlabs/prysm/v3/api/client/beacon"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

var saveFlags = struct {
	BeaconNodeHost string
	Timeout        time.Duration
}{}

var saveCmd = &cli.Command{
	Name:   "save",
	Usage:  "Save the latest finalized header and the most recent block it integrates. To be used for checkpoint sync.",
	Action: cliActionSave,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:        "beacon-node-host",
			Usage:       "host:port for beacon node connection",
			Destination: &saveFlags.BeaconNodeHost,
			Value:       "localhost:3500",
		},
		&cli.DurationFlag{
			Name:        "http-timeout",
			Usage:       "timeout for http requests made to beacon-node-url (uses duration format, ex: 2m31s). default: 4m",
			Destination: &saveFlags.Timeout,
			Value:       time.Minute * 4,
		},
	},
}

func cliActionSave(_ *cli.Context) error {
	ctx := context.Background()
	f := saveFlags

	opts := []beacon.ClientOpt{beacon.WithTimeout(f.Timeout)}
	client, err := beacon.NewClient(saveFlags.BeaconNodeHost, opts...)
	if err != nil {
		return err
	}

	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	od, err := beacon.DownloadFinalizedData(ctx, client)
	if err != nil {
		return err
	}

	blockPath, err := od.SaveBlock(cwd)
	if err != nil {
		return err
	}
	log.Printf("saved ssz-encoded block to to %s", blockPath)

	statePath, err := od.SaveState(cwd)
	if err != nil {
		return err
	}
	log.Printf("saved ssz-encoded state to to %s", statePath)

	return nil
}
