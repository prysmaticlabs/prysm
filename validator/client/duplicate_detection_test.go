package client

import (
	"context"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/slotutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	"github.com/prysmaticlabs/prysm/shared/timeutils"
	"github.com/prysmaticlabs/prysm/validator/client/testutil"
)

func Test_DuplicateDetectionEpochsByBalance(t *testing.T) {

	ctrl := gomock.NewController(t)
	ctx := context.Background()
	defer ctrl.Finish()

	// Set current config to minimal config
	params.OverrideBeaconConfig(params.MinimalSpecConfig())
	slot := types.Slot(240)
	currentEpoch := helpers.SlotToEpoch(slot)
	genesisTime := uint64(timeutils.Now().Unix()) - uint64(slot.Mul(params.BeaconConfig().SecondsPerSlot))

	ticker := slotutil.NewSlotTicker(time.Unix(int64(genesisTime), 0), params.BeaconConfig().SecondsPerSlot)

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

	v := &testutil.FakeValidator{
		NextSlotRet:        ticker.C(),
		DuplicateCheckFlag: true,
		Keymanager:         km,
	}

	err = v.StartDoppelgangerService(ctx)
	oneEpochs := helpers.SlotToEpoch(<-v.NextSlot())
	require.NoError(t, err)
	require.Equal(t, currentEpoch.Add(uint64(params.BeaconConfig().DuplicateValidatorEpochsCheck)),
		oneEpochs, "Initial Epoch (%d) vs After 1 epochs (%d)", currentEpoch, oneEpochs)
	/*
	beaconDB := dbTest.SetupDB(t)

	validators, _, headState := setupValidators(t, beaconDB, 100)
	want := make([]*ethpb.Validators_ValidatorContainer, len(validators))
	for i := 0; i < len(validators); i++ {
		want[i] = &ethpb.Validators_ValidatorContainer{
			Index:     types.ValidatorIndex(i),
			Validator: validators[i],
		}
	}

	bs := &Server{
		HeadFetcher: &mock.ChainService{
			State: headState,
		},
		GenesisTimeFetcher: &mock.ChainService{
			// We are in epoch 0.
			Genesis: time.Now(),
		},
		FinalizationFetcher: &mock.ChainService{
			FinalizedCheckPoint: &ethpb.Checkpoint{
				Epoch: 0,
			},
		},
		StateGen: stategen.New(beaconDB),
	}

	received, err := bs.ListValidators(context.Background(), &ethpb.ListValidatorsRequest{})
	require.NoError(t, err)
	assert.DeepSSZEqual(t, want, received.ValidatorList, "Incorrect respond of validators")
	*/
}
/*
func TestSleeping_ForDuplicateDetectionEpochs(t *testing.T) {
	ctrl := gomock.NewController(t)
	ctx := context.Background()
	defer ctrl.Finish()

	// Set current config to minimal config
	params.OverrideBeaconConfig(params.MinimalSpecConfig())
	slot := types.Slot(240)
	currentEpoch := helpers.SlotToEpoch(slot)
	genesisTime := uint64(timeutils.Now().Unix()) - uint64(slot.Mul(params.BeaconConfig().SecondsPerSlot))

	ticker := slotutil.NewSlotTicker(time.Unix(int64(genesisTime), 0), params.BeaconConfig().SecondsPerSlot)

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

	v := &testutil.FakeValidator{
		NextSlotRet:        ticker.C(),
		DuplicateCheckFlag: true,
		Keymanager:         km,
	}

	err = v.StartDoppelgangerService(ctx)
	oneEpochs := helpers.SlotToEpoch(<-v.NextSlot())
	require.NoError(t, err)
	require.Equal(t, currentEpoch.Add(uint64(params.BeaconConfig().DuplicateValidatorEpochsCheck)),
		oneEpochs, "Initial Epoch (%d) vs After 1 epochs (%d)", currentEpoch, oneEpochs)
}
*/

