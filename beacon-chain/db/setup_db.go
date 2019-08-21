package db

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"os"
	"path"

	"github.com/pkg/errors"
)

// SetupDB instantiates and returns a simulated backend BeaconDB instance.
// DEPRECATED: Use beacon-chain/db/testing.SetupDB
func SetupDB() (*BeaconDB, error) {
	randPath, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return nil, errors.Wrap(err, "could not generate random file path")
	}
	path := path.Join(os.TempDir(), fmt.Sprintf("/%d", randPath))
	if err := os.RemoveAll(path); err != nil {
		return nil, errors.Wrap(err, "failed to remove directory")
	}
	return NewDBDeprecated(path)
}
