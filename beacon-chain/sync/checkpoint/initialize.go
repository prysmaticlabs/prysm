package checkpoint

import (
	"context"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	log "github.com/sirupsen/logrus"
	"io"
)

type Initializer struct {
	BlockReadCloser io.ReadCloser
	StateReadCloser io.ReadCloser
}

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