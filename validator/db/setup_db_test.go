package db

import (
	"github.com/prysmaticlabs/prysm/shared/params"
	"os"
	"path/filepath"
	"testing"
)

func TestClearDB(t *testing.T) {
	db := SetupDB(t, []params.KeyBytes{})
	if err := db.ClearDB(); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(db.DatabasePath(), databaseFileName)); !os.IsNotExist(err) {
		t.Fatalf("DB was not cleared: %v", err)
	}
}
