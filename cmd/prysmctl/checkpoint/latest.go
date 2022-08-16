package checkpoint

import (
	"context"
	"fmt"
	"time"

	"github.com/prysmaticlabs/prysm/v3/api/client/beacon"
	"github.com/urfave/cli/v2"
)

var latestFlags = struct {
	BeaconNodeHost string
	Timeout        time.Duration
}{}

var latestCmd = &cli.Command{
	Name:   "latest",
	Usage:  "Compute the latest weak subjectivity checkpoint (block_root:epoch) using trusted server data.",
	Action: cliActionLatest,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:        "beacon-node-host",
			Usage:       "host:port for beacon node to query",
			Destination: &latestFlags.BeaconNodeHost,
			Value:       "http://localhost:3500",
		},
		&cli.DurationFlag{
			Name:        "http-timeout",
			Usage:       "timeout for http requests made to beacon-node-url (uses duration format, ex: 2m31s). default: 2m",
			Destination: &latestFlags.Timeout,
			Value:       time.Minute * 2,
		},
	},
}

func cliActionLatest(_ *cli.Context) error {
	ctx := context.Background()
	f := latestFlags

	opts := []beacon.ClientOpt{beacon.WithTimeout(f.Timeout)}
	client, err := beacon.NewClient(latestFlags.BeaconNodeHost, opts...)
	if err != nil {
		return err
	}

	ws, err := beacon.ComputeWeakSubjectivityCheckpoint(ctx, client)
	if err != nil {
		return err
	}
	fmt.Println("\nUse the following flag when starting a prysm Beacon Node to ensure the chain history " +
		"includes the Weak Subjectivity Checkpoint: ")
	fmt.Printf("--weak-subjectivity-checkpoint=%s\n\n", ws.CheckpointString())
	return nil
}
