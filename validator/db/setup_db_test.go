package db

import (
	"os"
	"testing"
)

func TestClearDB(t *testing.T) {
	db := SetupDB(t, [][48]byte{})
	defer TeardownDB(t, db)
	if err := db.ClearDB(); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(db.DatabasePath()); !os.IsNotExist(err) {
		t.Fatalf("DB was not cleared: %v", err)
	}
}
