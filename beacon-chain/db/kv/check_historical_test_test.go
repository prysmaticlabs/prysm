package kv

import (
	"strings"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/params"
)

func TestVerifySlotsPerArchivePoint(t *testing.T) {
	db := setupDB(t)

	// This should set default to 2048.
	if err := db.verifySlotsPerArchivePoint(); err != nil {
		t.Fatal(err)
	}

	// This should not fail with default 2048.
	if err := db.verifySlotsPerArchivePoint(); err != nil {
		t.Fatal(err)
	}

	params.SetupTestConfigCleanup(t)
	config := params.BeaconConfig()
	config.SlotsPerArchivedPoint = 256
	params.OverrideBeaconConfig(config)

	// This should fail.
	msg := "could not update --slots-per-archive-point after it has been set"
	if err := db.verifySlotsPerArchivePoint(); err == nil || !strings.Contains(err.Error(), msg) {
		t.Error("Did not get wanted error")
	}
}
