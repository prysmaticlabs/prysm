package checkpoint

import (
	"context"
	"fmt"
	"time"

	"github.com/prysmaticlabs/prysm/api/client/beacon"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

var latestFlags = struct {
	BeaconNodeHost string
	Timeout        string
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
		&cli.StringFlag{
			Name:        "http-timeout",
			Usage:       "timeout for http requests made to beacon-node-url (uses duration format, ex: 2m31s). default: 2m",
			Destination: &latestFlags.Timeout,
			Value:       "2m",
		},
	},
}

func cliActionLatest(_ *cli.Context) error {
	ctx := context.Background()
	f := latestFlags

	timeout, err := time.ParseDuration(f.Timeout)
	if err != nil {
		return err
	}
	opts := []beacon.ClientOpt{beacon.WithTimeout(timeout)}
	client, err := beacon.NewClient(latestFlags.BeaconNodeHost, opts...)
	if err != nil {
		return err
	}

	od, err := beacon.DownloadOriginData(ctx, client)
	if err != nil {
		log.Fatalf(err.Error())
	}
	fmt.Println("\nUse the following flag when starting a prysm Beacon Node to ensure the chain history " +
		"includes the Weak Subjectivity Checkpoint: ")
	fmt.Printf("--weak-subjectivity-checkpoint=%s\n\n", od.WeakSubjectivity().CheckpointString())
	return nil
}
