package checkpoint

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	log "github.com/sirupsen/logrus"
)

type Initializer interface {
	Initialize(ctx context.Context, d db.Database) error
}

func NewFileInitializer(blockPath string, statePath string) (*FileInitializer, error) {
	var err error
	if err = existsAndIsFile(blockPath); err != nil {
		return nil, err
	}
	if err = existsAndIsFile(statePath); err != nil {
		return nil, err
	}
	// stat just to make sure it actually exists and is a file
	return &FileInitializer{blockPath: blockPath, statePath: statePath}, nil
}

// FileInitializer initializes a beacon-node database to use checkpoint sync,
// using ssz-encoded block and state data stored in files on the local filesystem.
type FileInitializer struct {
	blockPath string
	statePath string
}

// Initialize is called in the BeaconNode db startup code if an Initializer is present.
// Initialize does what is needed to prepare the beacon node database for syncing from the weak subjectivity checkpoint.
func (fi *FileInitializer) Initialize(ctx context.Context, d db.Database) error {
	blockFH, err := os.Open(fi.blockPath)
	if err != nil {
		return errors.Wrapf(err, "error opening block file %s for checkpoint sync initialization", fi.blockPath)
	}
	defer func() {
		err := blockFH.Close()
		if err != nil {
			log.Errorf("error while closing checkpoint block input stream: %s", err)
		}
	}()
	serBlock, err := ioutil.ReadAll(blockFH)
	if err != nil {
		return errors.Wrapf(err, "error reading block file %s for checkpoint sync initialization", fi.blockPath)
	}

	stateFH, err := os.Open(fi.statePath)
	if err != nil {
		return errors.Wrapf(err, "error reading state file %s for checkpoint sync initialization", fi.statePath)
	}
	defer func() {
		err := stateFH.Close()
		if err != nil {
			log.Errorf("error while closing checkpoint state input stream: %s", err)
		}
	}()
	serState, err := ioutil.ReadAll(stateFH)
	if err != nil {
		return errors.Wrapf(err, "error reading block file %s for checkpoint sync initialization", fi.statePath)
	}

	return d.SaveOrigin(ctx, serState, serBlock)
}

var _ Initializer = &FileInitializer{}

func existsAndIsFile(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return errors.Wrapf(err, "error checking existence of ssz-encoded file %s for checkpoint sync init", path)
	}
	if info.IsDir() {
		return fmt.Errorf("%s is a directory, please specify full path to file", path)
	}
	return nil
}
