package kv

import (
	"io/ioutil"
	"testing"

	"github.com/prysmaticlabs/prysm/testing/require"
	"github.com/sirupsen/logrus"
)

func TestMain(m *testing.M) {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetOutput(ioutil.Discard)

	m.Run()
}

func setupDB(t testing.TB) *Store {
	cfg := &Config{}
	db, err := NewKVStore(t.TempDir(), cfg)
	require.NoError(t, err, "Failed to instantiate DB")
	t.Cleanup(func() {
		require.NoError(t, db.Close(), "Failed to close database")
	})
	return db
}
