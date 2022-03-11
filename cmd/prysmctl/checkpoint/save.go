package checkpoint

import (
	"context"
	"os"
	"time"

	"github.com/prysmaticlabs/prysm/api/client/beacon"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

var saveFlags = struct {
	BeaconNodeHost string
	Timeout        string
}{}

var saveCmd = &cli.Command{
	Name:   "save",
	Usage:  "query for the current weak subjectivity period epoch, then download the corresponding state and block. To be used for checkpoint sync.",
	Action: cliActionSave,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:        "beacon-node-host",
			Usage:       "host:port for beacon node connection",
			Destination: &saveFlags.BeaconNodeHost,
			Value:       "localhost:3500",
		},
		&cli.StringFlag{
			Name:        "http-timeout",
			Usage:       "timeout for http requests made to beacon-node-url (uses duration format, ex: 2m31s). default: 2m",
			Destination: &saveFlags.Timeout,
			Value:       "4m",
		},
	},
}

func cliActionSave(_ *cli.Context) error {
	f := saveFlags
	opts := make([]beacon.ClientOpt, 0)
	log.Printf("--beacon-node-url=%s", f.BeaconNodeHost)
	timeout, err := time.ParseDuration(f.Timeout)
	if err != nil {
		return err
	}
	opts = append(opts, beacon.WithTimeout(timeout))
	client, err := beacon.NewClient(saveFlags.BeaconNodeHost, opts...)
	if err != nil {
		return err
	}

	return saveCheckpoint(client)
}

func saveCheckpoint(client *beacon.Client) error {
	ctx := context.Background()
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	od, err := beacon.DownloadOriginData(ctx, client)
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
