package blockchain

import (
	"context"
	"testing"
	"time"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	testDB "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestVerifyCheckpointEpoch_Ok(t *testing.T) {
	helpers.ClearCache()
	db, sc := testDB.SetupDB(t)

	chainService := setupBeaconChain(t, db, sc)
	chainService.genesisTime = time.Now()

	assert.Equal(t, true, chainService.verifyCheckpointEpoch(&ethpb.Checkpoint{Root: make([]byte, 32)}))
	assert.Equal(t, false, chainService.verifyCheckpointEpoch(&ethpb.Checkpoint{Epoch: 1}))
}

func TestAttestationPreState_FarFutureSlot(t *testing.T) {
	helpers.ClearCache()
	db, sc := testDB.SetupDB(t)

	chainService := setupBeaconChain(t, db, sc)
	chainService.genesisTime = time.Now()

	e := helpers.MaxSlotBuffer/params.BeaconConfig().SlotsPerEpoch + 1
	_, err := chainService.AttestationCheckPtInfo(context.Background(), &ethpb.Attestation{Data: &ethpb.AttestationData{Target: &ethpb.Checkpoint{Epoch: e}}})
	require.ErrorContains(t, "exceeds max allowed value relative to the local clock", err)
}

func TestAttestationCheckPtInfo_FarFutureSlot(t *testing.T) {
	helpers.ClearCache()
	db, sc := testDB.SetupDB(t)

	chainService := setupBeaconChain(t, db, sc)
	chainService.genesisTime = time.Now()

	e := helpers.MaxSlotBuffer/params.BeaconConfig().SlotsPerEpoch + 1
	_, err := chainService.AttestationPreState(context.Background(), &ethpb.Attestation{Data: &ethpb.AttestationData{Target: &ethpb.Checkpoint{Epoch: e}}})
	require.ErrorContains(t, "exceeds max allowed value relative to the local clock", err)
}
