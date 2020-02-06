package blockchain

import (
	"testing"
	"time"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	testDB "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
)

func TestVerifyCheckpointEpoch_Ok(t *testing.T) {
	db := testDB.SetupDB(t)
	defer testDB.TeardownDB(t, db)

	chainService := setupBeaconChain(t, db)
	chainService.genesisTime = time.Now()

	if !chainService.verifyCheckpointEpoch(&ethpb.Checkpoint{}) {
		t.Error("Wanted true, got false")
	}

	if chainService.verifyCheckpointEpoch(&ethpb.Checkpoint{Epoch: 1}) {
		t.Error("Wanted false, got true")
	}
}
