package client

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	types "github.com/prysmaticlabs/eth2-types"
	fieldparams "github.com/prysmaticlabs/prysm/config/fieldparams"
	validator_service_config "github.com/prysmaticlabs/prysm/config/validator/service"
	"github.com/prysmaticlabs/prysm/crypto/bls"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/testing/assert"
	"github.com/prysmaticlabs/prysm/testing/mock"
	"github.com/prysmaticlabs/prysm/testing/require"
	slotutilmock "github.com/prysmaticlabs/prysm/time/slots/testing"
	walletMock "github.com/prysmaticlabs/prysm/validator/accounts/testing"
	"github.com/prysmaticlabs/prysm/validator/client/testutil"
	"github.com/prysmaticlabs/prysm/validator/keymanager/derived"
	remotekeymanagermock "github.com/prysmaticlabs/prysm/validator/keymanager/remote/mock"
	constant "github.com/prysmaticlabs/prysm/validator/testing"
	logTest "github.com/sirupsen/logrus/hooks/test"
	"github.com/tyler-smith/go-bip39"
	util "github.com/wealdtech/go-eth2-util"
)

func TestWaitActivation_ContextCanceled(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	client := mock.NewMockBeaconNodeValidatorClient(ctrl)
	privKey, err := bls.RandKey()
	require.NoError(t, err)
	pubKey := [fieldparams.BLSPubkeyLength]byte{}
	copy(pubKey[:], privKey.PublicKey().Marshal())
	km := &mockKeymanager{
		keysMap: map[[fieldparams.BLSPubkeyLength]byte]bls.SecretKey{
			pubKey: privKey,
		},
	}
	v := validator{
		validatorClient: client,
		keyManager:      km,
	}
	clientStream := mock.NewMockBeaconNodeValidator_WaitForActivationClient(ctrl)

	client.EXPECT().WaitForActivation(
		gomock.Any(),
		&ethpb.ValidatorActivationRequest{
			PublicKeys: [][]byte{pubKey[:]},
		},
	).Return(clientStream, nil)
	clientStream.EXPECT().Recv().Return(
		&ethpb.ValidatorActivationResponse{},
		nil,
	)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	assert.ErrorContains(t, cancelledCtx, v.WaitForActivation(ctx, nil))
}

func TestWaitActivation_StreamSetupFails_AttemptsToReconnect(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	client := mock.NewMockBeaconNodeValidatorClient(ctrl)
	privKey, err := bls.RandKey()
	require.NoError(t, err)
	pubKey := [fieldparams.BLSPubkeyLength]byte{}
	copy(pubKey[:], privKey.PublicKey().Marshal())
	km := &mockKeymanager{
		keysMap: map[[fieldparams.BLSPubkeyLength]byte]bls.SecretKey{
			pubKey: privKey,
		},
	}
	v := validator{
		validatorClient:        client,
		keyManager:             km,
		pubkeyToValidatorIndex: make(map[[fieldparams.BLSPubkeyLength]byte]types.ValidatorIndex),
		feeRecipientConfig: &validator_service_config.FeeRecipientConfig{
			ProposeConfig: nil,
			DefaultConfig: &validator_service_config.FeeRecipientOptions{
				FeeRecipient: common.HexToAddress("0x046Fb65722E7b2455043BFEBf6177F1D2e9738D9"),
			},
		},
	}
	v.pubkeyToValidatorIndex[pubKey] = 1
	clientStream := mock.NewMockBeaconNodeValidator_WaitForActivationClient(ctrl)
	client.EXPECT().WaitForActivation(
		gomock.Any(),
		&ethpb.ValidatorActivationRequest{
			PublicKeys: [][]byte{pubKey[:]},
		},
	).Return(clientStream, errors.New("failed stream")).Return(clientStream, nil)
	resp := generateMockStatusResponse([][]byte{pubKey[:]})
	resp.Statuses[0].Status.Status = ethpb.ValidatorStatus_ACTIVE
	clientStream.EXPECT().Recv().Return(resp, nil)
	assert.NoError(t, v.WaitForActivation(context.Background(), nil))
}

func TestWaitForActivation_ReceiveErrorFromStream_AttemptsReconnection(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	client := mock.NewMockBeaconNodeValidatorClient(ctrl)

	privKey, err := bls.RandKey()
	require.NoError(t, err)
	pubKey := [fieldparams.BLSPubkeyLength]byte{}
	copy(pubKey[:], privKey.PublicKey().Marshal())
	km := &mockKeymanager{
		keysMap: map[[fieldparams.BLSPubkeyLength]byte]bls.SecretKey{
			pubKey: privKey,
		},
	}
	v := validator{
		validatorClient:        client,
		keyManager:             km,
		pubkeyToValidatorIndex: make(map[[fieldparams.BLSPubkeyLength]byte]types.ValidatorIndex),
		feeRecipientConfig: &validator_service_config.FeeRecipientConfig{
			ProposeConfig: nil,
			DefaultConfig: &validator_service_config.FeeRecipientOptions{
				FeeRecipient: common.HexToAddress("0x046Fb65722E7b2455043BFEBf6177F1D2e9738D9"),
			},
		},
	}
	v.pubkeyToValidatorIndex[pubKey] = 1
	clientStream := mock.NewMockBeaconNodeValidator_WaitForActivationClient(ctrl)
	client.EXPECT().WaitForActivation(
		gomock.Any(),
		&ethpb.ValidatorActivationRequest{
			PublicKeys: [][]byte{pubKey[:]},
		},
	).Return(clientStream, nil)
	// A stream fails the first time, but succeeds the second time.
	resp := generateMockStatusResponse([][]byte{pubKey[:]})
	resp.Statuses[0].Status.Status = ethpb.ValidatorStatus_ACTIVE
	clientStream.EXPECT().Recv().Return(
		nil,
		errors.New("fails"),
	).Return(resp, nil)
	assert.NoError(t, v.WaitForActivation(context.Background(), nil))
}

func TestWaitActivation_LogsActivationEpochOK(t *testing.T) {
	hook := logTest.NewGlobal()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	client := mock.NewMockBeaconNodeValidatorClient(ctrl)
	privKey, err := bls.RandKey()
	require.NoError(t, err)
	pubKey := [fieldparams.BLSPubkeyLength]byte{}
	copy(pubKey[:], privKey.PublicKey().Marshal())
	km := &mockKeymanager{
		keysMap: map[[fieldparams.BLSPubkeyLength]byte]bls.SecretKey{
			pubKey: privKey,
		},
	}
	v := validator{
		validatorClient:        client,
		keyManager:             km,
		genesisTime:            1,
		pubkeyToValidatorIndex: make(map[[fieldparams.BLSPubkeyLength]byte]types.ValidatorIndex),
		feeRecipientConfig: &validator_service_config.FeeRecipientConfig{
			ProposeConfig: nil,
			DefaultConfig: &validator_service_config.FeeRecipientOptions{
				FeeRecipient: common.HexToAddress("0x046Fb65722E7b2455043BFEBf6177F1D2e9738D9"),
			},
		},
	}
	v.pubkeyToValidatorIndex[pubKey] = 1
	resp := generateMockStatusResponse([][]byte{pubKey[:]})
	resp.Statuses[0].Status.Status = ethpb.ValidatorStatus_ACTIVE
	clientStream := mock.NewMockBeaconNodeValidator_WaitForActivationClient(ctrl)
	client.EXPECT().WaitForActivation(
		gomock.Any(),
		&ethpb.ValidatorActivationRequest{
			PublicKeys: [][]byte{pubKey[:]},
		},
	).Return(clientStream, nil)
	clientStream.EXPECT().Recv().Return(
		resp,
		nil,
	)
	assert.NoError(t, v.WaitForActivation(context.Background(), nil), "Could not wait for activation")
	assert.LogsContain(t, hook, "Validator activated")
}

func TestWaitForActivation_Exiting(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	client := mock.NewMockBeaconNodeValidatorClient(ctrl)
	privKey, err := bls.RandKey()
	require.NoError(t, err)
	pubKey := [fieldparams.BLSPubkeyLength]byte{}
	copy(pubKey[:], privKey.PublicKey().Marshal())
	km := &mockKeymanager{
		keysMap: map[[fieldparams.BLSPubkeyLength]byte]bls.SecretKey{
			pubKey: privKey,
		},
	}
	v := validator{
		validatorClient:        client,
		keyManager:             km,
		genesisTime:            1,
		pubkeyToValidatorIndex: make(map[[fieldparams.BLSPubkeyLength]byte]types.ValidatorIndex),
		feeRecipientConfig: &validator_service_config.FeeRecipientConfig{
			ProposeConfig: nil,
			DefaultConfig: &validator_service_config.FeeRecipientOptions{
				FeeRecipient: common.HexToAddress("0x046Fb65722E7b2455043BFEBf6177F1D2e9738D9"),
			},
		},
	}
	v.pubkeyToValidatorIndex[pubKey] = 1
	resp := generateMockStatusResponse([][]byte{pubKey[:]})
	resp.Statuses[0].Status.Status = ethpb.ValidatorStatus_EXITING
	clientStream := mock.NewMockBeaconNodeValidator_WaitForActivationClient(ctrl)
	client.EXPECT().WaitForActivation(
		gomock.Any(),
		&ethpb.ValidatorActivationRequest{
			PublicKeys: [][]byte{pubKey[:]},
		},
	).Return(clientStream, nil)
	clientStream.EXPECT().Recv().Return(
		resp,
		nil,
	)
	assert.NoError(t, v.WaitForActivation(context.Background(), nil))
}

func TestWaitForActivation_RefetchKeys(t *testing.T) {
	originalPeriod := keyRefetchPeriod
	defer func() {
		keyRefetchPeriod = originalPeriod
	}()
	keyRefetchPeriod = 1 * time.Second

	hook := logTest.NewGlobal()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	client := mock.NewMockBeaconNodeValidatorClient(ctrl)
	privKey, err := bls.RandKey()
	require.NoError(t, err)
	pubKey := [fieldparams.BLSPubkeyLength]byte{}
	copy(pubKey[:], privKey.PublicKey().Marshal())
	km := &mockKeymanager{
		keysMap: map[[fieldparams.BLSPubkeyLength]byte]bls.SecretKey{
			pubKey: privKey,
		},
		fetchNoKeys: true,
	}
	v := validator{
		validatorClient:        client,
		keyManager:             km,
		genesisTime:            1,
		pubkeyToValidatorIndex: make(map[[fieldparams.BLSPubkeyLength]byte]types.ValidatorIndex),
		feeRecipientConfig: &validator_service_config.FeeRecipientConfig{
			ProposeConfig: nil,
			DefaultConfig: &validator_service_config.FeeRecipientOptions{
				FeeRecipient: common.HexToAddress("0x046Fb65722E7b2455043BFEBf6177F1D2e9738D9"),
			},
		},
	}
	v.pubkeyToValidatorIndex[pubKey] = 1
	resp := generateMockStatusResponse([][]byte{pubKey[:]})
	resp.Statuses[0].Status.Status = ethpb.ValidatorStatus_ACTIVE
	clientStream := mock.NewMockBeaconNodeValidator_WaitForActivationClient(ctrl)
	client.EXPECT().WaitForActivation(
		gomock.Any(),
		&ethpb.ValidatorActivationRequest{
			PublicKeys: [][]byte{pubKey[:]},
		},
	).Return(clientStream, nil)
	clientStream.EXPECT().Recv().Return(
		resp,
		nil,
	)
	assert.NoError(t, v.waitForActivation(context.Background(), make(chan [][fieldparams.BLSPubkeyLength]byte)), "Could not wait for activation")
	assert.LogsContain(t, hook, msgNoKeysFetched)
	assert.LogsContain(t, hook, "Validator activated")
}

// Regression test for a scenario where you start with an inactive key and then import an active key.
func TestWaitForActivation_AccountsChanged(t *testing.T) {
	hook := logTest.NewGlobal()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	t.Run("Imported keymanager", func(t *testing.T) {
		inactivePrivKey, err := bls.RandKey()
		require.NoError(t, err)
		inactivePubKey := [fieldparams.BLSPubkeyLength]byte{}
		copy(inactivePubKey[:], inactivePrivKey.PublicKey().Marshal())
		activePrivKey, err := bls.RandKey()
		require.NoError(t, err)
		activePubKey := [fieldparams.BLSPubkeyLength]byte{}
		copy(activePubKey[:], activePrivKey.PublicKey().Marshal())
		km := &mockKeymanager{
			keysMap: map[[fieldparams.BLSPubkeyLength]byte]bls.SecretKey{
				inactivePubKey: inactivePrivKey,
			},
		}
		client := mock.NewMockBeaconNodeValidatorClient(ctrl)
		v := validator{
			validatorClient:        client,
			keyManager:             km,
			genesisTime:            1,
			pubkeyToValidatorIndex: make(map[[fieldparams.BLSPubkeyLength]byte]types.ValidatorIndex),
			feeRecipientConfig: &validator_service_config.FeeRecipientConfig{
				ProposeConfig: nil,
				DefaultConfig: &validator_service_config.FeeRecipientOptions{
					FeeRecipient: common.HexToAddress("0x046Fb65722E7b2455043BFEBf6177F1D2e9738D9"),
				},
			},
		}
		v.pubkeyToValidatorIndex[inactivePubKey] = 1
		inactiveResp := generateMockStatusResponse([][]byte{inactivePubKey[:]})
		inactiveResp.Statuses[0].Status.Status = ethpb.ValidatorStatus_UNKNOWN_STATUS
		inactiveClientStream := mock.NewMockBeaconNodeValidator_WaitForActivationClient(ctrl)
		client.EXPECT().WaitForActivation(
			gomock.Any(),
			&ethpb.ValidatorActivationRequest{
				PublicKeys: [][]byte{inactivePubKey[:]},
			},
		).Return(inactiveClientStream, nil)
		inactiveClientStream.EXPECT().Recv().Return(
			inactiveResp,
			nil,
		).AnyTimes()

		activeResp := generateMockStatusResponse([][]byte{inactivePubKey[:], activePubKey[:]})
		activeResp.Statuses[0].Status.Status = ethpb.ValidatorStatus_UNKNOWN_STATUS
		activeResp.Statuses[1].Status.Status = ethpb.ValidatorStatus_ACTIVE
		activeClientStream := mock.NewMockBeaconNodeValidator_WaitForActivationClient(ctrl)
		client.EXPECT().WaitForActivation(
			gomock.Any(),
			&ethpb.ValidatorActivationRequest{
				PublicKeys: [][]byte{inactivePubKey[:], activePubKey[:]},
			},
		).Return(activeClientStream, nil)
		activeClientStream.EXPECT().Recv().Return(
			activeResp,
			nil,
		)

		go func() {
			// We add the active key into the keymanager and simulate a key refresh.
			time.Sleep(time.Second * 1)
			km.keysMap[activePubKey] = activePrivKey
			v.pubkeyToValidatorIndex[activePubKey] = 1
			km.SimulateAccountChanges(make([][fieldparams.BLSPubkeyLength]byte, 0))
		}()

		assert.NoError(t, v.WaitForActivation(context.Background(), nil))
		assert.LogsContain(t, hook, "Waiting for deposit to be observed by beacon node")
		assert.LogsContain(t, hook, "Validator activated")
	})

	t.Run("Derived keymanager", func(t *testing.T) {
		seed := bip39.NewSeed(constant.TestMnemonic, "")
		inactivePrivKey, err :=
			util.PrivateKeyFromSeedAndPath(seed, fmt.Sprintf(derived.ValidatingKeyDerivationPathTemplate, 0))
		require.NoError(t, err)
		inactivePubKey := [fieldparams.BLSPubkeyLength]byte{}
		copy(inactivePubKey[:], inactivePrivKey.PublicKey().Marshal())
		activePrivKey, err :=
			util.PrivateKeyFromSeedAndPath(seed, fmt.Sprintf(derived.ValidatingKeyDerivationPathTemplate, 1))
		require.NoError(t, err)
		activePubKey := [fieldparams.BLSPubkeyLength]byte{}
		copy(activePubKey[:], activePrivKey.PublicKey().Marshal())
		wallet := &walletMock.Wallet{
			Files:            make(map[string]map[string][]byte),
			AccountPasswords: make(map[string]string),
			WalletPassword:   "secretPassw0rd$1999",
		}
		ctx := context.Background()
		km, err := derived.NewKeymanager(ctx, &derived.SetupConfig{
			Wallet:           wallet,
			ListenForChanges: true,
		})
		require.NoError(t, err)
		err = km.RecoverAccountsFromMnemonic(ctx, constant.TestMnemonic, "", 1)
		require.NoError(t, err)
		client := mock.NewMockBeaconNodeValidatorClient(ctrl)
		v := validator{
			validatorClient:        client,
			keyManager:             km,
			genesisTime:            1,
			pubkeyToValidatorIndex: make(map[[fieldparams.BLSPubkeyLength]byte]types.ValidatorIndex),
			feeRecipientConfig: &validator_service_config.FeeRecipientConfig{
				ProposeConfig: nil,
				DefaultConfig: &validator_service_config.FeeRecipientOptions{
					FeeRecipient: common.HexToAddress("0x046Fb65722E7b2455043BFEBf6177F1D2e9738D9"),
				},
			},
		}
		v.pubkeyToValidatorIndex[inactivePubKey] = 1

		inactiveResp := generateMockStatusResponse([][]byte{inactivePubKey[:]})
		inactiveResp.Statuses[0].Status.Status = ethpb.ValidatorStatus_UNKNOWN_STATUS
		inactiveClientStream := mock.NewMockBeaconNodeValidator_WaitForActivationClient(ctrl)
		client.EXPECT().WaitForActivation(
			gomock.Any(),
			&ethpb.ValidatorActivationRequest{
				PublicKeys: [][]byte{inactivePubKey[:]},
			},
		).Return(inactiveClientStream, nil)
		inactiveClientStream.EXPECT().Recv().Return(
			inactiveResp,
			nil,
		).AnyTimes()

		activeResp := generateMockStatusResponse([][]byte{inactivePubKey[:], activePubKey[:]})
		activeResp.Statuses[0].Status.Status = ethpb.ValidatorStatus_UNKNOWN_STATUS
		activeResp.Statuses[1].Status.Status = ethpb.ValidatorStatus_ACTIVE
		activeClientStream := mock.NewMockBeaconNodeValidator_WaitForActivationClient(ctrl)
		client.EXPECT().WaitForActivation(
			gomock.Any(),
			&ethpb.ValidatorActivationRequest{
				PublicKeys: [][]byte{inactivePubKey[:], activePubKey[:]},
			},
		).Return(activeClientStream, nil)
		activeClientStream.EXPECT().Recv().Return(
			activeResp,
			nil,
		)

		channel := make(chan [][fieldparams.BLSPubkeyLength]byte)
		go func() {
			// We add the active key into the keymanager and simulate a key refresh.
			time.Sleep(time.Second * 1)
			err = km.RecoverAccountsFromMnemonic(ctx, constant.TestMnemonic, "", 2)
			require.NoError(t, err)
			keys, err := km.FetchValidatingPublicKeys(ctx)
			require.NoError(t, err)
			for _, key := range keys {
				v.pubkeyToValidatorIndex[key] = 1
			}
			channel <- [][fieldparams.BLSPubkeyLength]byte{}
		}()

		assert.NoError(t, v.waitForActivation(context.Background(), channel))
		assert.LogsContain(t, hook, "Waiting for deposit to be observed by beacon node")
		assert.LogsContain(t, hook, "Validator activated")
	})
}

func TestWaitForActivation_RemoteKeymanager(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	client := mock.NewMockBeaconNodeValidatorClient(ctrl)
	stream := mock.NewMockBeaconNodeValidator_WaitForActivationClient(ctrl)
	client.EXPECT().WaitForActivation(
		gomock.Any(),
		gomock.Any(),
	).Return(stream, nil /* err */).AnyTimes()

	inactiveKey := bytesutil.ToBytes48([]byte("inactive"))
	activeKey := bytesutil.ToBytes48([]byte("active"))
	km := remotekeymanagermock.NewMock()
	km.PublicKeys = [][fieldparams.BLSPubkeyLength]byte{inactiveKey, activeKey}
	slot := types.Slot(0)

	t.Run("activated", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		hook := logTest.NewGlobal()

		tickerChan := make(chan types.Slot)
		ticker := &slotutilmock.MockTicker{
			Channel: tickerChan,
		}
		v := validator{
			validatorClient:        client,
			keyManager:             &km,
			ticker:                 ticker,
			pubkeyToValidatorIndex: make(map[[fieldparams.BLSPubkeyLength]byte]types.ValidatorIndex),
			feeRecipientConfig: &validator_service_config.FeeRecipientConfig{
				ProposeConfig: nil,
				DefaultConfig: &validator_service_config.FeeRecipientOptions{
					FeeRecipient: common.HexToAddress("0x046Fb65722E7b2455043BFEBf6177F1D2e9738D9"),
				},
			},
		}
		v.pubkeyToValidatorIndex[activeKey] = 1
		v.pubkeyToValidatorIndex[inactiveKey] = 2
		go func() {
			tickerChan <- slot
			// Cancel after timeout to avoid waiting on channel forever in case test goes wrong.
			time.Sleep(time.Second)
			cancel()
		}()

		resp := testutil.GenerateMultipleValidatorStatusResponse([][]byte{inactiveKey[:], activeKey[:]})
		resp.Statuses[0].Status = ethpb.ValidatorStatus_UNKNOWN_STATUS
		resp.Statuses[1].Status = ethpb.ValidatorStatus_ACTIVE
		client.EXPECT().MultipleValidatorStatus(
			gomock.Any(),
			&ethpb.MultipleValidatorStatusRequest{
				PublicKeys: [][]byte{inactiveKey[:], activeKey[:]},
			},
		).Return(resp, nil /* err */)

		err := v.waitForActivation(ctx, nil /* accountsChangedChan */)
		require.NoError(t, err)
		assert.LogsContain(t, hook, "Waiting for deposit to be observed by beacon node")
		assert.LogsContain(t, hook, "Validator activated")
	})

	t.Run("cancelled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())

		tickerChan := make(chan types.Slot)
		ticker := &slotutilmock.MockTicker{
			Channel: tickerChan,
		}
		v := validator{
			validatorClient: client,
			keyManager:      &km,
			ticker:          ticker,
		}
		go func() {
			cancel()
			tickerChan <- slot
		}()

		err := v.waitForActivation(ctx, nil /* accountsChangedChan */)
		assert.ErrorContains(t, "context canceled, not waiting for activation anymore", err)
	})
	t.Run("reloaded", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		hook := logTest.NewGlobal()
		remoteKm := remotekeymanagermock.NewMock()
		remoteKm.PublicKeys = [][fieldparams.BLSPubkeyLength]byte{inactiveKey}

		tickerChan := make(chan types.Slot)
		ticker := &slotutilmock.MockTicker{
			Channel: tickerChan,
		}
		v := validator{
			validatorClient:        client,
			keyManager:             &remoteKm,
			ticker:                 ticker,
			pubkeyToValidatorIndex: make(map[[fieldparams.BLSPubkeyLength]byte]types.ValidatorIndex),
			feeRecipientConfig: &validator_service_config.FeeRecipientConfig{
				ProposeConfig: nil,
				DefaultConfig: &validator_service_config.FeeRecipientOptions{
					FeeRecipient: common.HexToAddress("0x046Fb65722E7b2455043BFEBf6177F1D2e9738D9"),
				},
			},
		}
		v.pubkeyToValidatorIndex[activeKey] = 1
		v.pubkeyToValidatorIndex[inactiveKey] = 2
		go func() {
			tickerChan <- slot
			time.Sleep(time.Second)
			remoteKm.PublicKeys = [][fieldparams.BLSPubkeyLength]byte{inactiveKey, activeKey}
			tickerChan <- slot
			// Cancel after timeout to avoid waiting on channel forever in case test goes wrong.
			time.Sleep(time.Second)
			cancel()
		}()

		resp := testutil.GenerateMultipleValidatorStatusResponse([][]byte{inactiveKey[:]})
		resp.Statuses[0].Status = ethpb.ValidatorStatus_UNKNOWN_STATUS
		client.EXPECT().MultipleValidatorStatus(
			gomock.Any(),
			&ethpb.MultipleValidatorStatusRequest{
				PublicKeys: [][]byte{inactiveKey[:]},
			},
		).Return(resp, nil /* err */)
		resp2 := testutil.GenerateMultipleValidatorStatusResponse([][]byte{inactiveKey[:], activeKey[:]})
		resp2.Statuses[0].Status = ethpb.ValidatorStatus_UNKNOWN_STATUS
		resp2.Statuses[1].Status = ethpb.ValidatorStatus_ACTIVE
		client.EXPECT().MultipleValidatorStatus(
			gomock.Any(),
			&ethpb.MultipleValidatorStatusRequest{
				PublicKeys: [][]byte{inactiveKey[:], activeKey[:]},
			},
		).Return(resp2, nil /* err */)

		err := v.waitForActivation(ctx, remoteKm.ReloadPublicKeysChan /* accountsChangedChan */)
		require.NoError(t, err)
		assert.LogsContain(t, hook, "Waiting for deposit to be observed by beacon node")
		assert.LogsContain(t, hook, "Validator activated")
	})
}
