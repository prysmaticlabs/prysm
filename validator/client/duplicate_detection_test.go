package client

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"

	"github.com/prysmaticlabs/prysm/shared/mock"
	dbTest "github.com/prysmaticlabs/prysm/validator/db/testing"
)

func TestSleeping_DuplicateDetection_NoKeys(t *testing.T) {
	ctrl := gomock.NewController(t)
	ctx := context.Background()
	defer ctrl.Finish()

	client := mock.NewMockBeaconNodeValidatorClient(ctrl)
	db := dbTest.SetupDB(t, [][48]byte{})

	client.EXPECT().DetectDoppelganger(
		gomock.Any(), // ctx
		gomock.Any(), // keys, targets
	).Return(&ethpb.DetectDoppelgangerResponse{PublicKey: make([]byte, 0)}, nil /*err*/)

	km := &mockKeymanager{
		fetchNoKeys: true,
	}

	v := validator{
		validatorClient: client,
		db:              db,
		keyManager:      km,
	}

	// Fetch keys from the mock KM does not error. So just an aesthetic check of zero length.
	key, err := v.DoppelgangerService(ctx)
	require.NoError(t, err)
	require.Equal(t, 0, len(key))
}

/*
func TestSleeping_DuplicateDetection(t *testing.T) {
	ctrl := gomock.NewController(t)
	ctx := context.Background()
	defer ctrl.Finish()

	client := mock.NewMockBeaconNodeValidatorClient(ctrl)
	db := dbTest.SetupDB(t, [][48]byte{})

	client.EXPECT().DetectDoppelganger(
		gomock.Any(), // ctx
		gomock.Any(), // epoch
	).Return(&ethpb.DetectDoppelgangerResponse{PublicKey: make([]byte, 0)}, nil )

	// Set random Keys
	//validatorKey, err := bls.RandKey()
	//pubKey := [48]byte{}
	//require.NoError(t, err)
	//copy(pubKey[:], validatorKey.PublicKey().Marshal())
	km := &mockKeymanager{
		fetchNoKeys: true,
	}

	//v := &testutil.FakeValidator{
	//	NextSlotRet:        ticker.C(),
	//	DuplicateCheckFlag: true,
	//	Keymanager:         km,
	//}

	v := validator{
		validatorClient: client,
		db:              db,
		keyManager:      km,
	}

	key, err := v.DoppelgangerService(ctx)
	//oneEpochs := helpers.SlotToEpoch(<-v.NextSlot())
	require.NoError(t, err)
	require.Equal(t, 0, len(key))
	//
	//require.Equal(t, currentEpoch.Add(uint64(params.BeaconConfig().DuplicateValidatorEpochsCheck)),
	//oneEpochs, "Initial Epoch (%d) vs After 1 epochs (%d)", currentEpoch, oneEpochs)
	//assert.ErrorContains(t, "Doppelganger detection - failed to retrieve validator keys and indices", err)
}
*/
