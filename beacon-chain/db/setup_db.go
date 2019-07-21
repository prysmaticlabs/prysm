package db

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"os"
	"path"
)

// SetupDB instantiates and returns a simulated backend BeaconDB instance.
func SetupDB() (*BeaconDB, error) {
	randPath, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return nil, fmt.Errorf("could not generate random file path: %v", err)
	}
	path := path.Join(os.TempDir(), fmt.Sprintf("/%d", randPath))
	if err := os.RemoveAll(path); err != nil {
		return nil, fmt.Errorf("failed to remove directory: %v", err)
	}
	return NewDB(path)
}

// TeardownDB cleans up a simulated backend BeaconDB instance.
func TeardownDB(db *BeaconDB) {
	if err := db.Close(); err != nil {
		log.Fatalf("failed to close database: %v", err)
	}
	if err := os.RemoveAll(db.DatabasePath); err != nil {
		log.Fatalf("could not remove tmp db dir: %v", err)
	}
}
