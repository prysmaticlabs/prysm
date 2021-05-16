package client

import (
	"context"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/mock"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/timeutils"

	slotutil "github.com/prysmaticlabs/prysm/shared/slotutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	dbTest "github.com/prysmaticlabs/prysm/validator/db/testing"
)

func TestSleeping_ForDuplicateDetectionEpochs(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
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

	// Set random Keys
	validatorKey, err := bls.RandKey()
	pubKey := [48]byte{}
	require.NoError(t, err)
	copy(pubKey[:], validatorKey.PublicKey().Marshal())
	km := &mockKeymanager{
		keysMap: map[[48]byte]bls.SecretKey{
			pubKey: validatorKey,
		},
	}

	//km, err := w.InitializeKeymanager(ctx, iface.InitKeymanagerConfig{ListenForChanges: false})
	//	require.NoError(t, err)
	//	s := &Server{
	//		keymanager:                km,
	//		walletInitialized:         true,
	//		wallet:                    w,
	//		beaconNodeClient:          mockNodeClient,
	//		beaconNodeValidatorClient: mockValidatorClient,
	//	}
	//	numAccounts := 2
	//	dr, ok := km.(*derived.Keymanager)
	//	require.Equal(t, true, ok)
	//	err = dr.RecoverAccountsFromMnemonic(ctx, constant.TestMnemonic, "", numAccounts)
	//	require.NoError(t, err)
	//	pubKeys, err := dr.FetchValidatingPublicKeys(ctx)
	//	require.NoError(t, err)
	//
	//	rawPubKeys := make([][]byte, len(pubKeys))
	//	for i, key := range pubKeys {
	//		rawPubKeys[i] = key[:]
	//	}

	v := validator{
		validatorClient: client,
		db:              db,
		ticker:          ticker,
		genesisTime:     genesisTime,
		keyManager: km,
	}

	err = v.startDoppelgangerService(ctx)
	oneEpochs := helpers.SlotToEpoch(<-v.NextSlot())
	require.NoError(t, err)
	require.Equal(t, currentEpoch.Add(uint64(params.BeaconConfig().DuplicateValidatorEpochsCheck)),
		oneEpochs, "Initial Epoch (%d) vs After 1 epochs (%d)", currentEpoch, oneEpochs)
	ctrl.Finish()
}
