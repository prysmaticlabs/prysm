package client

import (
	"context"
	//"errors"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/shared/mock"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/timeutils"

	//"github.com/prysmaticlabs/prysm/validator/client/iface"
	slotutilmock "github.com/prysmaticlabs/prysm/shared/slotutil/testing"
	dbTest "github.com/prysmaticlabs/prysm/validator/db/testing"

	//"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	//"github.com/prysmaticlabs/prysm/validator/client/testutil"
)

func TestSleeping_TwoEpochs(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	client := mock.NewMockBeaconNodeValidatorClient(ctrl)
	db := dbTest.SetupDB(t, [][48]byte{})

	// Set current config to minimal config
	params.OverrideBeaconConfig(params.MinimalSpecConfig())
	slot := types.Slot(300)
	currentEpoch := helpers.SlotToEpoch(slot)
	genesisTime := uint64(timeutils.Now().Unix()) - uint64(slot.Mul(params.BeaconConfig().SecondsPerSlot))

	tickerChan := make(chan types.Slot)
	ticker := &slotutilmock.MockTicker{
		Channel: tickerChan,
	}
	v := validator{
		validatorClient: client,
		db:              db,
		ticker:          ticker,
		genesisTime:     genesisTime,
	}

	go func() {
		tickerChan <- slot
		// Cancel after 2 epochs to avoid waiting on channel forever.
		time.Sleep(time.Duration(params.BeaconConfig().SecondsPerSlot *
			uint64(params.BeaconConfig().SlotsPerEpoch.Mul(
				uint64(params.BeaconConfig().DuplicateValidatorEpochsCheck)).Add(1))))
		cancel()

	}()
	err := v.startDoppelgangerService(ctx)
	twoEpochs := helpers.SlotToEpoch(<-v.NextSlot())
	require.NoError(t, err)
	require.Equal(t, currentEpoch.Add(2), twoEpochs, "Initial Epoch (%d) vs After 2 "+
		"epochs (%d)", currentEpoch, twoEpochs)
}
