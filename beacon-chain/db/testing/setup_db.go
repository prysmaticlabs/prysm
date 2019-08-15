package testing

import (
	"crypto/rand"
	"fmt"
	"log"
	"math/big"
	"os"
	"path"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/beacon-chain/db/kv"
)

// SetupDB instantiates and returns database backed by key value store.
func SetupDB() (db.Database, error) {
	randPath, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return nil, errors.Wrap(err, "could not generate random file path")
	}
	p := path.Join(os.TempDir(), fmt.Sprintf("/%d", randPath))
	if err := os.RemoveAll(p); err != nil {
		return nil, errors.Wrap(err, "failed to remove directory")
	}
	return kv.NewKVStore(p)
}

// TeardownDB closes a database and destroyes the fails at the database path.
func TeardownDB(db db.Database) {
	if err := db.Close(); err != nil {
		log.Fatalf("failed to close database: %v", err)
	}
	if err := os.RemoveAll(db.DatabasePath()); err != nil {
		log.Fatalf("could not remove tmp db dir: %v", err)
	}
}
