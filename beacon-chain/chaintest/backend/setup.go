package backend

import (
	"fmt"
	"os"
	"path"

	"github.com/prysmaticlabs/prysm/beacon-chain/db"
)

// setupDB instantiates and returns a simulated backend BeaconDB instance.
func setupDB() (*db.BeaconDB, error) {
	path := path.Join(os.TempDir(), "/simulateddb")
	if err := os.RemoveAll(path); err != nil {
		return nil, fmt.Errorf("failed to remove directory: %v", err)
	}

	return db.NewDB(path)
}

// teardownDB cleans up a simulated backend BeaconDB instance.
func teardownDB(db *db.BeaconDB) error {
	if err := db.Close(); err != nil {
		return fmt.Errorf("failed to close database: %v", err)
	}
	path := path.Join(os.TempDir(), "/simulateddb")
	return os.RemoveAll(path)
}
