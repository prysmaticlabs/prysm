package checkpoint

import (
	"context"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/api/client/beacon"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
)

type APIInitializer struct {
	c *beacon.Client
}

func NewAPIInitializer(beaconNodeHost string) (*APIInitializer, error) {
	c, err := beacon.NewClient(beaconNodeHost)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to parse beacon node url or hostname - %s", beaconNodeHost)
	}
	return &APIInitializer{c: c}, nil
}

func (dl *APIInitializer) Initialize(ctx context.Context, d db.Database) error {
	od, err := beacon.DownloadOriginData(ctx, dl.c)
	if err != nil {
		return errors.Wrap(err, "Error retrieving checkpoint origin state and block")
	}
	return d.SaveOrigin(ctx, od.StateBytes(), od.BlockBytes())
}
