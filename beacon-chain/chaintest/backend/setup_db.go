package backend

import (
	"fmt"
	"os"
	"path"

	log "github.com/sirupsen/logrus"

	"math/rand"

	"github.com/prysmaticlabs/prysm/beacon-chain/db"
)

// setupDB instantiates and returns a simulated backend BeaconDB instance.
func setupDB() (*db.BeaconDB, error) {
	path := path.Join(os.TempDir(), fmt.Sprintf("/%d", rand.Int()))
	if err := os.RemoveAll(path); err != nil {
		return nil, fmt.Errorf("failed to remove directory: %v", err)
	}

	return db.NewDB(path)
}

// teardownDB cleans up a simulated backend BeaconDB instance.
func teardownDB(db *db.BeaconDB) {
	if err := db.Close(); err != nil {
		log.Fatalf("failed to close database: %v", err)
	}
	path := path.Join(os.TempDir(), fmt.Sprintf("/%d", rand.Int()))
	if err := os.RemoveAll(path); err != nil {
		log.Fatalf("could not remove tmp dir: %v", err)
	}
}
