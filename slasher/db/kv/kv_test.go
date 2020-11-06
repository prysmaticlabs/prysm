package kv

import (
	"crypto/rand"
	"fmt"
	"io/ioutil"
	"math/big"
	"os"
	"path"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	"github.com/sirupsen/logrus"
)

func TestMain(m *testing.M) {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetOutput(ioutil.Discard)

	os.Exit(m.Run())
}

func setupDB(t testing.TB) *Store {
	randPath, err := rand.Int(rand.Reader, big.NewInt(1000000))
	require.NoError(t, err, "Could not generate random file path")
	p := path.Join(testutil.TempDir(), fmt.Sprintf("/%d", randPath))
	require.NoError(t, os.RemoveAll(p), "Failed to remove directory")
	cfg := &Config{}
	db, err := NewKVStore(p, cfg)
	require.NoError(t, err, "Failed to instantiate DB")
	t.Cleanup(func() {
		require.NoError(t, db.Close(), "Failed to close database")
		require.NoError(t, os.RemoveAll(db.DatabasePath()), "Failed to remove directory")
	})
	return db
}

func setupDBDiffCacheSize(t testing.TB, cacheSize int) *Store {
	randPath, err := rand.Int(rand.Reader, big.NewInt(1000000))
	require.NoError(t, err, "Could not generate random file path")
	p := path.Join(testutil.TempDir(), fmt.Sprintf("/%d", randPath))
	require.NoError(t, os.RemoveAll(p), "Failed to remove directory")
	cfg := &Config{SpanCacheSize: cacheSize}
	db, err := NewKVStore(p, cfg)
	require.NoError(t, err, "Failed to instantiate DB")
	t.Cleanup(func() {
		require.NoError(t, db.Close(), "Failed to close database")
		require.NoError(t, os.RemoveAll(db.DatabasePath()), "Failed to remove directory")
	})
	return db
}
