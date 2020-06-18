package testing

import (
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/db"
)

// Cleanup clears and closes a database.
func Cleanup(t *testing.T, database db.Database) {
	if err := database.ClearDB(); err != nil {
		t.Error(err)
	}
	if err := database.Close(); err != nil {
		t.Error(err)
	}
}
