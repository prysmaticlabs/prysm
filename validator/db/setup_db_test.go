package db

import (
	"os"
	"testing"
)

func TestClearDB(t *testing.T) {
	slasherDB := SetupDB(t)
	defer TeardownDB(t, slasherDB)
	if err := slasherDB.ClearDB(); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(slasherDB.DatabasePath()); !os.IsNotExist(err) {
		t.Fatalf("db wasnt cleared %v", err)
	}
}
