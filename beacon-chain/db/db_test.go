package db

import (
	"os"
	"strings"
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/db/kv"
	testutil "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
)

var _ = Database(&kv.Store{})

func TestClearDB(t *testing.T) {
	beaconDB := testutil.SetupDB(t)
	defer testutil.TeardownDB(t, beaconDB)
	path := strings.TrimSuffix(beaconDB.DatabasePath(), "beaconchain.db")
	if err := ClearDB(path); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(beaconDB.DatabasePath()); !os.IsNotExist(err) {
		t.Fatalf("db wasnt cleared %v", err)
	}
}
