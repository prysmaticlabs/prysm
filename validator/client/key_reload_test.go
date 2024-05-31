package client

import (
	"context"
	"testing"

	gomock2 "github.com/golang/mock/gomock"
	validator2 "github.com/prysmaticlabs/prysm/v5/consensus-types/validator"
	"github.com/prysmaticlabs/prysm/v5/validator/client/iface"

	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	fieldparams "github.com/prysmaticlabs/prysm/v5/config/fieldparams"
	ethpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v5/testing/assert"
	"github.com/prysmaticlabs/prysm/v5/testing/require"
	validatormock "github.com/prysmaticlabs/prysm/v5/testing/validator-mock"
	"github.com/prysmaticlabs/prysm/v5/validator/client/testutil"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func TestValidator_HandleKeyReload(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	ctrl2 := gomock2.NewController(t)
	defer ctrl2.Finish()

	t.Run("active", func(t *testing.T) {
		hook := logTest.NewGlobal()

		inactive := randKeypair(t)
		active := randKeypair(t)

		client := validatormock.NewMockValidatorClient(ctrl2)
		chainClient := validatormock.NewMockChainClient(ctrl)
		prysmChainClient := validatormock.NewMockPrysmChainClient(ctrl)
		v := validator{
			validatorClient:  client,
			km:               newMockKeymanager(t, inactive),
			genesisTime:      1,
			chainClient:      chainClient,
			prysmChainClient: prysmChainClient,
		}

		resp := testutil.GenerateMultipleValidatorStatusResponse([][]byte{inactive.pub[:], active.pub[:]})
		resp.Statuses[0].Status = ethpb.ValidatorStatus_UNKNOWN_STATUS
		resp.Statuses[1].Status = ethpb.ValidatorStatus_ACTIVE
		client.EXPECT().MultipleValidatorStatus(
			gomock.Any(),
			&ethpb.MultipleValidatorStatusRequest{
				PublicKeys: [][]byte{inactive.pub[:], active.pub[:]},
			},
		).Return(resp, nil)
		prysmChainClient.EXPECT().GetValidatorCount(
			gomock.Any(),
			"head",
			[]validator2.Status{validator2.Active},
		).Return([]iface.ValidatorCount{}, nil)

		anyActive, err := v.HandleKeyReload(context.Background(), [][fieldparams.BLSPubkeyLength]byte{inactive.pub, active.pub})
		require.NoError(t, err)
		assert.Equal(t, true, anyActive)
		assert.LogsContain(t, hook, "Waiting for deposit to be observed by beacon node")
		assert.LogsContain(t, hook, "Validator activated")
	})

	t.Run("no active", func(t *testing.T) {
		hook := logTest.NewGlobal()

		client := validatormock.NewMockValidatorClient(ctrl2)
		chainClient := validatormock.NewMockChainClient(ctrl)
		prysmChainClient := validatormock.NewMockPrysmChainClient(ctrl)
		kp := randKeypair(t)
		v := validator{
			validatorClient:  client,
			km:               newMockKeymanager(t, kp),
			genesisTime:      1,
			chainClient:      chainClient,
			prysmChainClient: prysmChainClient,
		}

		resp := testutil.GenerateMultipleValidatorStatusResponse([][]byte{kp.pub[:]})
		resp.Statuses[0].Status = ethpb.ValidatorStatus_UNKNOWN_STATUS
		client.EXPECT().MultipleValidatorStatus(
			gomock.Any(),
			&ethpb.MultipleValidatorStatusRequest{
				PublicKeys: [][]byte{kp.pub[:]},
			},
		).Return(resp, nil)
		prysmChainClient.EXPECT().GetValidatorCount(
			gomock.Any(),
			"head",
			[]validator2.Status{validator2.Active},
		).Return([]iface.ValidatorCount{}, nil)

		anyActive, err := v.HandleKeyReload(context.Background(), [][fieldparams.BLSPubkeyLength]byte{kp.pub})
		require.NoError(t, err)
		assert.Equal(t, false, anyActive)
		assert.LogsContain(t, hook, "Waiting for deposit to be observed by beacon node")
		assert.LogsDoNotContain(t, hook, "Validator activated")
	})

	t.Run("error when getting status", func(t *testing.T) {
		kp := randKeypair(t)
		client := validatormock.NewMockValidatorClient(ctrl2)
		v := validator{
			validatorClient: client,
			km:              newMockKeymanager(t, kp),
			genesisTime:     1,
		}

		client.EXPECT().MultipleValidatorStatus(
			gomock.Any(),
			&ethpb.MultipleValidatorStatusRequest{
				PublicKeys: [][]byte{kp.pub[:]},
			},
		).Return(nil, errors.New("error"))

		_, err := v.HandleKeyReload(context.Background(), [][fieldparams.BLSPubkeyLength]byte{kp.pub})
		assert.ErrorContains(t, "error", err)
	})
}
