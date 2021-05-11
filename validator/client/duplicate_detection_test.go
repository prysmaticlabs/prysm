package client

import (
	"context"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/shared/mock"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/timeutils"

	slotutil "github.com/prysmaticlabs/prysm/shared/slotutil"
	dbTest "github.com/prysmaticlabs/prysm/validator/db/testing"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestSleeping_ForDuplicateDetectionEpochs(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	client := mock.NewMockBeaconNodeValidatorClient(ctrl)
	db := dbTest.SetupDB(t, [][48]byte{})

	// Set current config to minimal config
	params.OverrideBeaconConfig(params.MinimalSpecConfig())
	slot := types.Slot(300)
	currentEpoch := helpers.SlotToEpoch(slot)
	genesisTime := uint64(timeutils.Now().Unix()) - uint64(slot.Mul(params.BeaconConfig().SecondsPerSlot))

	ticker := slotutil.NewSlotTicker(time.Unix(int64(genesisTime), 0),
		params.BeaconConfig().SecondsPerSlot)

	defer ticker.Done()
	v := validator{
		validatorClient: client,
		db:              db,
		ticker:          ticker,
		genesisTime:     genesisTime,
	}

	err := v.startDoppelgangerService(ctx)
	oneEpochs := helpers.SlotToEpoch(<-v.NextSlot())
	require.NoError(t, err)
	require.Equal(t, currentEpoch.Add(uint64(params.BeaconConfig().DuplicateValidatorEpochsCheck)),
		oneEpochs, "Initial Epoch (%d) vs After 1 epochs (%d)", currentEpoch, oneEpochs)
}
