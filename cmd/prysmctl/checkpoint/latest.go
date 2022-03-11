package checkpoint

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"time"

	"github.com/prysmaticlabs/prysm/api/client/openapi"
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

func setupClient() (*openapi.Client, error) {
	f := latestFlags
	log.Printf("--beacon-node-url=%s", f.BeaconNodeHost)

	opts := make([]openapi.ClientOpt, 0)
	timeout, err := time.ParseDuration(f.Timeout)
	if err != nil {
		return nil, err
	}
	opts = append(opts, openapi.WithTimeout(timeout))
	validatedHost, err := validHostname(latestFlags.BeaconNodeHost)
	if err != nil {
		return nil, err
	}
	log.Printf("host:port=%s", validatedHost)
	client, err := openapi.NewClient(validatedHost, opts...)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func cliActionLatest(_ *cli.Context) error {
	ctx := context.Background()
	client, err := setupClient()
	if err != nil {
		log.Fatalf(err.Error())
	}
	od, err := openapi.DownloadOriginData(ctx, client)
	if err != nil {
		log.Fatalf(err.Error())
	}
	ws := od.WeakSubjectivity()
	fmt.Println("\nUse the following flag when starting a prysm Beacon Node to ensure the chain history " +
		"includes the Weak Subjectivity Checkpoint:")
	fmt.Printf("--weak-subjectivity-checkpoint=%#x:%d\n\n", ws.BlockRoot, ws.Epoch)
	return nil
}

func validHostname(h string) (string, error) {
	// try to parse as url (being permissive)
	u, err := url.Parse(h)
	if err == nil && u.Host != "" {
		return u.Host, nil
	}
	// try to parse as host:port
	host, port, err := net.SplitHostPort(h)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s:%s", host, port), nil
}
