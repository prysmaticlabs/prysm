package checkpoint

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/api/client/openapi"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"io"
)

type APIInitializer struct {
	c *openapi.Client
}

func NewAPIInitializer(beaconNodeHost string) (*APIInitializer, error) {
	c, err := openapi.NewClient(beaconNodeHost)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("unable to parse beacon node url or hostname - %s", beaconNodeHost))
	}
	return &APIInitializer{c: c}, nil
}

func (dl *APIInitializer) StateReader(ctx context.Context) (io.Reader, error) {
	return nil, nil
}

func (dl *APIInitializer) BlockReader(ctx context.Context) (io.Reader, error) {
	return nil, nil
}

func (dl *APIInitializer) Initialize(ctx context.Context, d db.Database) error {
	od, err := openapi.DownloadOriginData(ctx, dl.c)
	if err != nil {
		return errors.Wrap(err, "Error retrieving checkpoint origin state and block")
	}
	return d.SaveOrigin(ctx, od.StateBytes, od.BlockBytes)
}
