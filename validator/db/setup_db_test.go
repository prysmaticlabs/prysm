package db

import (
	"os"
	"path/filepath"
	"testing"
)

func TestClearDB(t *testing.T) {
	db := SetupDB(t, [][48]byte{})
	if err := db.ClearDB(); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(db.DatabasePath(), databaseFileName)); !os.IsNotExist(err) {
		t.Fatalf("DB was not cleared: %v", err)
	}
}
