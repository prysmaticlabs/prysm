package blockchain

import (
	"testing"
	"time"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	testDB "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
)

func TestVerifyCheckpointEpoch_Ok(t *testing.T) {
	helpers.ClearCache()
	db, sc := testDB.SetupDB(t)

	chainService := setupBeaconChain(t, db, sc)
	chainService.genesisTime = time.Now()

	assert.Equal(t, true, chainService.verifyCheckpointEpoch(&ethpb.Checkpoint{}))
	assert.Equal(t, false, chainService.verifyCheckpointEpoch(&ethpb.Checkpoint{Epoch: 1}))
}
