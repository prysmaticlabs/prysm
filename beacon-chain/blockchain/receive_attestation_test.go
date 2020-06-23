package blockchain

import (
	"testing"
	"time"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	testDB "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
)

func TestVerifyCheckpointEpoch_Ok(t *testing.T) {
	helpers.ClearCache()
	db, sc := testDB.SetupDB(t)

	chainService := setupBeaconChain(t, db, sc)
	chainService.genesisTime = time.Now()

	if !chainService.verifyCheckpointEpoch(&ethpb.Checkpoint{}) {
		t.Error("Wanted true, got false")
	}

	if chainService.verifyCheckpointEpoch(&ethpb.Checkpoint{Epoch: 1}) {
		t.Error("Wanted false, got true")
	}
}
