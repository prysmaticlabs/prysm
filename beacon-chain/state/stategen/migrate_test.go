package stategen

import (
	"context"
	"testing"

	testDB "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func TestMigrateToCold_NoBlock(t *testing.T) {
	hook := logTest.NewGlobal()
	ctx := context.Background()
	db := testDB.SetupDB(t)
	defer testDB.TeardownDB(t, db)

	service := New(db)

	beaconState, _ := testutil.DeterministicGenesisState(t, 32)
	if err := service.MigrateToCold(ctx, beaconState, [32]byte{}); err != nil {
		t.Fatal(err)
	}

	testutil.AssertLogsContain(t, hook, "Set hot and cold state split point")
}