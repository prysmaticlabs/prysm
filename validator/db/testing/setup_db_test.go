package testing

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"testing"

	"github.com/prysmaticlabs/prysm/validator/db/kv"
)

func TestClearDB(t *testing.T) {
	// Setting up manually is required, since SetupDB() will also register a teardown procedure.
	randPath, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		t.Fatalf("Could not generate random file path: %v", err)
	}
	p := filepath.Join(TempDir(), fmt.Sprintf("/%d", randPath))
	if err := os.RemoveAll(p); err != nil {
		t.Fatalf("Failed to remove directory: %v", err)
	}
	testDB, err := kv.NewKVStore(p, [][48]byte{})
	if err != nil {
		t.Fatalf("Failed to instantiate DB: %v", err)
	}
	if err := testDB.ClearDB(); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(testDB.DatabasePath(), "validator.db")); !os.IsNotExist(err) {
		t.Fatalf("DB was not cleared: %v", err)
	}
}
