package db

import (
	"os"
	"testing"
)

func TestClearDB(t *testing.T) {
	slasherDB, err := SetupSlasherDB()
	if err != nil {
		t.Fatal(err)
	}
	defer TeardownSlasherDB(t, slasherDB)
	if err := slasherDB.ClearDB(); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(slasherDB.DatabasePath()); !os.IsNotExist(err) {
		t.Fatalf("db wasnt cleared %v", err)
	}
}
