package withdrawal

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"

	"github.com/ethereum/go-ethereum/log"
	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"
	"go.opencensus.io/trace"
	"gopkg.in/yaml.v2"
)

var withdrawalFlags = struct {
	BeaconNodeHost string
	File           string
}{}

var Commands = []*cli.Command{
	{
		Name:    "update-withdrawal-address",
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
	ctx := context.Background()
	apiPath := "/blsToExecutionAddress"
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
	u, err := url.ParseRequestURI(f.BeaconNodeHost)
	if err != nil {
		return errors.Wrap(err, "invalid format, unable to parse url")
	}
	if u.Scheme == "" || u.Host == "" {
		return fmt.Errorf("url must be in the format of http(s)://host:port url used: %v", f.BeaconNodeHost)
	}

	ctx, span := trace.StartSpan(ctx, "withdrawal.blsToExecutionAddress")
	defer span.End()

	fullpath := f.BeaconNodeHost + apiPath

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fullpath, bytes.NewBuffer(b)) //TODO:change this b
	if err != nil {
		return errors.Wrap(err, "invalid format, failed to create new Post Request Object")
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	//start := time.Now()
	resp, err := client.Do(req)
	//duration := time.Since(start)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API request to %s , responded with a status other than OK, status: %v", fullpath, resp.Status)
	}
	log.Info("Successfully published message to update withdrawal address.")
	return nil
}
