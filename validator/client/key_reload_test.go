package client

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/mock"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	"github.com/prysmaticlabs/prysm/validator/client/testutil"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func TestValidator_HandleKeyReload(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	t.Run("active", func(t *testing.T) {
		hook := logTest.NewGlobal()

		inactivePrivKey, err := bls.RandKey()
		require.NoError(t, err)
		inactivePubKey := [48]byte{}
		copy(inactivePubKey[:], inactivePrivKey.PublicKey().Marshal())
		activePrivKey, err := bls.RandKey()
		require.NoError(t, err)
		activePubKey := [48]byte{}
		copy(activePubKey[:], activePrivKey.PublicKey().Marshal())
		km := &mockKeymanager{
			keysMap: map[[48]byte]bls.SecretKey{
				inactivePubKey: inactivePrivKey,
			},
		}
		client := mock.NewMockBeaconNodeValidatorClient(ctrl)
		v := validator{
			validatorClient: client,
			keyManager:      km,
			genesisTime:     1,
		}

		resp := testutil.GenerateMultipleValidatorStatusResponse([][]byte{inactivePubKey[:], activePubKey[:]})
		resp.Statuses[0].Status = ethpb.ValidatorStatus_UNKNOWN_STATUS
		resp.Statuses[1].Status = ethpb.ValidatorStatus_ACTIVE
		client.EXPECT().MultipleValidatorStatus(
			gomock.Any(),
			&ethpb.MultipleValidatorStatusRequest{
				PublicKeys: [][]byte{inactivePubKey[:], activePubKey[:]},
			},
		).Return(resp, nil)

		anyActive, err := v.HandleKeyReload(context.Background(), [][48]byte{inactivePubKey, activePubKey})
		require.NoError(t, err)
		assert.Equal(t, true, anyActive)
		assert.LogsContain(t, hook, "Waiting for deposit to be observed by beacon node")
		assert.LogsContain(t, hook, "Validator activated")
	})

	t.Run("no active", func(t *testing.T) {
		hook := logTest.NewGlobal()

		inactivePrivKey, err := bls.RandKey()
		require.NoError(t, err)
		inactivePubKey := [48]byte{}
		copy(inactivePubKey[:], inactivePrivKey.PublicKey().Marshal())
		km := &mockKeymanager{
			keysMap: map[[48]byte]bls.SecretKey{
				inactivePubKey: inactivePrivKey,
			},
		}
		client := mock.NewMockBeaconNodeValidatorClient(ctrl)
		v := validator{
			validatorClient: client,
			keyManager:      km,
			genesisTime:     1,
		}

		resp := testutil.GenerateMultipleValidatorStatusResponse([][]byte{inactivePubKey[:]})
		resp.Statuses[0].Status = ethpb.ValidatorStatus_UNKNOWN_STATUS
		client.EXPECT().MultipleValidatorStatus(
			gomock.Any(),
			&ethpb.MultipleValidatorStatusRequest{
				PublicKeys: [][]byte{inactivePubKey[:]},
			},
		).Return(resp, nil)

		anyActive, err := v.HandleKeyReload(context.Background(), [][48]byte{inactivePubKey})
		require.NoError(t, err)
		assert.Equal(t, false, anyActive)
		assert.LogsContain(t, hook, "Waiting for deposit to be observed by beacon node")
		assert.LogsDoNotContain(t, hook, "Validator activated")
	})

	t.Run("error when getting status", func(t *testing.T) {
		inactivePrivKey, err := bls.RandKey()
		require.NoError(t, err)
		inactivePubKey := [48]byte{}
		copy(inactivePubKey[:], inactivePrivKey.PublicKey().Marshal())
		km := &mockKeymanager{
			keysMap: map[[48]byte]bls.SecretKey{
				inactivePubKey: inactivePrivKey,
			},
		}
		client := mock.NewMockBeaconNodeValidatorClient(ctrl)
		v := validator{
			validatorClient: client,
			keyManager:      km,
			genesisTime:     1,
		}

		client.EXPECT().MultipleValidatorStatus(
			gomock.Any(),
			&ethpb.MultipleValidatorStatusRequest{
				PublicKeys: [][]byte{inactivePubKey[:]},
			},
		).Return(nil, errors.New("error"))

		_, err = v.HandleKeyReload(context.Background(), [][48]byte{inactivePubKey})
		assert.ErrorContains(t, "error", err)
	})
}
