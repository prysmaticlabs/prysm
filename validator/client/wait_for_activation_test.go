package client

import (
	"context"
	"testing"
	"time"

	fieldparams "github.com/prysmaticlabs/prysm/v5/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v5/config/params"
	ethpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v5/testing/assert"
	"github.com/prysmaticlabs/prysm/v5/testing/require"
	validatormock "github.com/prysmaticlabs/prysm/v5/testing/validator-mock"
	"github.com/prysmaticlabs/prysm/v5/validator/client/iface"
	"github.com/prysmaticlabs/prysm/v5/validator/client/testutil"
	logTest "github.com/sirupsen/logrus/hooks/test"
	"go.uber.org/mock/gomock"
)

func TestWaitActivation_Exiting_OK(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	validatorClient := validatormock.NewMockValidatorClient(ctrl)
	chainClient := validatormock.NewMockChainClient(ctrl)
	prysmChainClient := validatormock.NewMockPrysmChainClient(ctrl)
	kp := randKeypair(t)
	v := validator{
		validatorClient:  validatorClient,
		km:               newMockKeymanager(t, kp),
		chainClient:      chainClient,
		prysmChainClient: prysmChainClient,
	}
	ctx := context.Background()
	resp := testutil.GenerateMultipleValidatorStatusResponse([][]byte{kp.pub[:]})
	resp.Statuses[0].Status = ethpb.ValidatorStatus_EXITING
	validatorClient.EXPECT().MultipleValidatorStatus(
		gomock.Any(),
		&ethpb.MultipleValidatorStatusRequest{
			PublicKeys: [][]byte{kp.pub[:]},
		},
	).Return(resp, nil)
	prysmChainClient.EXPECT().ValidatorCount(
		gomock.Any(),
		"head",
		gomock.Any(),
	).Return([]iface.ValidatorCount{
		{
			Status: "EXITING",
			Count:  1,
		},
	}, nil).AnyTimes()

	require.NoError(t, v.WaitForActivation(ctx, nil))
	require.Equal(t, 1, len(v.pubkeyToStatus))
}

func TestWaitForActivation_RefetchKeys(t *testing.T) {
	params.SetupTestConfigCleanup(t)
	cfg := params.MainnetConfig().Copy()
	cfg.ConfigName = "test"
	cfg.SecondsPerSlot = 1
	params.OverrideBeaconConfig(cfg)
	hook := logTest.NewGlobal()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	validatorClient := validatormock.NewMockValidatorClient(ctrl)
	chainClient := validatormock.NewMockChainClient(ctrl)
	prysmChainClient := validatormock.NewMockPrysmChainClient(ctrl)

	kp := randKeypair(t)
	km := newMockKeymanager(t)

	v := validator{
		validatorClient:  validatorClient,
		km:               km,
		chainClient:      chainClient,
		prysmChainClient: prysmChainClient,
		pubkeyToStatus:   make(map[[48]byte]*validatorStatus),
	}
	resp := testutil.GenerateMultipleValidatorStatusResponse([][]byte{kp.pub[:]})
	resp.Statuses[0].Status = ethpb.ValidatorStatus_ACTIVE

	validatorClient.EXPECT().MultipleValidatorStatus(
		gomock.Any(),
		&ethpb.MultipleValidatorStatusRequest{
			PublicKeys: [][]byte{kp.pub[:]},
		},
	).Return(resp, nil)
	prysmChainClient.EXPECT().ValidatorCount(
		gomock.Any(),
		"head",
		gomock.Any(),
	).Return([]iface.ValidatorCount{
		{
			Status: "ACTIVE",
			Count:  1,
		},
	}, nil)

	accountChan := make(chan [][fieldparams.BLSPubkeyLength]byte)
	sub := km.SubscribeAccountChanges(accountChan)
	defer func() {
		sub.Unsubscribe()
		close(accountChan)
	}()
	// update the accounts from 0 to 1 after a delay
	go func() {
		time.Sleep(1 * time.Second)
		require.NoError(t, km.add(kp))
		km.SimulateAccountChanges([][48]byte{kp.pub})
	}()
	assert.NoError(t, v.internalWaitForActivation(context.Background(), accountChan), "Could not wait for activation")
	assert.LogsContain(t, hook, msgNoKeysFetched)
	assert.LogsContain(t, hook, "Validator activated")
}

//type MultipleValidatorStatusRequestMatcher struct {
//	pubkeys [][fieldparams.BLSPubkeyLength]byte
//}
//
//func (m *MultipleValidatorStatusRequestMatcher) Matches(x interface{}) bool {
//	req, ok := x.(*ethpb.PrepareBeaconProposerRequest)
//	if !ok {
//		return false
//	}
//
//	if len(req.Recipients) != len(m.expectedRecipients) {
//		return false
//	}
//
//	// Build maps for efficient comparison
//	expectedMap := make(map[primitives.ValidatorIndex][]byte)
//	for _, recipient := range m.expectedRecipients {
//		expectedMap[recipient.ValidatorIndex] = recipient.FeeRecipient
//	}
//
//	// Compare the maps
//	for _, fc := range req.Recipients {
//		expectedFeeRecipient, exists := expectedMap[fc.ValidatorIndex]
//		if !exists || !bytes.Equal(expectedFeeRecipient, fc.FeeRecipient) {
//			return false
//		}
//	}
//	return true
//}
//
//func (m *PrepareBeaconProposerRequestMatcher) String() string {
//	return fmt.Sprintf("matches PrepareBeaconProposerRequest with Recipients: %v", m.expectedRecipients)
//}

// Regression test for a scenario where you start with an inactive key and then import an active key.
func TestWaitForActivation_AccountsChanged(t *testing.T) {
	hook := logTest.NewGlobal()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	t.Run("Imported keymanager", func(t *testing.T) {
		inactive := randKeypair(t)
		active := randKeypair(t)
		km := newMockKeymanager(t, inactive)
		validatorClient := validatormock.NewMockValidatorClient(ctrl)
		chainClient := validatormock.NewMockChainClient(ctrl)
		prysmChainClient := validatormock.NewMockPrysmChainClient(ctrl)
		v := validator{
			validatorClient:  validatorClient,
			km:               km,
			chainClient:      chainClient,
			prysmChainClient: prysmChainClient,
			pubkeyToStatus:   make(map[[48]byte]*validatorStatus),
		}
		inactiveResp := testutil.GenerateMultipleValidatorStatusResponse([][]byte{inactive.pub[:]})
		inactiveResp.Statuses[0].Status = ethpb.ValidatorStatus_UNKNOWN_STATUS

		validatorClient.EXPECT().MultipleValidatorStatus(
			gomock.Any(),
			&ethpb.MultipleValidatorStatusRequest{
				PublicKeys: [][]byte{inactive.pub[:]},
			},
		).Return(inactiveResp, nil)
		prysmChainClient.EXPECT().ValidatorCount(
			gomock.Any(),
			"head",
			gomock.Any(),
		).Return([]iface.ValidatorCount{}, nil).AnyTimes()

		activeResp := testutil.GenerateMultipleValidatorStatusResponse([][]byte{inactive.pub[:], active.pub[:]})
		activeResp.Statuses[0].Status = ethpb.ValidatorStatus_UNKNOWN_STATUS
		activeResp.Statuses[1].Status = ethpb.ValidatorStatus_ACTIVE
		validatorClient.EXPECT().MultipleValidatorStatus(
			gomock.Any(),
			&ethpb.MultipleValidatorStatusRequest{
				PublicKeys: [][]byte{inactive.pub[:], active.pub[:]},
			},
		).Return(activeResp, nil)

		chainClient.EXPECT().ChainHead(
			gomock.Any(),
			gomock.Any(),
		).Return(
			&ethpb.ChainHead{HeadEpoch: 0},
			nil,
		)

		go func() {
			// We add the active key into the keymanager and simulate a key refresh.
			time.Sleep(time.Second * 1)
			require.NoError(t, km.add(active))
			km.SimulateAccountChanges(make([][fieldparams.BLSPubkeyLength]byte, 0))
		}()

		assert.NoError(t, v.WaitForActivation(context.Background(), nil))
		assert.LogsContain(t, hook, "Waiting for deposit to be observed by beacon node")
		assert.LogsContain(t, hook, "Validator activated")
	})

	//t.Run("Derived keymanager", func(t *testing.T) {
	//	seed := bip39.NewSeed(constant.TestMnemonic, "")
	//	inactivePrivKey, err :=
	//		util.PrivateKeyFromSeedAndPath(seed, fmt.Sprintf(derived.ValidatingKeyDerivationPathTemplate, 0))
	//	require.NoError(t, err)
	//	var inactivePubKey [fieldparams.BLSPubkeyLength]byte
	//	copy(inactivePubKey[:], inactivePrivKey.PublicKey().Marshal())
	//	activePrivKey, err :=
	//		util.PrivateKeyFromSeedAndPath(seed, fmt.Sprintf(derived.ValidatingKeyDerivationPathTemplate, 1))
	//	require.NoError(t, err)
	//	var activePubKey [fieldparams.BLSPubkeyLength]byte
	//	copy(activePubKey[:], activePrivKey.PublicKey().Marshal())
	//	wallet := &walletMock.Wallet{
	//		Files:            make(map[string]map[string][]byte),
	//		AccountPasswords: make(map[string]string),
	//		WalletPassword:   "secretPassw0rd$1999",
	//	}
	//	ctx := context.Background()
	//	km, err := derived.NewKeymanager(ctx, &derived.SetupConfig{
	//		Wallet:           wallet,
	//		ListenForChanges: true,
	//	})
	//	require.NoError(t, err)
	//	err = km.RecoverAccountsFromMnemonic(ctx, constant.TestMnemonic, derived.DefaultMnemonicLanguage, "", 1)
	//	require.NoError(t, err)
	//	validatorClient := validatormock.NewMockValidatorClient(ctrl)
	//	chainClient := validatormock.NewMockChainClient(ctrl)
	//	prysmChainClient := validatormock.NewMockPrysmChainClient(ctrl)
	//	v := validator{
	//		validatorClient:  validatorClient,
	//		km:               km,
	//		genesisTime:      1,
	//		chainClient:      chainClient,
	//		prysmChainClient: prysmChainClient,
	//		pubkeyToStatus:   make(map[[48]byte]*validatorStatus),
	//	}
	//
	//	inactiveResp := generateMockStatusResponse([][]byte{inactivePubKey[:]})
	//	inactiveResp.Statuses[0].Status.Status = ethpb.ValidatorStatus_UNKNOWN_STATUS
	//	inactiveClientStream := mock.NewMockBeaconNodeValidator_WaitForActivationClient(ctrl)
	//	validatorClient.EXPECT().WaitForActivation(
	//		gomock.Any(),
	//		&ethpb.ValidatorActivationRequest{
	//			PublicKeys: [][]byte{inactivePubKey[:]},
	//		},
	//	).DoAndReturn(func(ctx context.Context, in *ethpb.ValidatorActivationRequest) (*mock.MockBeaconNodeValidator_WaitForActivationClient, error) {
	//		//delay a bit so that other key can be added
	//		time.Sleep(time.Second * 2)
	//		return inactiveClientStream, nil
	//	})
	//	prysmChainClient.EXPECT().ValidatorCount(
	//		gomock.Any(),
	//		"head",
	//		[]validatorType.Status{validatorType.Active},
	//	).Return([]iface.ValidatorCount{}, nil).AnyTimes()
	//	inactiveClientStream.EXPECT().Recv().Return(
	//		inactiveResp,
	//		nil,
	//	).AnyTimes()
	//
	//	activeResp := generateMockStatusResponse([][]byte{inactivePubKey[:], activePubKey[:]})
	//	activeResp.Statuses[0].Status.Status = ethpb.ValidatorStatus_UNKNOWN_STATUS
	//	activeResp.Statuses[1].Status.Status = ethpb.ValidatorStatus_ACTIVE
	//	activeClientStream := mock.NewMockBeaconNodeValidator_WaitForActivationClient(ctrl)
	//	validatorClient.EXPECT().WaitForActivation(
	//		gomock.Any(),
	//		&ethpb.ValidatorActivationRequest{
	//			PublicKeys: [][]byte{inactivePubKey[:], activePubKey[:]},
	//		},
	//	).Return(activeClientStream, nil)
	//	activeClientStream.EXPECT().Recv().Return(
	//		activeResp,
	//		nil,
	//	)
	//
	//	channel := make(chan [][fieldparams.BLSPubkeyLength]byte)
	//	go func() {
	//		// We add the active key into the keymanager and simulate a key refresh.
	//		time.Sleep(time.Second * 1)
	//		err = km.RecoverAccountsFromMnemonic(ctx, constant.TestMnemonic, derived.DefaultMnemonicLanguage, "", 2)
	//		require.NoError(t, err)
	//		channel <- [][fieldparams.BLSPubkeyLength]byte{}
	//	}()
	//
	//	assert.NoError(t, v.internalWaitForActivation(context.Background(), channel))
	//	assert.LogsContain(t, hook, "Waiting for deposit to be observed by beacon node")
	//	assert.LogsContain(t, hook, "Validator activated")
	//})
}

//func TestWaitForActivation_ReceiveErrorFromStream_AttemptsReconnection(t *testing.T) {
//	ctrl := gomock.NewController(t)
//	defer ctrl.Finish()
//	validatorClient := validatormock.NewMockValidatorClient(ctrl)
//	chainClient := validatormock.NewMockChainClient(ctrl)
//	prysmChainClient := validatormock.NewMockPrysmChainClient(ctrl)
//	kp := randKeypair(t)
//	v := validator{
//		validatorClient:  validatorClient,
//		km:               newMockKeymanager(t, kp),
//		chainClient:      chainClient,
//		prysmChainClient: prysmChainClient,
//		pubkeyToStatus:   make(map[[48]byte]*validatorStatus),
//	}
//	clientStream := mock.NewMockBeaconNodeValidator_WaitForActivationClient(ctrl)
//	validatorClient.EXPECT().WaitForActivation(
//		gomock.Any(),
//		&ethpb.ValidatorActivationRequest{
//			PublicKeys: [][]byte{kp.pub[:]},
//		},
//	).Return(clientStream, nil)
//	prysmChainClient.EXPECT().ValidatorCount(
//		gomock.Any(),
//		"head",
//		[]validatorType.Status{validatorType.Active},
//	).Return([]iface.ValidatorCount{}, nil)
//	// A stream fails the first time, but succeeds the second time.
//	resp := generateMockStatusResponse([][]byte{kp.pub[:]})
//	resp.Statuses[0].Status.Status = ethpb.ValidatorStatus_ACTIVE
//	clientStream.EXPECT().Recv().Return(
//		nil,
//		errors.New("fails"),
//	).Return(resp, nil)
//	assert.NoError(t, v.WaitForActivation(context.Background(), nil))
//}
//
//func TestWaitActivation_LogsActivationEpochOK(t *testing.T) {
//	hook := logTest.NewGlobal()
//	ctrl := gomock.NewController(t)
//	defer ctrl.Finish()
//	validatorClient := validatormock.NewMockValidatorClient(ctrl)
//	chainClient := validatormock.NewMockChainClient(ctrl)
//	prysmChainClient := validatormock.NewMockPrysmChainClient(ctrl)
//	kp := randKeypair(t)
//	v := validator{
//		validatorClient:  validatorClient,
//		km:               newMockKeymanager(t, kp),
//		genesisTime:      1,
//		chainClient:      chainClient,
//		prysmChainClient: prysmChainClient,
//		pubkeyToStatus:   make(map[[48]byte]*validatorStatus),
//	}
//	resp := generateMockStatusResponse([][]byte{kp.pub[:]})
//	resp.Statuses[0].Status.Status = ethpb.ValidatorStatus_ACTIVE
//	clientStream := mock.NewMockBeaconNodeValidator_WaitForActivationClient(ctrl)
//	validatorClient.EXPECT().WaitForActivation(
//		gomock.Any(),
//		&ethpb.ValidatorActivationRequest{
//			PublicKeys: [][]byte{kp.pub[:]},
//		},
//	).Return(clientStream, nil)
//	prysmChainClient.EXPECT().ValidatorCount(
//		gomock.Any(),
//		"head",
//		[]validatorType.Status{validatorType.Active},
//	).Return([]iface.ValidatorCount{}, nil)
//	clientStream.EXPECT().Recv().Return(
//		resp,
//		nil,
//	)
//	assert.NoError(t, v.WaitForActivation(context.Background(), nil), "Could not wait for activation")
//	assert.LogsContain(t, hook, "Validator activated")
//}
//

//}
//
//func TestWaitActivation_NotAllValidatorsActivatedOK(t *testing.T) {
//	ctrl := gomock.NewController(t)
//	defer ctrl.Finish()
//	validatorClient := validatormock.NewMockValidatorClient(ctrl)
//	chainClient := validatormock.NewMockChainClient(ctrl)
//	prysmChainClient := validatormock.NewMockPrysmChainClient(ctrl)
//
//	kp := randKeypair(t)
//	v := validator{
//		validatorClient:  validatorClient,
//		km:               newMockKeymanager(t, kp),
//		chainClient:      chainClient,
//		prysmChainClient: prysmChainClient,
//		pubkeyToStatus:   make(map[[48]byte]*validatorStatus),
//	}
//	resp := generateMockStatusResponse([][]byte{kp.pub[:]})
//	resp.Statuses[0].Status.Status = ethpb.ValidatorStatus_ACTIVE
//	clientStream := mock.NewMockBeaconNodeValidator_WaitForActivationClient(ctrl)
//	validatorClient.EXPECT().WaitForActivation(
//		gomock.Any(),
//		gomock.Any(),
//	).Return(clientStream, nil)
//	prysmChainClient.EXPECT().ValidatorCount(
//		gomock.Any(),
//		"head",
//		[]validatorType.Status{validatorType.Active},
//	).Return([]iface.ValidatorCount{}, nil).Times(2)
//	clientStream.EXPECT().Recv().Return(
//		&ethpb.ValidatorActivationResponse{},
//		nil,
//	)
//	clientStream.EXPECT().Recv().Return(
//		resp,
//		nil,
//	)
//	assert.NoError(t, v.WaitForActivation(context.Background(), nil), "Could not wait for activation")
//}
