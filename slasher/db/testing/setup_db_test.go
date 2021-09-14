package testing

import (
	"os"
	"path/filepath"
	"testing"

	slasherDB "github.com/prysmaticlabs/prysm/slasher/db"
	"github.com/prysmaticlabs/prysm/slasher/db/kv"
	"github.com/prysmaticlabs/prysm/testing/require"
)

func TestClearDB(t *testing.T) {
	// Setting up manually is required, since SetupDB() will also register a teardown procedure.
	cfg := &kv.Config{}
	db, err := slasherDB.NewDB(t.TempDir(), cfg)
	require.NoError(t, err, "Failed to instantiate DB")
	db.EnableSpanCache(false)
	require.NoError(t, db.ClearDB())
	_, err = os.Stat(filepath.Join(db.DatabasePath(), "slasher.db"))
	require.Equal(t, true, os.IsNotExist(err), "Db wasnt cleared %v", err)
}
