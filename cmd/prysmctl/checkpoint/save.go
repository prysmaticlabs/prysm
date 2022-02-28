package checkpoint

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/pkg/errors"
	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/api/client/openapi"
	"github.com/prysmaticlabs/prysm/proto/sniff"
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
	opts := make([]openapi.ClientOpt, 0)
	log.Printf("--beacon-node-url=%s", f.BeaconNodeHost)
	timeout, err := time.ParseDuration(f.Timeout)
	if err != nil {
		return err
	}
	opts = append(opts, openapi.WithTimeout(timeout))
	client, err := openapi.NewClient(saveFlags.BeaconNodeHost, opts...)
	if err != nil {
		return err
	}

	return saveCheckpoint(client)
}

func saveCheckpoint(client *openapi.Client) error {
	ctx := context.Background()

	od, err := openapi.DownloadOriginData(ctx, client)
	if err != nil {
		log.Fatalf(err.Error())
	}

	blockPath := fname("block", od.ConfigFork, od.Block.Block().Slot(), od.WeakSubjectivity.BlockRoot)
	log.Printf("saving ssz-encoded block to to %s", blockPath)
	err = os.WriteFile(blockPath, od.BlockBytes, 0600)
	if err != nil {
		return err
	}

	stateRoot, err := od.State.HashTreeRoot(ctx)
	if err != nil {
		return errors.Wrap(err, "Could not compute HTR of state downloaded from remote beacon node")
	}
	statePath := fname("state", od.ConfigFork, od.State.Slot(), stateRoot)
	log.Printf("saving ssz-encoded state to to %s", statePath)
	err = os.WriteFile(statePath, od.StateBytes, 0600)
	if err != nil {
		return err
	}

	return nil
}

func fname(prefix string, cf *sniff.ConfigFork, slot types.Slot, root [32]byte) string {
	return fmt.Sprintf("%s_%s_%s_%d-%#x.ssz", prefix, cf.ConfigName.String(), cf.Fork.String(), slot, root)
}
