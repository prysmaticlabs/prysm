package client

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"math"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/golang/mock/gomock"
	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/async/event"
	"github.com/prysmaticlabs/prysm/config/features"
	fieldparams "github.com/prysmaticlabs/prysm/config/fieldparams"
	"github.com/prysmaticlabs/prysm/config/params"
	validator_service_config "github.com/prysmaticlabs/prysm/config/validator/service"
	"github.com/prysmaticlabs/prysm/crypto/bls"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	ethpbservice "github.com/prysmaticlabs/prysm/proto/eth/service"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	validatorpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/validator-client"
	"github.com/prysmaticlabs/prysm/testing/assert"
	mock2 "github.com/prysmaticlabs/prysm/testing/mock"
	"github.com/prysmaticlabs/prysm/testing/require"
	"github.com/prysmaticlabs/prysm/testing/util"
	"github.com/prysmaticlabs/prysm/validator/accounts/wallet"
	"github.com/prysmaticlabs/prysm/validator/client/iface"
	dbTest "github.com/prysmaticlabs/prysm/validator/db/testing"
	"github.com/prysmaticlabs/prysm/validator/keymanager"
	"github.com/prysmaticlabs/prysm/validator/keymanager/local"
	remote_web3signer "github.com/prysmaticlabs/prysm/validator/keymanager/remote-web3signer"
	"github.com/sirupsen/logrus"
	logTest "github.com/sirupsen/logrus/hooks/test"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

func init() {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetOutput(ioutil.Discard)
}

var _ iface.Validator = (*validator)(nil)

const cancelledCtx = "context has been canceled"

func genMockKeymanager(numKeys int) *mockKeymanager {
	km := make(map[[fieldparams.BLSPubkeyLength]byte]bls.SecretKey, numKeys)
	for i := 0; i < numKeys; i++ {
		k, err := bls.RandKey()
		if err != nil {
			panic(err)
		}
		km[bytesutil.ToBytes48(k.PublicKey().Marshal())] = k
	}

	return &mockKeymanager{keysMap: km}
}

type mockKeymanager struct {
	lock                sync.RWMutex
	keysMap             map[[fieldparams.BLSPubkeyLength]byte]bls.SecretKey
	fetchNoKeys         bool
	accountsChangedFeed *event.Feed
}

func (m *mockKeymanager) FetchValidatingPublicKeys(_ context.Context) ([][fieldparams.BLSPubkeyLength]byte, error) {
	m.lock.RLock()
	defer m.lock.RUnlock()
	keys := make([][fieldparams.BLSPubkeyLength]byte, 0)
	if m.fetchNoKeys {
		m.fetchNoKeys = false
		return keys, nil
	}
	for pubKey := range m.keysMap {
		keys = append(keys, pubKey)
	}
	return keys, nil
}

func (m *mockKeymanager) Sign(_ context.Context, req *validatorpb.SignRequest) (bls.Signature, error) {
	pubKey := [fieldparams.BLSPubkeyLength]byte{}
	copy(pubKey[:], req.PublicKey)
	privKey, ok := m.keysMap[pubKey]
	if !ok {
		return nil, errors.New("not found")
	}
	sig := privKey.Sign(req.SigningRoot)
	return sig, nil
}

func (m *mockKeymanager) SubscribeAccountChanges(pubKeysChan chan [][fieldparams.BLSPubkeyLength]byte) event.Subscription {
	if m.accountsChangedFeed == nil {
		m.accountsChangedFeed = &event.Feed{}
	}
	return m.accountsChangedFeed.Subscribe(pubKeysChan)
}

func (m *mockKeymanager) SimulateAccountChanges(newKeys [][fieldparams.BLSPubkeyLength]byte) {
	m.accountsChangedFeed.Send(newKeys)
}

func (*mockKeymanager) ExtractKeystores(
	ctx context.Context, publicKeys []bls.PublicKey, password string,
) ([]*keymanager.Keystore, error) {
	return nil, errors.New("extracting keys not supported on mock keymanager")
}

func (*mockKeymanager) ListKeymanagerAccounts(
	context.Context, keymanager.ListKeymanagerAccountConfig) error {
	return nil
}

func (*mockKeymanager) DeleteKeystores(context.Context, [][]byte,
) ([]*ethpbservice.DeletedKeystoreStatus, error) {
	return nil, nil
}

func generateMockStatusResponse(pubkeys [][]byte) *ethpb.ValidatorActivationResponse {
	multipleStatus := make([]*ethpb.ValidatorActivationResponse_Status, len(pubkeys))
	for i, key := range pubkeys {
		multipleStatus[i] = &ethpb.ValidatorActivationResponse_Status{
			PublicKey: key,
			Status: &ethpb.ValidatorStatusResponse{
				Status: ethpb.ValidatorStatus_UNKNOWN_STATUS,
			},
		}
	}
	return &ethpb.ValidatorActivationResponse{Statuses: multipleStatus}
}

func TestWaitForChainStart_SetsGenesisInfo(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	client := mock2.NewMockBeaconNodeValidatorClient(ctrl)

	db := dbTest.SetupDB(t, [][fieldparams.BLSPubkeyLength]byte{})
	v := validator{
		validatorClient: client,
		db:              db,
	}

	// Make sure its clean at the start.
	savedGenValRoot, err := db.GenesisValidatorsRoot(context.Background())
	require.NoError(t, err)
	assert.DeepEqual(t, []byte(nil), savedGenValRoot, "Unexpected saved genesis validators root")

	genesis := uint64(time.Unix(1, 0).Unix())
	genesisValidatorsRoot := bytesutil.ToBytes32([]byte("validators"))
	clientStream := mock2.NewMockBeaconNodeValidator_WaitForChainStartClient(ctrl)
	client.EXPECT().WaitForChainStart(
		gomock.Any(),
		&emptypb.Empty{},
	).Return(clientStream, nil)
	clientStream.EXPECT().Recv().Return(
		&ethpb.ChainStartResponse{
			Started:               true,
			GenesisTime:           genesis,
			GenesisValidatorsRoot: genesisValidatorsRoot[:],
		},
		nil,
	)
	require.NoError(t, v.WaitForChainStart(context.Background()))
	savedGenValRoot, err = db.GenesisValidatorsRoot(context.Background())
	require.NoError(t, err)

	assert.DeepEqual(t, genesisValidatorsRoot[:], savedGenValRoot, "Unexpected saved genesis validators root")
	assert.Equal(t, genesis, v.genesisTime, "Unexpected chain start time")
	assert.NotNil(t, v.ticker, "Expected ticker to be set, received nil")

	// Make sure theres no errors running if its the same data.
	client.EXPECT().WaitForChainStart(
		gomock.Any(),
		&emptypb.Empty{},
	).Return(clientStream, nil)
	clientStream.EXPECT().Recv().Return(
		&ethpb.ChainStartResponse{
			Started:               true,
			GenesisTime:           genesis,
			GenesisValidatorsRoot: genesisValidatorsRoot[:],
		},
		nil,
	)
	require.NoError(t, v.WaitForChainStart(context.Background()))
}

func TestWaitForChainStart_SetsGenesisInfo_IncorrectSecondTry(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	client := mock2.NewMockBeaconNodeValidatorClient(ctrl)

	db := dbTest.SetupDB(t, [][fieldparams.BLSPubkeyLength]byte{})
	v := validator{
		validatorClient: client,
		db:              db,
	}
	genesis := uint64(time.Unix(1, 0).Unix())
	genesisValidatorsRoot := bytesutil.ToBytes32([]byte("validators"))
	clientStream := mock2.NewMockBeaconNodeValidator_WaitForChainStartClient(ctrl)
	client.EXPECT().WaitForChainStart(
		gomock.Any(),
		&emptypb.Empty{},
	).Return(clientStream, nil)
	clientStream.EXPECT().Recv().Return(
		&ethpb.ChainStartResponse{
			Started:               true,
			GenesisTime:           genesis,
			GenesisValidatorsRoot: genesisValidatorsRoot[:],
		},
		nil,
	)
	require.NoError(t, v.WaitForChainStart(context.Background()))
	savedGenValRoot, err := db.GenesisValidatorsRoot(context.Background())
	require.NoError(t, err)

	assert.DeepEqual(t, genesisValidatorsRoot[:], savedGenValRoot, "Unexpected saved genesis validators root")
	assert.Equal(t, genesis, v.genesisTime, "Unexpected chain start time")
	assert.NotNil(t, v.ticker, "Expected ticker to be set, received nil")

	genesisValidatorsRoot = bytesutil.ToBytes32([]byte("badvalidators"))

	// Make sure theres no errors running if its the same data.
	client.EXPECT().WaitForChainStart(
		gomock.Any(),
		&emptypb.Empty{},
	).Return(clientStream, nil)
	clientStream.EXPECT().Recv().Return(
		&ethpb.ChainStartResponse{
			Started:               true,
			GenesisTime:           genesis,
			GenesisValidatorsRoot: genesisValidatorsRoot[:],
		},
		nil,
	)
	err = v.WaitForChainStart(context.Background())
	require.ErrorContains(t, "does not match root saved", err)
}

func TestWaitForChainStart_ContextCanceled(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	client := mock2.NewMockBeaconNodeValidatorClient(ctrl)

	v := validator{
		//keyManager:      testKeyManager,
		validatorClient: client,
	}
	genesis := uint64(time.Unix(0, 0).Unix())
	genesisValidatorsRoot := bytesutil.PadTo([]byte("validators"), 32)
	clientStream := mock2.NewMockBeaconNodeValidator_WaitForChainStartClient(ctrl)
	client.EXPECT().WaitForChainStart(
		gomock.Any(),
		&emptypb.Empty{},
	).Return(clientStream, nil)
	clientStream.EXPECT().Recv().Return(
		&ethpb.ChainStartResponse{
			Started:               true,
			GenesisTime:           genesis,
			GenesisValidatorsRoot: genesisValidatorsRoot,
		},
		nil,
	)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	assert.ErrorContains(t, cancelledCtx, v.WaitForChainStart(ctx))
}

func TestWaitForChainStart_StreamSetupFails(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	client := mock2.NewMockBeaconNodeValidatorClient(ctrl)

	privKey, err := bls.RandKey()
	require.NoError(t, err)
	pubKey := [fieldparams.BLSPubkeyLength]byte{}
	copy(pubKey[:], privKey.PublicKey().Marshal())
	km := &mockKeymanager{
		keysMap: make(map[[fieldparams.BLSPubkeyLength]byte]bls.SecretKey),
	}
	v := validator{
		validatorClient: client,
		keyManager:      km,
	}
	clientStream := mock2.NewMockBeaconNodeValidator_WaitForChainStartClient(ctrl)
	client.EXPECT().WaitForChainStart(
		gomock.Any(),
		&emptypb.Empty{},
	).Return(clientStream, errors.New("failed stream"))
	err = v.WaitForChainStart(context.Background())
	want := "could not setup beacon chain ChainStart streaming client"
	assert.ErrorContains(t, want, err)
}

func TestWaitForChainStart_ReceiveErrorFromStream(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	client := mock2.NewMockBeaconNodeValidatorClient(ctrl)

	v := validator{
		validatorClient: client,
	}
	clientStream := mock2.NewMockBeaconNodeValidator_WaitForChainStartClient(ctrl)
	client.EXPECT().WaitForChainStart(
		gomock.Any(),
		&emptypb.Empty{},
	).Return(clientStream, nil)
	clientStream.EXPECT().Recv().Return(
		nil,
		errors.New("fails"),
	)
	err := v.WaitForChainStart(context.Background())
	want := "could not receive ChainStart from stream"
	assert.ErrorContains(t, want, err)
}

func TestCanonicalHeadSlot_FailedRPC(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	client := mock2.NewMockBeaconChainClient(ctrl)
	v := validator{
		beaconClient: client,
		genesisTime:  1,
	}
	client.EXPECT().GetChainHead(
		gomock.Any(),
		gomock.Any(),
	).Return(nil, errors.New("failed"))
	_, err := v.CanonicalHeadSlot(context.Background())
	assert.ErrorContains(t, "failed", err)
}

func TestCanonicalHeadSlot_OK(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	client := mock2.NewMockBeaconChainClient(ctrl)
	v := validator{
		beaconClient: client,
	}
	client.EXPECT().GetChainHead(
		gomock.Any(),
		gomock.Any(),
	).Return(&ethpb.ChainHead{HeadSlot: 0}, nil)
	headSlot, err := v.CanonicalHeadSlot(context.Background())
	require.NoError(t, err)
	assert.Equal(t, types.Slot(0), headSlot, "Mismatch slots")
}

func TestWaitMultipleActivation_LogsActivationEpochOK(t *testing.T) {
	ctx := context.Background()
	hook := logTest.NewGlobal()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	client := mock2.NewMockBeaconNodeValidatorClient(ctrl)
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
		pubkeyToValidatorIndex: map[[fieldparams.BLSPubkeyLength]byte]types.ValidatorIndex{pubKey: 1},
		feeRecipientConfig: &validator_service_config.FeeRecipientConfig{
			ProposeConfig: nil,
			DefaultConfig: &validator_service_config.FeeRecipientOptions{
				FeeRecipient: common.HexToAddress("0x6e35733c5af9B61374A128e6F85f553aF09ff89A"),
			},
		},
	}

	resp := generateMockStatusResponse([][]byte{pubKey[:]})
	resp.Statuses[0].Status.Status = ethpb.ValidatorStatus_ACTIVE
	clientStream := mock2.NewMockBeaconNodeValidator_WaitForActivationClient(ctrl)
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
	require.NoError(t, v.WaitForActivation(ctx, nil), "Could not wait for activation")
	require.LogsContain(t, hook, "Validator activated")
}

func TestWaitActivation_NotAllValidatorsActivatedOK(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	client := mock2.NewMockBeaconNodeValidatorClient(ctrl)

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
		pubkeyToValidatorIndex: map[[fieldparams.BLSPubkeyLength]byte]types.ValidatorIndex{pubKey: 1},
		feeRecipientConfig: &validator_service_config.FeeRecipientConfig{
			ProposeConfig: nil,
			DefaultConfig: &validator_service_config.FeeRecipientOptions{
				FeeRecipient: common.HexToAddress("0x6e35733c5af9B61374A128e6F85f553aF09ff89A"),
			},
		},
	}
	resp := generateMockStatusResponse([][]byte{pubKey[:]})
	resp.Statuses[0].Status.Status = ethpb.ValidatorStatus_ACTIVE
	clientStream := mock2.NewMockBeaconNodeValidator_WaitForActivationClient(ctrl)
	client.EXPECT().WaitForActivation(
		gomock.Any(),
		gomock.Any(),
	).Return(clientStream, nil)
	clientStream.EXPECT().Recv().Return(
		&ethpb.ValidatorActivationResponse{},
		nil,
	)
	clientStream.EXPECT().Recv().Return(
		resp,
		nil,
	)
	assert.NoError(t, v.WaitForActivation(context.Background(), nil), "Could not wait for activation")
}

func TestWaitSync_ContextCanceled(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	n := mock2.NewMockNodeClient(ctrl)

	v := validator{
		node: n,
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	n.EXPECT().GetSyncStatus(
		gomock.Any(),
		gomock.Any(),
	).Return(&ethpb.SyncStatus{Syncing: true}, nil)

	assert.ErrorContains(t, cancelledCtx, v.WaitForSync(ctx))
}

func TestWaitSync_NotSyncing(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	n := mock2.NewMockNodeClient(ctrl)

	v := validator{
		node: n,
	}

	n.EXPECT().GetSyncStatus(
		gomock.Any(),
		gomock.Any(),
	).Return(&ethpb.SyncStatus{Syncing: false}, nil)

	require.NoError(t, v.WaitForSync(context.Background()))
}

func TestWaitSync_Syncing(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	n := mock2.NewMockNodeClient(ctrl)

	v := validator{
		node: n,
	}

	n.EXPECT().GetSyncStatus(
		gomock.Any(),
		gomock.Any(),
	).Return(&ethpb.SyncStatus{Syncing: true}, nil)

	n.EXPECT().GetSyncStatus(
		gomock.Any(),
		gomock.Any(),
	).Return(&ethpb.SyncStatus{Syncing: false}, nil)

	require.NoError(t, v.WaitForSync(context.Background()))
}

func TestUpdateDuties_DoesNothingWhenNotEpochStart_AlreadyExistingAssignments(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	client := mock2.NewMockBeaconNodeValidatorClient(ctrl)

	slot := types.Slot(1)
	v := validator{
		validatorClient: client,
		duties: &ethpb.DutiesResponse{
			Duties: []*ethpb.DutiesResponse_Duty{
				{
					Committee:      []types.ValidatorIndex{},
					AttesterSlot:   10,
					CommitteeIndex: 20,
				},
			},
		},
	}
	client.EXPECT().GetDuties(
		gomock.Any(),
		gomock.Any(),
	).Times(0)

	assert.NoError(t, v.UpdateDuties(context.Background(), slot), "Could not update assignments")
}

func TestUpdateDuties_ReturnsError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	client := mock2.NewMockBeaconNodeValidatorClient(ctrl)

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
		duties: &ethpb.DutiesResponse{
			Duties: []*ethpb.DutiesResponse_Duty{
				{
					CommitteeIndex: 1,
				},
			},
		},
	}

	expected := errors.New("bad")

	client.EXPECT().GetDuties(
		gomock.Any(),
		gomock.Any(),
	).Return(nil, expected)

	assert.ErrorContains(t, expected.Error(), v.UpdateDuties(context.Background(), params.BeaconConfig().SlotsPerEpoch))
	assert.Equal(t, (*ethpb.DutiesResponse)(nil), v.duties, "Assignments should have been cleared on failure")
}

func TestUpdateDuties_OK(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	client := mock2.NewMockBeaconNodeValidatorClient(ctrl)

	slot := params.BeaconConfig().SlotsPerEpoch
	privKey, err := bls.RandKey()
	require.NoError(t, err)
	pubKey := [fieldparams.BLSPubkeyLength]byte{}
	copy(pubKey[:], privKey.PublicKey().Marshal())
	km := &mockKeymanager{
		keysMap: map[[fieldparams.BLSPubkeyLength]byte]bls.SecretKey{
			pubKey: privKey,
		},
	}
	resp := &ethpb.DutiesResponse{
		Duties: []*ethpb.DutiesResponse_Duty{
			{
				AttesterSlot:   params.BeaconConfig().SlotsPerEpoch,
				ValidatorIndex: 200,
				CommitteeIndex: 100,
				Committee:      []types.ValidatorIndex{0, 1, 2, 3},
				PublicKey:      []byte("testPubKey_1"),
				ProposerSlots:  []types.Slot{params.BeaconConfig().SlotsPerEpoch + 1},
			},
		},
	}
	v := validator{
		keyManager:      km,
		validatorClient: client,
	}
	client.EXPECT().GetDuties(
		gomock.Any(),
		gomock.Any(),
	).Return(resp, nil)

	var wg sync.WaitGroup
	wg.Add(1)

	client.EXPECT().SubscribeCommitteeSubnets(
		gomock.Any(),
		gomock.Any(),
	).DoAndReturn(func(_ context.Context, _ *ethpb.CommitteeSubnetsSubscribeRequest, arg2 ...grpc.CallOption) (*emptypb.Empty, error) {
		wg.Done()
		return nil, nil
	})

	require.NoError(t, v.UpdateDuties(context.Background(), slot), "Could not update assignments")

	util.WaitTimeout(&wg, 3*time.Second)

	assert.Equal(t, params.BeaconConfig().SlotsPerEpoch+1, v.duties.Duties[0].ProposerSlots[0], "Unexpected validator assignments")
	assert.Equal(t, params.BeaconConfig().SlotsPerEpoch, v.duties.Duties[0].AttesterSlot, "Unexpected validator assignments")
	assert.Equal(t, resp.Duties[0].CommitteeIndex, v.duties.Duties[0].CommitteeIndex, "Unexpected validator assignments")
	assert.Equal(t, resp.Duties[0].ValidatorIndex, v.duties.Duties[0].ValidatorIndex, "Unexpected validator assignments")
}

func TestUpdateDuties_OK_FilterBlacklistedPublicKeys(t *testing.T) {
	hook := logTest.NewGlobal()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	client := mock2.NewMockBeaconNodeValidatorClient(ctrl)
	slot := params.BeaconConfig().SlotsPerEpoch

	numValidators := 10
	keysMap := make(map[[fieldparams.BLSPubkeyLength]byte]bls.SecretKey)
	blacklistedPublicKeys := make(map[[fieldparams.BLSPubkeyLength]byte]bool)
	for i := 0; i < numValidators; i++ {
		priv, err := bls.RandKey()
		require.NoError(t, err)
		pubKey := [fieldparams.BLSPubkeyLength]byte{}
		copy(pubKey[:], priv.PublicKey().Marshal())
		keysMap[pubKey] = priv
		blacklistedPublicKeys[pubKey] = true
	}

	km := &mockKeymanager{
		keysMap: keysMap,
	}
	resp := &ethpb.DutiesResponse{
		Duties: []*ethpb.DutiesResponse_Duty{},
	}
	v := validator{
		keyManager:                     km,
		validatorClient:                client,
		eipImportBlacklistedPublicKeys: blacklistedPublicKeys,
	}
	client.EXPECT().GetDuties(
		gomock.Any(),
		gomock.Any(),
	).Return(resp, nil)

	var wg sync.WaitGroup
	wg.Add(1)
	client.EXPECT().SubscribeCommitteeSubnets(
		gomock.Any(),
		gomock.Any(),
	).DoAndReturn(func(_ context.Context, _ *ethpb.CommitteeSubnetsSubscribeRequest, arg2 ...grpc.CallOption) (*emptypb.Empty, error) {
		wg.Done()
		return nil, nil
	})

	require.NoError(t, v.UpdateDuties(context.Background(), slot), "Could not update assignments")

	util.WaitTimeout(&wg, 3*time.Second)

	for range blacklistedPublicKeys {
		assert.LogsContain(t, hook, "Not including slashable public key")
	}
}

func TestRolesAt_OK(t *testing.T) {
	v, m, validatorKey, finish := setup(t)
	defer finish()

	v.duties = &ethpb.DutiesResponse{
		Duties: []*ethpb.DutiesResponse_Duty{
			{
				CommitteeIndex:  1,
				AttesterSlot:    1,
				PublicKey:       validatorKey.PublicKey().Marshal(),
				IsSyncCommittee: true,
			},
		},
		NextEpochDuties: []*ethpb.DutiesResponse_Duty{
			{
				CommitteeIndex:  1,
				AttesterSlot:    1,
				PublicKey:       validatorKey.PublicKey().Marshal(),
				IsSyncCommittee: true,
			},
		},
	}

	m.validatorClient.EXPECT().DomainData(
		gomock.Any(), // ctx
		gomock.Any(), // epoch
	).Return(&ethpb.DomainResponse{SignatureDomain: make([]byte, 32)}, nil /*err*/)

	m.validatorClient.EXPECT().GetSyncSubcommitteeIndex(
		gomock.Any(), // ctx
		&ethpb.SyncSubcommitteeIndexRequest{
			PublicKey: validatorKey.PublicKey().Marshal(),
			Slot:      1,
		},
	).Return(&ethpb.SyncSubcommitteeIndexResponse{}, nil /*err*/)

	roleMap, err := v.RolesAt(context.Background(), 1)
	require.NoError(t, err)

	assert.Equal(t, iface.RoleAttester, roleMap[bytesutil.ToBytes48(validatorKey.PublicKey().Marshal())][0])
	assert.Equal(t, iface.RoleAggregator, roleMap[bytesutil.ToBytes48(validatorKey.PublicKey().Marshal())][1])
	assert.Equal(t, iface.RoleSyncCommittee, roleMap[bytesutil.ToBytes48(validatorKey.PublicKey().Marshal())][2])

	// Test sync committee role at epoch boundary.
	v.duties = &ethpb.DutiesResponse{
		Duties: []*ethpb.DutiesResponse_Duty{
			{
				CommitteeIndex:  1,
				AttesterSlot:    1,
				PublicKey:       validatorKey.PublicKey().Marshal(),
				IsSyncCommittee: false,
			},
		},
		NextEpochDuties: []*ethpb.DutiesResponse_Duty{
			{
				CommitteeIndex:  1,
				AttesterSlot:    1,
				PublicKey:       validatorKey.PublicKey().Marshal(),
				IsSyncCommittee: true,
			},
		},
	}

	m.validatorClient.EXPECT().GetSyncSubcommitteeIndex(
		gomock.Any(), // ctx
		&ethpb.SyncSubcommitteeIndexRequest{
			PublicKey: validatorKey.PublicKey().Marshal(),
			Slot:      31,
		},
	).Return(&ethpb.SyncSubcommitteeIndexResponse{}, nil /*err*/)

	roleMap, err = v.RolesAt(context.Background(), params.BeaconConfig().SlotsPerEpoch-1)
	require.NoError(t, err)
	assert.Equal(t, iface.RoleSyncCommittee, roleMap[bytesutil.ToBytes48(validatorKey.PublicKey().Marshal())][0])
}

func TestRolesAt_DoesNotAssignProposer_Slot0(t *testing.T) {
	v, m, validatorKey, finish := setup(t)
	defer finish()

	v.duties = &ethpb.DutiesResponse{
		Duties: []*ethpb.DutiesResponse_Duty{
			{
				CommitteeIndex: 1,
				AttesterSlot:   0,
				ProposerSlots:  []types.Slot{0},
				PublicKey:      validatorKey.PublicKey().Marshal(),
			},
		},
	}

	m.validatorClient.EXPECT().DomainData(
		gomock.Any(), // ctx
		gomock.Any(), // epoch
	).Return(&ethpb.DomainResponse{SignatureDomain: make([]byte, 32)}, nil /*err*/)

	roleMap, err := v.RolesAt(context.Background(), 0)
	require.NoError(t, err)

	assert.Equal(t, iface.RoleAttester, roleMap[bytesutil.ToBytes48(validatorKey.PublicKey().Marshal())][0])
}

func TestCheckAndLogValidatorStatus_OK(t *testing.T) {
	nonexistentIndex := types.ValidatorIndex(^uint64(0))
	type statusTest struct {
		name   string
		status *validatorStatus
		log    string
		active bool
	}
	pubKeys := [][]byte{bytesutil.Uint64ToBytesLittleEndian(0)}
	tests := []statusTest{
		{
			name: "UNKNOWN_STATUS, no deposit found yet",
			status: &validatorStatus{
				publicKey: pubKeys[0],
				index:     nonexistentIndex,
				status: &ethpb.ValidatorStatusResponse{
					Status: ethpb.ValidatorStatus_UNKNOWN_STATUS,
				},
			},
			log:    "Waiting for deposit to be observed by beacon node",
			active: false,
		},
		{
			name: "DEPOSITED into state",
			status: &validatorStatus{
				publicKey: pubKeys[0],
				index:     30,
				status: &ethpb.ValidatorStatusResponse{
					Status:                    ethpb.ValidatorStatus_DEPOSITED,
					PositionInActivationQueue: 30,
				},
			},
			log:    "Deposit processed, entering activation queue after finalization\" index=30 positionInActivationQueue=30",
			active: false,
		},
		{
			name: "PENDING",
			status: &validatorStatus{
				publicKey: pubKeys[0],
				index:     50,
				status: &ethpb.ValidatorStatusResponse{
					Status:                    ethpb.ValidatorStatus_PENDING,
					ActivationEpoch:           params.BeaconConfig().FarFutureEpoch,
					PositionInActivationQueue: 6,
				},
			},
			log:    "Waiting to be assigned activation epoch\" index=50 positionInActivationQueue=6",
			active: false,
		},
		{
			name: "PENDING",
			status: &validatorStatus{
				publicKey: pubKeys[0],
				index:     89,
				status: &ethpb.ValidatorStatusResponse{
					Status:                    ethpb.ValidatorStatus_PENDING,
					ActivationEpoch:           60,
					PositionInActivationQueue: 5,
				},
			},
			log:    "Waiting for activation\" activationEpoch=60 index=89",
			active: false,
		},
		{
			name: "ACTIVE",
			status: &validatorStatus{
				publicKey: pubKeys[0],
				index:     89,
				status: &ethpb.ValidatorStatusResponse{
					Status: ethpb.ValidatorStatus_ACTIVE,
				},
			},
			active: true,
		},
		{
			name: "EXITING",
			status: &validatorStatus{
				publicKey: pubKeys[0],
				index:     89,
				status: &ethpb.ValidatorStatusResponse{
					Status: ethpb.ValidatorStatus_EXITING,
				},
			},
			active: true,
		},
		{
			name: "EXITED",
			status: &validatorStatus{
				publicKey: pubKeys[0],
				status: &ethpb.ValidatorStatusResponse{
					Status: ethpb.ValidatorStatus_EXITED,
				},
			},
			log:    "Validator exited",
			active: false,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			hook := logTest.NewGlobal()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			client := mock2.NewMockBeaconNodeValidatorClient(ctrl)
			v := validator{
				validatorClient: client,
				duties: &ethpb.DutiesResponse{
					Duties: []*ethpb.DutiesResponse_Duty{
						{
							CommitteeIndex: 1,
						},
					},
				},
			}

			active := v.checkAndLogValidatorStatus([]*validatorStatus{test.status})
			require.Equal(t, test.active, active)
			if test.log != "" {
				require.LogsContain(t, hook, test.log)
			}
		})
	}
}

func TestAllValidatorsAreExited_AllExited(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	client := mock2.NewMockBeaconNodeValidatorClient(ctrl)

	statuses := []*ethpb.ValidatorStatusResponse{
		{Status: ethpb.ValidatorStatus_EXITED},
		{Status: ethpb.ValidatorStatus_EXITED},
	}

	client.EXPECT().MultipleValidatorStatus(
		gomock.Any(), // ctx
		gomock.Any(), // request
	).Return(&ethpb.MultipleValidatorStatusResponse{Statuses: statuses}, nil /*err*/)

	v := validator{keyManager: genMockKeymanager(2), validatorClient: client}
	exited, err := v.AllValidatorsAreExited(context.Background())
	require.NoError(t, err)
	assert.Equal(t, true, exited)
}

func TestAllValidatorsAreExited_NotAllExited(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	client := mock2.NewMockBeaconNodeValidatorClient(ctrl)

	statuses := []*ethpb.ValidatorStatusResponse{
		{Status: ethpb.ValidatorStatus_ACTIVE},
		{Status: ethpb.ValidatorStatus_EXITED},
	}

	client.EXPECT().MultipleValidatorStatus(
		gomock.Any(), // ctx
		gomock.Any(), // request
	).Return(&ethpb.MultipleValidatorStatusResponse{Statuses: statuses}, nil /*err*/)

	v := validator{keyManager: genMockKeymanager(2), validatorClient: client}
	exited, err := v.AllValidatorsAreExited(context.Background())
	require.NoError(t, err)
	assert.Equal(t, false, exited)
}

func TestAllValidatorsAreExited_PartialResult(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	client := mock2.NewMockBeaconNodeValidatorClient(ctrl)

	statuses := []*ethpb.ValidatorStatusResponse{
		{Status: ethpb.ValidatorStatus_EXITED},
	}

	client.EXPECT().MultipleValidatorStatus(
		gomock.Any(), // ctx
		gomock.Any(), // request
	).Return(&ethpb.MultipleValidatorStatusResponse{Statuses: statuses}, nil /*err*/)

	v := validator{keyManager: genMockKeymanager(2), validatorClient: client}
	exited, err := v.AllValidatorsAreExited(context.Background())
	require.ErrorContains(t, "number of status responses did not match number of requested keys", err)
	assert.Equal(t, false, exited)
}

func TestAllValidatorsAreExited_NoKeys(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	client := mock2.NewMockBeaconNodeValidatorClient(ctrl)
	v := validator{keyManager: genMockKeymanager(0), validatorClient: client}
	exited, err := v.AllValidatorsAreExited(context.Background())
	require.NoError(t, err)
	assert.Equal(t, false, exited)
}

// TestAllValidatorsAreExited_CorrectRequest is a regression test that checks if the request contains the correct keys
func TestAllValidatorsAreExited_CorrectRequest(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	client := mock2.NewMockBeaconNodeValidatorClient(ctrl)

	// Create two different public keys
	pubKey0 := [fieldparams.BLSPubkeyLength]byte{1, 2, 3, 4}
	pubKey1 := [fieldparams.BLSPubkeyLength]byte{6, 7, 8, 9}
	// This is the request expected from AllValidatorsAreExited()
	request := &ethpb.MultipleValidatorStatusRequest{
		PublicKeys: [][]byte{
			pubKey0[:],
			pubKey1[:],
		},
	}
	statuses := []*ethpb.ValidatorStatusResponse{
		{Status: ethpb.ValidatorStatus_ACTIVE},
		{Status: ethpb.ValidatorStatus_EXITED},
	}

	client.EXPECT().MultipleValidatorStatus(
		gomock.Any(), // ctx
		request,      // request
	).Return(&ethpb.MultipleValidatorStatusResponse{Statuses: statuses}, nil /*err*/)

	keysMap := make(map[[fieldparams.BLSPubkeyLength]byte]bls.SecretKey)
	// secretKey below is just filler and is used multiple times
	secretKeyBytes := [32]byte{1}
	secretKey, err := bls.SecretKeyFromBytes(secretKeyBytes[:])
	require.NoError(t, err)
	keysMap[pubKey0] = secretKey
	keysMap[pubKey1] = secretKey

	// If AllValidatorsAreExited does not create the expected request, this test will fail
	v := validator{
		keyManager:             &mockKeymanager{keysMap: keysMap},
		validatorClient:        client,
		pubkeyToValidatorIndex: make(map[[fieldparams.BLSPubkeyLength]byte]types.ValidatorIndex),
	}
	exited, err := v.AllValidatorsAreExited(context.Background())
	require.NoError(t, err)
	assert.Equal(t, false, exited)
}

func TestService_ReceiveBlocks_NilBlock(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	valClient := mock2.NewMockBeaconNodeValidatorClient(ctrl)
	v := validator{
		blockFeed:       new(event.Feed),
		validatorClient: valClient,
	}
	stream := mock2.NewMockBeaconNodeValidatorAltair_StreamBlocksClient(ctrl)
	ctx, cancel := context.WithCancel(context.Background())
	valClient.EXPECT().StreamBlocksAltair(
		gomock.Any(),
		&ethpb.StreamBlocksRequest{VerifiedOnly: true},
	).Return(stream, nil)
	stream.EXPECT().Context().Return(ctx).AnyTimes()
	stream.EXPECT().Recv().Return(
		&ethpb.StreamBlocksResponse{Block: &ethpb.StreamBlocksResponse_Phase0Block{
			Phase0Block: &ethpb.SignedBeaconBlock{}}},
		nil,
	).Do(func() {
		cancel()
	})
	connectionErrorChannel := make(chan error)
	v.ReceiveBlocks(ctx, connectionErrorChannel)
	require.Equal(t, types.Slot(0), v.highestValidSlot)
}

func TestService_ReceiveBlocks_SetHighest(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	client := mock2.NewMockBeaconNodeValidatorClient(ctrl)

	v := validator{
		validatorClient: client,
		blockFeed:       new(event.Feed),
	}
	stream := mock2.NewMockBeaconNodeValidatorAltair_StreamBlocksClient(ctrl)
	ctx, cancel := context.WithCancel(context.Background())
	client.EXPECT().StreamBlocksAltair(
		gomock.Any(),
		&ethpb.StreamBlocksRequest{VerifiedOnly: true},
	).Return(stream, nil)
	stream.EXPECT().Context().Return(ctx).AnyTimes()
	slot := types.Slot(100)
	stream.EXPECT().Recv().Return(
		&ethpb.StreamBlocksResponse{
			Block: &ethpb.StreamBlocksResponse_Phase0Block{
				Phase0Block: &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{Slot: slot, Body: &ethpb.BeaconBlockBody{}}}},
		},
		nil,
	).Do(func() {
		cancel()
	})
	connectionErrorChannel := make(chan error)
	v.ReceiveBlocks(ctx, connectionErrorChannel)
	require.Equal(t, slot, v.highestValidSlot)
}

type doppelGangerRequestMatcher struct {
	req *ethpb.DoppelGangerRequest
}

var _ gomock.Matcher = (*doppelGangerRequestMatcher)(nil)

func (m *doppelGangerRequestMatcher) Matches(x interface{}) bool {
	r, ok := x.(*ethpb.DoppelGangerRequest)
	if !ok {
		panic("Invalid match type")
	}
	return gomock.InAnyOrder(m.req.ValidatorRequests).Matches(r.ValidatorRequests)
}

func (m *doppelGangerRequestMatcher) String() string {
	return fmt.Sprintf("%#v", m.req.ValidatorRequests)
}

func TestValidator_CheckDoppelGanger(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	flgs := features.Get()
	flgs.EnableDoppelGanger = true
	reset := features.InitWithReset(flgs)
	defer reset()
	tests := []struct {
		name            string
		validatorSetter func(t *testing.T) *validator
		err             string
	}{
		{
			name: "no doppelganger",
			validatorSetter: func(t *testing.T) *validator {
				client := mock2.NewMockBeaconNodeValidatorClient(ctrl)
				km := genMockKeymanager(10)
				keys, err := km.FetchValidatingPublicKeys(context.Background())
				assert.NoError(t, err)
				db := dbTest.SetupDB(t, keys)
				req := &ethpb.DoppelGangerRequest{ValidatorRequests: []*ethpb.DoppelGangerRequest_ValidatorRequest{}}
				resp := &ethpb.DoppelGangerRequest{ValidatorRequests: []*ethpb.DoppelGangerRequest_ValidatorRequest{}}
				for _, k := range keys {
					pkey := k
					att := createAttestation(10, 12)
					rt, err := att.Data.HashTreeRoot()
					assert.NoError(t, err)
					assert.NoError(t, db.SaveAttestationForPubKey(context.Background(), pkey, rt, att))
					resp.ValidatorRequests = append(resp.ValidatorRequests, &ethpb.DoppelGangerRequest_ValidatorRequest{PublicKey: pkey[:], Epoch: att.Data.Target.Epoch, SignedRoot: rt[:]})
					req.ValidatorRequests = append(req.ValidatorRequests, &ethpb.DoppelGangerRequest_ValidatorRequest{PublicKey: pkey[:], Epoch: att.Data.Target.Epoch, SignedRoot: rt[:]})
				}
				v := &validator{
					validatorClient: client,
					keyManager:      km,
					db:              db,
				}
				client.EXPECT().CheckDoppelGanger(
					gomock.Any(),                     // ctx
					&doppelGangerRequestMatcher{req}, // request
				).Return(nil, nil /*err*/)

				return v
			},
		},
		{
			name: "multiple doppelganger exists",
			validatorSetter: func(t *testing.T) *validator {
				client := mock2.NewMockBeaconNodeValidatorClient(ctrl)
				km := genMockKeymanager(10)
				keys, err := km.FetchValidatingPublicKeys(context.Background())
				assert.NoError(t, err)
				db := dbTest.SetupDB(t, keys)
				req := &ethpb.DoppelGangerRequest{ValidatorRequests: []*ethpb.DoppelGangerRequest_ValidatorRequest{}}
				resp := &ethpb.DoppelGangerResponse{Responses: []*ethpb.DoppelGangerResponse_ValidatorResponse{}}
				for i, k := range keys {
					pkey := k
					att := createAttestation(10, 12)
					rt, err := att.Data.HashTreeRoot()
					assert.NoError(t, err)
					assert.NoError(t, db.SaveAttestationForPubKey(context.Background(), pkey, rt, att))
					if i%3 == 0 {
						resp.Responses = append(resp.Responses, &ethpb.DoppelGangerResponse_ValidatorResponse{PublicKey: pkey[:], DuplicateExists: true})
					}
					req.ValidatorRequests = append(req.ValidatorRequests, &ethpb.DoppelGangerRequest_ValidatorRequest{PublicKey: pkey[:], Epoch: att.Data.Target.Epoch, SignedRoot: rt[:]})
				}
				v := &validator{
					validatorClient: client,
					keyManager:      km,
					db:              db,
				}
				client.EXPECT().CheckDoppelGanger(
					gomock.Any(),                     // ctx
					&doppelGangerRequestMatcher{req}, // request
				).Return(resp, nil /*err*/)
				return v
			},
			err: "Duplicate instances exists in the network for validator keys",
		},
		{
			name: "single doppelganger exists",
			validatorSetter: func(t *testing.T) *validator {
				client := mock2.NewMockBeaconNodeValidatorClient(ctrl)
				km := genMockKeymanager(10)
				keys, err := km.FetchValidatingPublicKeys(context.Background())
				assert.NoError(t, err)
				db := dbTest.SetupDB(t, keys)
				req := &ethpb.DoppelGangerRequest{ValidatorRequests: []*ethpb.DoppelGangerRequest_ValidatorRequest{}}
				resp := &ethpb.DoppelGangerResponse{Responses: []*ethpb.DoppelGangerResponse_ValidatorResponse{}}
				for i, k := range keys {
					pkey := k
					att := createAttestation(10, 12)
					rt, err := att.Data.HashTreeRoot()
					assert.NoError(t, err)
					assert.NoError(t, db.SaveAttestationForPubKey(context.Background(), pkey, rt, att))
					if i%9 == 0 {
						resp.Responses = append(resp.Responses, &ethpb.DoppelGangerResponse_ValidatorResponse{PublicKey: pkey[:], DuplicateExists: true})
					}
					req.ValidatorRequests = append(req.ValidatorRequests, &ethpb.DoppelGangerRequest_ValidatorRequest{PublicKey: pkey[:], Epoch: att.Data.Target.Epoch, SignedRoot: rt[:]})
				}
				v := &validator{
					validatorClient: client,
					keyManager:      km,
					db:              db,
				}
				client.EXPECT().CheckDoppelGanger(
					gomock.Any(),                     // ctx
					&doppelGangerRequestMatcher{req}, // request
				).Return(resp, nil /*err*/)
				return v
			},
			err: "Duplicate instances exists in the network for validator keys",
		},
		{
			name: "multiple attestations saved",
			validatorSetter: func(t *testing.T) *validator {
				client := mock2.NewMockBeaconNodeValidatorClient(ctrl)
				km := genMockKeymanager(10)
				keys, err := km.FetchValidatingPublicKeys(context.Background())
				assert.NoError(t, err)
				db := dbTest.SetupDB(t, keys)
				req := &ethpb.DoppelGangerRequest{ValidatorRequests: []*ethpb.DoppelGangerRequest_ValidatorRequest{}}
				resp := &ethpb.DoppelGangerResponse{Responses: []*ethpb.DoppelGangerResponse_ValidatorResponse{}}
				attLimit := 5
				for i, k := range keys {
					pkey := k
					for j := 0; j < attLimit; j++ {
						att := createAttestation(10+types.Epoch(j), 12+types.Epoch(j))
						rt, err := att.Data.HashTreeRoot()
						assert.NoError(t, err)
						assert.NoError(t, db.SaveAttestationForPubKey(context.Background(), pkey, rt, att))
						if j == attLimit-1 {
							req.ValidatorRequests = append(req.ValidatorRequests, &ethpb.DoppelGangerRequest_ValidatorRequest{PublicKey: pkey[:], Epoch: att.Data.Target.Epoch, SignedRoot: rt[:]})
						}
					}
					if i%3 == 0 {
						resp.Responses = append(resp.Responses, &ethpb.DoppelGangerResponse_ValidatorResponse{PublicKey: pkey[:], DuplicateExists: true})
					}
				}
				v := &validator{
					validatorClient: client,
					keyManager:      km,
					db:              db,
				}
				client.EXPECT().CheckDoppelGanger(
					gomock.Any(),                     // ctx
					&doppelGangerRequestMatcher{req}, // request
				).Return(resp, nil /*err*/)
				return v
			},
			err: "Duplicate instances exists in the network for validator keys",
		},
		{
			name: "no history exists",
			validatorSetter: func(t *testing.T) *validator {
				client := mock2.NewMockBeaconNodeValidatorClient(ctrl)
				// Use only 1 key for deterministic order.
				km := genMockKeymanager(1)
				keys, err := km.FetchValidatingPublicKeys(context.Background())
				assert.NoError(t, err)
				db := dbTest.SetupDB(t, keys)
				resp := &ethpb.DoppelGangerResponse{Responses: []*ethpb.DoppelGangerResponse_ValidatorResponse{}}
				req := &ethpb.DoppelGangerRequest{ValidatorRequests: []*ethpb.DoppelGangerRequest_ValidatorRequest{}}
				for _, k := range keys {
					resp.Responses = append(resp.Responses, &ethpb.DoppelGangerResponse_ValidatorResponse{PublicKey: k[:], DuplicateExists: false})
					req.ValidatorRequests = append(req.ValidatorRequests, &ethpb.DoppelGangerRequest_ValidatorRequest{PublicKey: k[:], SignedRoot: make([]byte, 32), Epoch: 0})
				}
				v := &validator{
					validatorClient: client,
					keyManager:      km,
					db:              db,
				}
				client.EXPECT().CheckDoppelGanger(
					gomock.Any(), // ctx
					req,          // request
				).Return(resp, nil /*err*/)
				return v
			},
			err: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := tt.validatorSetter(t)
			if err := v.CheckDoppelGanger(context.Background()); tt.err != "" {
				assert.ErrorContains(t, tt.err, err)
			}
		})
	}
}

func TestValidatorAttestationsAreOrdered(t *testing.T) {
	km := genMockKeymanager(10)
	keys, err := km.FetchValidatingPublicKeys(context.Background())
	assert.NoError(t, err)
	db := dbTest.SetupDB(t, keys)

	k := keys[0]
	att := createAttestation(10, 14)
	rt, err := att.Data.HashTreeRoot()
	assert.NoError(t, err)
	assert.NoError(t, db.SaveAttestationForPubKey(context.Background(), k, rt, att))

	att = createAttestation(6, 8)
	rt, err = att.Data.HashTreeRoot()
	assert.NoError(t, err)
	assert.NoError(t, db.SaveAttestationForPubKey(context.Background(), k, rt, att))

	att = createAttestation(10, 12)
	rt, err = att.Data.HashTreeRoot()
	assert.NoError(t, err)
	assert.NoError(t, db.SaveAttestationForPubKey(context.Background(), k, rt, att))

	att = createAttestation(2, 3)
	rt, err = att.Data.HashTreeRoot()
	assert.NoError(t, err)
	assert.NoError(t, db.SaveAttestationForPubKey(context.Background(), k, rt, att))

	histories, err := db.AttestationHistoryForPubKey(context.Background(), k)
	assert.NoError(t, err)
	r := retrieveLatestRecord(histories)
	assert.Equal(t, r.Target, types.Epoch(14))
}

func createAttestation(source, target types.Epoch) *ethpb.IndexedAttestation {
	return &ethpb.IndexedAttestation{
		Data: &ethpb.AttestationData{
			Source: &ethpb.Checkpoint{
				Epoch: source,
				Root:  make([]byte, 32),
			},
			Target: &ethpb.Checkpoint{
				Epoch: target,
				Root:  make([]byte, 32),
			},
			BeaconBlockRoot: make([]byte, 32),
		},
		Signature: make([]byte, fieldparams.BLSSignatureLength),
	}
}

func TestIsSyncCommitteeAggregator_OK(t *testing.T) {
	v, m, validatorKey, finish := setup(t)
	defer finish()

	slot := types.Slot(1)
	pubKey := validatorKey.PublicKey().Marshal()

	m.validatorClient.EXPECT().GetSyncSubcommitteeIndex(
		gomock.Any(), // ctx
		&ethpb.SyncSubcommitteeIndexRequest{
			PublicKey: validatorKey.PublicKey().Marshal(),
			Slot:      1,
		},
	).Return(&ethpb.SyncSubcommitteeIndexResponse{}, nil /*err*/)

	aggregator, err := v.isSyncCommitteeAggregator(context.Background(), slot, bytesutil.ToBytes48(pubKey))
	require.NoError(t, err)
	require.Equal(t, false, aggregator)

	c := params.BeaconConfig()
	c.TargetAggregatorsPerSyncSubcommittee = math.MaxUint64
	params.OverrideBeaconConfig(c)

	m.validatorClient.EXPECT().DomainData(
		gomock.Any(), // ctx
		gomock.Any(), // epoch
	).Return(&ethpb.DomainResponse{SignatureDomain: make([]byte, 32)}, nil /*err*/)

	m.validatorClient.EXPECT().GetSyncSubcommitteeIndex(
		gomock.Any(), // ctx
		&ethpb.SyncSubcommitteeIndexRequest{
			PublicKey: validatorKey.PublicKey().Marshal(),
			Slot:      1,
		},
	).Return(&ethpb.SyncSubcommitteeIndexResponse{Indices: []types.CommitteeIndex{0}}, nil /*err*/)

	aggregator, err = v.isSyncCommitteeAggregator(context.Background(), slot, bytesutil.ToBytes48(pubKey))
	require.NoError(t, err)
	require.Equal(t, true, aggregator)
}

func TestValidator_WaitForKeymanagerInitialization_web3Signer(t *testing.T) {
	ctx := context.Background()
	db := dbTest.SetupDB(t, [][fieldparams.BLSPubkeyLength]byte{})
	root := make([]byte, 32)
	copy(root[2:], "a")
	err := db.SaveGenesisValidatorsRoot(ctx, root)
	require.NoError(t, err)
	w := wallet.NewWalletForWeb3Signer()
	decodedKey, err := hexutil.Decode("0xa2b5aaad9c6efefe7bb9b1243a043404f3362937cfb6b31833929833173f476630ea2cfeb0d9ddf15f97ca8685948820")
	require.NoError(t, err)
	keys := [][48]byte{
		bytesutil.ToBytes48(decodedKey),
	}
	v := validator{
		db:     db,
		useWeb: false,
		wallet: w,
		Web3SignerConfig: &remote_web3signer.SetupConfig{
			BaseEndpoint:       "http://localhost:8545",
			ProvidedPublicKeys: keys,
		},
	}
	err = v.WaitForKeymanagerInitialization(context.Background())
	require.NoError(t, err)
	km, err := v.Keymanager()
	require.NoError(t, err)
	require.NotNil(t, km)
}

func TestValidator_WaitForKeymanagerInitialization_Web(t *testing.T) {
	ctx := context.Background()
	db := dbTest.SetupDB(t, [][fieldparams.BLSPubkeyLength]byte{})
	root := make([]byte, 32)
	copy(root[2:], "a")
	err := db.SaveGenesisValidatorsRoot(ctx, root)
	require.NoError(t, err)
	walletChan := make(chan *wallet.Wallet, 1)
	v := validator{
		db:                      db,
		useWeb:                  true,
		walletInitializedFeed:   &event.Feed{},
		walletIntializedChannel: walletChan,
	}
	wait := make(chan struct{})
	go func() {
		defer close(wait)
		err = v.WaitForKeymanagerInitialization(ctx)
		require.NoError(t, err)
		km, err := v.Keymanager()
		require.NoError(t, err)
		require.NotNil(t, km)
	}()

	walletChan <- wallet.New(&wallet.Config{
		KeymanagerKind: keymanager.Local,
	})
	<-wait
}

func TestValidator_WaitForKeymanagerInitialization_Interop(t *testing.T) {
	ctx := context.Background()
	db := dbTest.SetupDB(t, [][fieldparams.BLSPubkeyLength]byte{})
	root := make([]byte, 32)
	copy(root[2:], "a")
	err := db.SaveGenesisValidatorsRoot(ctx, root)
	require.NoError(t, err)
	v := validator{
		db:     db,
		useWeb: false,
		interopKeysConfig: &local.InteropKeymanagerConfig{
			NumValidatorKeys: 2,
			Offset:           1,
		},
	}
	err = v.WaitForKeymanagerInitialization(ctx)
	require.NoError(t, err)
	km, err := v.Keymanager()
	require.NoError(t, err)
	require.NotNil(t, km)
}

func TestValidator_UdpateFeeRecipient(t *testing.T) {
	ctrl := gomock.NewController(t)
	ctx := context.Background()
	db := dbTest.SetupDB(t, [][fieldparams.BLSPubkeyLength]byte{})
	client := mock2.NewMockBeaconNodeValidatorClient(ctrl)
	tests := []struct {
		name            string
		validatorSetter func(t *testing.T) *validator
		err             string
	}{
		{
			name: " Happy Path",
			validatorSetter: func(t *testing.T) *validator {

				v := validator{
					validatorClient:        client,
					db:                     db,
					pubkeyToValidatorIndex: make(map[[fieldparams.BLSPubkeyLength]byte]types.ValidatorIndex),
					useWeb:                 false,
					interopKeysConfig: &local.InteropKeymanagerConfig{
						NumValidatorKeys: 1,
						Offset:           1,
					},
				}

				err := v.WaitForKeymanagerInitialization(ctx)
				require.NoError(t, err)
				km, err := v.Keymanager()
				require.NoError(t, err)
				keys, err := km.FetchValidatingPublicKeys(ctx)
				require.NoError(t, err)
				v.feeRecipientConfig = &validator_service_config.FeeRecipientConfig{
					ProposeConfig: nil,
					DefaultConfig: &validator_service_config.FeeRecipientOptions{
						FeeRecipient: common.HexToAddress("0x046Fb65722E7b2455043BFEBf6177F1D2e9738D9"),
					},
				}
				client.EXPECT().ValidatorIndex(
					ctx, // ctx
					&ethpb.ValidatorIndexRequest{PublicKey: keys[0][:]},
				).Return(&ethpb.ValidatorIndexResponse{
					Index: 1,
				}, nil)

				return &v
			},
		},
		{
			name: " Skip if no config",
			validatorSetter: func(t *testing.T) *validator {

				v := validator{
					validatorClient:        client,
					db:                     db,
					pubkeyToValidatorIndex: make(map[[fieldparams.BLSPubkeyLength]byte]types.ValidatorIndex),
					useWeb:                 false,
					interopKeysConfig: &local.InteropKeymanagerConfig{
						NumValidatorKeys: 1,
						Offset:           1,
					},
				}
				err := v.WaitForKeymanagerInitialization(ctx)
				require.NoError(t, err)
				return &v
			},
		},
		{
			name: " Happy Path validator index not found in cache",
			validatorSetter: func(t *testing.T) *validator {

				v := validator{
					validatorClient:        client,
					db:                     db,
					pubkeyToValidatorIndex: make(map[[fieldparams.BLSPubkeyLength]byte]types.ValidatorIndex),
					useWeb:                 false,
					interopKeysConfig: &local.InteropKeymanagerConfig{
						NumValidatorKeys: 1,
						Offset:           1,
					},
				}
				err := v.WaitForKeymanagerInitialization(ctx)
				require.NoError(t, err)
				v.feeRecipientConfig = &validator_service_config.FeeRecipientConfig{
					ProposeConfig: nil,
					DefaultConfig: &validator_service_config.FeeRecipientOptions{
						FeeRecipient: common.HexToAddress("0x046Fb65722E7b2455043BFEBf6177F1D2e9738D9"),
					},
				}
				km, err := v.Keymanager()
				require.NoError(t, err)
				keys, err := km.FetchValidatingPublicKeys(ctx)
				require.NoError(t, err)
				client.EXPECT().ValidatorIndex(
					ctx, // ctx
					&ethpb.ValidatorIndexRequest{PublicKey: keys[0][:]},
				).Return(&ethpb.ValidatorIndexResponse{
					Index: 1,
				}, nil)
				return &v
			},
		},
		{
			name: " Happy Path proposer config not nil",
			validatorSetter: func(t *testing.T) *validator {

				v := validator{
					validatorClient:        client,
					db:                     db,
					pubkeyToValidatorIndex: make(map[[fieldparams.BLSPubkeyLength]byte]types.ValidatorIndex),
					useWeb:                 false,
					interopKeysConfig: &local.InteropKeymanagerConfig{
						NumValidatorKeys: 1,
						Offset:           1,
					},
				}
				err := v.WaitForKeymanagerInitialization(ctx)
				require.NoError(t, err)
				config := make(map[[fieldparams.BLSPubkeyLength]byte]*validator_service_config.FeeRecipientOptions)
				km, err := v.Keymanager()
				require.NoError(t, err)
				keys, err := km.FetchValidatingPublicKeys(ctx)
				require.NoError(t, err)
				client.EXPECT().ValidatorIndex(
					ctx, // ctx
					&ethpb.ValidatorIndexRequest{PublicKey: keys[0][:]},
				).Return(&ethpb.ValidatorIndexResponse{
					Index: 1,
				}, nil)
				config[keys[0]] = &validator_service_config.FeeRecipientOptions{
					FeeRecipient: common.HexToAddress("0x046Fb65722E7b2455043BFEBf6177F1D2e9738D9"),
				}
				v.feeRecipientConfig = &validator_service_config.FeeRecipientConfig{
					ProposeConfig: config,
					DefaultConfig: &validator_service_config.FeeRecipientOptions{
						FeeRecipient: common.HexToAddress("0x046Fb65722E7b2455043BFEBf6177F1D2e9738D9"),
					},
				}
				return &v
			},
		},
		{
			name: " proposer config not nil but fee recipient empty ",
			validatorSetter: func(t *testing.T) *validator {

				v := validator{
					validatorClient:        client,
					db:                     db,
					pubkeyToValidatorIndex: make(map[[fieldparams.BLSPubkeyLength]byte]types.ValidatorIndex),
					useWeb:                 false,
					interopKeysConfig: &local.InteropKeymanagerConfig{
						NumValidatorKeys: 1,
						Offset:           1,
					},
				}
				err := v.WaitForKeymanagerInitialization(ctx)
				require.NoError(t, err)
				config := make(map[[fieldparams.BLSPubkeyLength]byte]*validator_service_config.FeeRecipientOptions)
				km, err := v.Keymanager()
				require.NoError(t, err)
				keys, err := km.FetchValidatingPublicKeys(ctx)
				require.NoError(t, err)
				client.EXPECT().ValidatorIndex(
					ctx, // ctx
					&ethpb.ValidatorIndexRequest{PublicKey: keys[0][:]},
				).Return(&ethpb.ValidatorIndexResponse{
					Index: 1,
				}, nil)
				config[keys[0]] = &validator_service_config.FeeRecipientOptions{
					FeeRecipient: common.Address{},
				}
				v.feeRecipientConfig = &validator_service_config.FeeRecipientConfig{
					ProposeConfig: config,
					DefaultConfig: &validator_service_config.FeeRecipientOptions{
						FeeRecipient: common.HexToAddress("0x046Fb65722E7b2455043BFEBf6177F1D2e9738D9"),
					},
				}
				return &v
			},
		},
		{
			name: "Validator index not found with proposeconfig",
			validatorSetter: func(t *testing.T) *validator {

				v := validator{
					validatorClient:        client,
					db:                     db,
					pubkeyToValidatorIndex: make(map[[fieldparams.BLSPubkeyLength]byte]types.ValidatorIndex),
					useWeb:                 false,
					interopKeysConfig: &local.InteropKeymanagerConfig{
						NumValidatorKeys: 1,
						Offset:           1,
					},
				}
				err := v.WaitForKeymanagerInitialization(ctx)
				require.NoError(t, err)
				config := make(map[[fieldparams.BLSPubkeyLength]byte]*validator_service_config.FeeRecipientOptions)
				km, err := v.Keymanager()
				require.NoError(t, err)
				keys, err := km.FetchValidatingPublicKeys(ctx)
				require.NoError(t, err)
				client.EXPECT().ValidatorIndex(
					ctx, // ctx
					&ethpb.ValidatorIndexRequest{PublicKey: keys[0][:]},
				).Return(nil, errors.New("Could not find validator index for public key"))
				config[keys[0]] = &validator_service_config.FeeRecipientOptions{
					FeeRecipient: common.HexToAddress("0x046Fb65722E7b2455043BFEBf6177F1D2e9738D9"),
				}
				v.feeRecipientConfig = &validator_service_config.FeeRecipientConfig{
					ProposeConfig: config,
					DefaultConfig: &validator_service_config.FeeRecipientOptions{
						FeeRecipient: common.HexToAddress("0x046Fb65722E7b2455043BFEBf6177F1D2e9738D9"),
					},
				}
				return &v
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := tt.validatorSetter(t)
			km, err := v.Keymanager()
			require.NoError(t, err)
			if err := v.UpdateFeeRecipient(ctx, km); tt.err != "" {
				assert.ErrorContains(t, tt.err, err)
			}
		})
	}
}
