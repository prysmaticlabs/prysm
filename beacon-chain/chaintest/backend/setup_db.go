package backend

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"os"
	"path"

	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	log "github.com/sirupsen/logrus"
)

// setupDB instantiates and returns a simulated backend BeaconDB instance.
func setupDB() (*db.BeaconDB, error) {
	randPath, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return nil, fmt.Errorf("could not generate random file path: %v", err)
	}
	path := path.Join(os.TempDir(), fmt.Sprintf("/%d", randPath))
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
	if err := os.RemoveAll(db.DatabasePath); err != nil {
		log.Fatalf("could not remove tmp db dir: %v", err)
	}
}
