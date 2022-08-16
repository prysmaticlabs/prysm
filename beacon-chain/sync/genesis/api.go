package genesis

import (
	"context"

	log "github.com/sirupsen/logrus"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v3/api/client/beacon"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/db"
)

// APIInitializer manages initializing the genesis state and block to prepare the beacon node for syncing.
// The genesis state is retrieved from the remote beacon node api, using the debug state retrieval endpoint.
type APIInitializer struct {
	c *beacon.Client
}

// NewAPIInitializer creates an APIInitializer, handling the set up of a beacon node api client
// using the provided host string.
func NewAPIInitializer(beaconNodeHost string) (*APIInitializer, error) {
	c, err := beacon.NewClient(beaconNodeHost)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to parse beacon node url or hostname - %s", beaconNodeHost)
	}
	return &APIInitializer{c: c}, nil
}

// Initialize downloads origin state and block for checkpoint sync and initializes database records to
// prepare the node to begin syncing from that point.
func (dl *APIInitializer) Initialize(ctx context.Context, d db.Database) error {
	existing, err := d.GenesisState(ctx)
	if err != nil {
		return err
	}
	if existing != nil && !existing.IsNil() {
		htr, err := existing.HashTreeRoot(ctx)
		if err != nil {
			return errors.Wrap(err, "error while computing hash_tree_root of existing genesis state")
		}
		log.Warnf("database contains genesis with htr=%#x, ignoring remote genesis state parameter", htr)
		return nil
	}
	sb, err := dl.c.GetState(ctx, beacon.IdGenesis)
	if err != nil {
		return errors.Wrapf(err, "Error retrieving genesis state from %s", dl.c.NodeURL())
	}
	return d.LoadGenesis(ctx, sb)
}
