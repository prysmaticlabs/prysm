package checkpoint

import (
	"context"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	log "github.com/sirupsen/logrus"
	"io"
)

// Initializer holds io.ReadClosers for the block + state needed to initialize a beacon-node database
// to begin syncing from a weak subjectivity checkpoint block.
type Initializer struct {
	BlockReadCloser io.ReadCloser
	StateReadCloser io.ReadCloser
}

// Initialize is called in the BeaconNode db startup code if an Initializer is present.
// Initialize does what is needed to prepare the beacon node database for syncing from the weak subjectivity checkpoint.
func (ini *Initializer) Initialize(ctx context.Context, d db.Database) error {
	defer func() {
		err := ini.BlockReadCloser.Close()
		if err != nil {
			log.Errorf("error while closing checkpoint block input stream: %s", err)
		}
	}()
	defer func() {
		err := ini.StateReadCloser.Close()
		if err != nil {
			log.Errorf("error while closing checkpoint state input stream: %s", err)
		}
	}()
	return d.SaveOrigin(ctx, ini.StateReadCloser, ini.BlockReadCloser)
}
