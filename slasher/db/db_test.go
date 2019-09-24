package db

import (
	"os"
	"testing"

	testdb "github.com/prysmaticlabs/prysm/slasher/db/testing"
)

func TestClearDB(t *testing.T) {
	slasherDB := testdb.SetupSlasherDB(t)
	defer testdb.TeardownSlasherDB(t, slasherDB)
	if err := slasherDB.ClearDB(); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(slasherDB.DatabasePath()); !os.IsNotExist(err) {
		t.Fatalf("db wasnt cleared %v", err)
	}
}
