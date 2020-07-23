package testing

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	"github.com/prysmaticlabs/prysm/validator/db/kv"
)

func TestClearDB(t *testing.T) {
	// Setting up manually is required, since SetupDB() will also register a teardown procedure.
	randPath, err := rand.Int(rand.Reader, big.NewInt(1000000))
	require.NoError(t, err, "Could not generate random file path")
	p := filepath.Join(TempDir(), fmt.Sprintf("/%d", randPath))
	require.NoError(t, os.RemoveAll(p), "Failed to remove directory")
	testDB, err := kv.NewKVStore(p, [][48]byte{})
	require.NoError(t, err, "Failed to instantiate DB")
	require.NoError(t, testDB.ClearDB())

	if _, err := os.Stat(filepath.Join(testDB.DatabasePath(), "validator.db")); !os.IsNotExist(err) {
		t.Fatalf("DB was not cleared: %v", err)
	}
}
