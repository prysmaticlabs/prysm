package kv

import (
	"context"
	"io/ioutil"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	"github.com/sirupsen/logrus"
)

func TestMain(m *testing.M) {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetOutput(ioutil.Discard)

	m.Run()
}

// setupDB instantiates and returns a DB instance for the validator client.
func setupDB(t testing.TB, pubkeys [][48]byte) *Store {
	db, err := NewKVStore(context.Background(), t.TempDir(), &Config{
		PubKeys: pubkeys,
	})
	require.NoError(t, err, "Failed to instantiate DB")
	err = db.UpdatePublicKeysBuckets(pubkeys)
	require.NoError(t, err, "Failed to create old buckets for public keys")
	t.Cleanup(func() {
		require.NoError(t, db.Close(), "Failed to close database")
		require.NoError(t, db.ClearDB(), "Failed to clear database")
	})
	return db
}
