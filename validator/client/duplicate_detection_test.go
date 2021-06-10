package client

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bls"
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

func TestSleeping_DuplicateDetection_WithKeys(t *testing.T) {
	ctrl := gomock.NewController(t)
	ctx := context.Background()
	defer ctrl.Finish()

	client := mock.NewMockBeaconNodeValidatorClient(ctrl)
	db := dbTest.SetupDB(t, [][48]byte{})

	// Set random Keys
	privKey, err := bls.RandKey()
	require.NoError(t, err)
	pubKey := [48]byte{}
	copy(pubKey[:], privKey.PublicKey().Marshal())
	km := &mockKeymanager{
		fetchNoKeys: false,
		keysMap: map[[48]byte]bls.SecretKey{
			pubKey: privKey,
		},
	}
	client.EXPECT().DetectDoppelganger(
		gomock.Any(), // ctx
		gomock.Any(), // keys, targets
	).Return(&ethpb.DetectDoppelgangerResponse{PublicKey: pubKey[:]}, nil)

	v := validator{
		validatorClient: client,
		db:              db,
		keyManager:      km,
	}

	key, err := v.DoppelgangerService(ctx)
	//oneEpochs := helpers.SlotToEpoch(<-v.NextSlot())
	require.NoError(t, err)
	require.Equal(t, len(pubKey), len(key))
}
