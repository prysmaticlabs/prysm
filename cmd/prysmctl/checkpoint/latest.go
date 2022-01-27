package checkpoint

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/api/client/openapi"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/proto/sniff"
	"github.com/prysmaticlabs/prysm/time/slots"
	"io"
	"net"
	"net/url"
	"strconv"
	"time"

	//"time"

	//"github.com/prysmaticlabs/prysm/api/client/openapi"
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
	log.Printf("--beacon-node-url=%s", f.BeaconNodeHost)

	opts := make([]openapi.ClientOpt, 0)
	timeout, err := time.ParseDuration(f.Timeout)
	if err != nil {
		return err
	}
	opts = append(opts, openapi.WithTimeout(timeout))
	validatedHost, err := validHostname(latestFlags.BeaconNodeHost)
	if err != nil {
		return err
	}
	log.Printf("host:port=%s", validatedHost)
	client, err := openapi.NewClient(validatedHost, opts...)
	if err != nil {
		return err
	}
	stateReader, err := client.GetStateById(openapi.StateIdHead)
	stateBytes, err := io.ReadAll(stateReader)
	if err != nil {
		return errors.Wrap(err, "failed to read response body for get head state api call")
	}
	log.Printf("state response byte len=%d", len(stateBytes))
	state, err := sniff.BeaconState(stateBytes)
	if err != nil {
		return errors.Wrap(err, "error unmarshaling state to correct version")
	}
	cf, err := sniff.ConfigForkForState(stateBytes)
	if err != nil {
		return errors.Wrap(err, "error detecting chain config for beacon state")
	}
	params.OverrideBeaconConfig(cf.Config)
	epoch, err := helpers.LatestWeakSubjectivityEpoch(ctx, state)
	if err != nil {
		return errors.Wrap(err, "error computing the weak subjectivity epoch from head state")
	}
	bSlot, err := slots.EpochStart(epoch)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("Error computing first slot of epoch=%d", epoch))
	}
	root, err := client.GetBlockRoot(strconv.Itoa(int(bSlot)))
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("error requesting block root from api for slot=%d", bSlot))
	}
	wsFlag := fmt.Sprintf("--weak-subjectivity-checkpoint=%#x:%d", root, epoch)
	log.Printf("latest weak subjectivity checkpoint verification flag:\n%s", wsFlag)

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
