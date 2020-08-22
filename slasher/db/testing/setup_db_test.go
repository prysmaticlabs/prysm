package testing

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"os"
	"path"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	slasherDB "github.com/prysmaticlabs/prysm/slasher/db"
	"github.com/prysmaticlabs/prysm/slasher/db/kv"
)

func TestClearDB(t *testing.T) {
	// Setting up manually is required, since SetupDB() will also register a teardown procedure.
	randPath, err := rand.Int(rand.Reader, big.NewInt(1000000))
	require.NoError(t, err, "Could not generate random file path")
	p := path.Join(testutil.TempDir(), fmt.Sprintf("/%d", randPath))
	require.NoError(t, os.RemoveAll(p), "Failed to remove directory")
	cfg := &kv.Config{}
	db, err := slasherDB.NewDB(p, cfg)
	require.NoError(t, err, "Failed to instantiate DB")
	db.EnableSpanCache(false)
	require.NoError(t, db.ClearDB())
	_, err = os.Stat(db.DatabasePath())
	require.Equal(t, true, os.IsNotExist(err), "Db wasnt cleared %v", err)
}
