package client

import (
	"context"
	"errors"
	"io/ioutil"
	"reflect"
	"strings"
	"testing"
	"time"

	ptypes "github.com/gogo/protobuf/types"
	"github.com/golang/mock/gomock"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	slashpb "github.com/prysmaticlabs/prysm/proto/slashing"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/mock"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	dbTest "github.com/prysmaticlabs/prysm/validator/db/testing"
	keymanager "github.com/prysmaticlabs/prysm/validator/keymanager/v1"
	"github.com/sirupsen/logrus"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func init() {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetOutput(ioutil.Discard)
}

var _ = Validator(&validator{})

const cancelledCtx = "context has been canceled"

func publicKeys(km keymanager.KeyManager) [][]byte {
	keys, err := km.FetchValidatingKeys()
	if err != nil {
		log.WithError(err).Debug("Cannot fetch validating keys")
	}
	res := make([][]byte, len(keys))
	for i := range keys {
		res[i] = keys[i][:]
	}
	return res
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

func TestWaitForChainStart_SetsChainStartGenesisTime(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	client := mock.NewMockBeaconNodeValidatorClient(ctrl)

	v := validator{
		keyManager:      testKeyManager,
		validatorClient: client,
	}
	genesis := uint64(time.Unix(1, 0).Unix())
	clientStream := mock.NewMockBeaconNodeValidator_WaitForChainStartClient(ctrl)
	client.EXPECT().WaitForChainStart(
		gomock.Any(),
		&ptypes.Empty{},
	).Return(clientStream, nil)
	clientStream.EXPECT().Recv().Return(
		&ethpb.ChainStartResponse{
			Started:     true,
			GenesisTime: genesis,
		},
		nil,
	)
	if err := v.WaitForChainStart(context.Background()); err != nil {
		t.Fatal(err)
	}
	if v.genesisTime != genesis {
		t.Errorf("Expected chain start time to equal %d, received %d", genesis, v.genesisTime)
	}
	if v.ticker == nil {
		t.Error("Expected ticker to be set, received nil")
	}
}

func TestWaitForChainStart_ContextCanceled(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	client := mock.NewMockBeaconNodeValidatorClient(ctrl)

	v := validator{
		keyManager:      testKeyManager,
		validatorClient: client,
	}
	genesis := uint64(time.Unix(0, 0).Unix())
	clientStream := mock.NewMockBeaconNodeValidator_WaitForChainStartClient(ctrl)
	client.EXPECT().WaitForChainStart(
		gomock.Any(),
		&ptypes.Empty{},
	).Return(clientStream, nil)
	clientStream.EXPECT().Recv().Return(
		&ethpb.ChainStartResponse{
			Started:     true,
			GenesisTime: genesis,
		},
		nil,
	)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := v.WaitForChainStart(ctx)
	want := cancelledCtx
	if !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %v, received %v", want, err)
	}
}

func TestWaitForChainStart_StreamSetupFails(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	client := mock.NewMockBeaconNodeValidatorClient(ctrl)

	v := validator{
		keyManager:      testKeyManager,
		validatorClient: client,
	}
	clientStream := mock.NewMockBeaconNodeValidator_WaitForChainStartClient(ctrl)
	client.EXPECT().WaitForChainStart(
		gomock.Any(),
		&ptypes.Empty{},
	).Return(clientStream, errors.New("failed stream"))
	err := v.WaitForChainStart(context.Background())
	want := "could not setup beacon chain ChainStart streaming client"
	if !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %v, received %v", want, err)
	}
}

func TestWaitForChainStart_ReceiveErrorFromStream(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	client := mock.NewMockBeaconNodeValidatorClient(ctrl)

	v := validator{
		keyManager:      testKeyManager,
		validatorClient: client,
	}
	clientStream := mock.NewMockBeaconNodeValidator_WaitForChainStartClient(ctrl)
	client.EXPECT().WaitForChainStart(
		gomock.Any(),
		&ptypes.Empty{},
	).Return(clientStream, nil)
	clientStream.EXPECT().Recv().Return(
		nil,
		errors.New("fails"),
	)
	err := v.WaitForChainStart(context.Background())
	want := "could not receive ChainStart from stream"
	if !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %v, received %v", want, err)
	}
}

func TestWaitForSynced_SetsGenesisTime(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	client := mock.NewMockBeaconNodeValidatorClient(ctrl)

	v := validator{
		keyManager:      testKeyManager,
		validatorClient: client,
	}
	genesis := uint64(time.Unix(1, 0).Unix())
	clientStream := mock.NewMockBeaconNodeValidator_WaitForSyncedClient(ctrl)
	client.EXPECT().WaitForSynced(
		gomock.Any(),
		&ptypes.Empty{},
	).Return(clientStream, nil)
	clientStream.EXPECT().Recv().Return(
		&ethpb.SyncedResponse{
			Synced:      true,
			GenesisTime: genesis,
		},
		nil,
	)
	if err := v.WaitForSynced(context.Background()); err != nil {
		t.Fatal(err)
	}
	if v.genesisTime != genesis {
		t.Errorf("Expected chain start time to equal %d, received %d", genesis, v.genesisTime)
	}
	if v.ticker == nil {
		t.Error("Expected ticker to be set, received nil")
	}
}

func TestWaitForSynced_ContextCanceled(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	client := mock.NewMockBeaconNodeValidatorClient(ctrl)

	v := validator{
		keyManager:      testKeyManager,
		validatorClient: client,
	}
	genesis := uint64(time.Unix(0, 0).Unix())
	clientStream := mock.NewMockBeaconNodeValidator_WaitForSyncedClient(ctrl)
	client.EXPECT().WaitForSynced(
		gomock.Any(),
		&ptypes.Empty{},
	).Return(clientStream, nil)
	clientStream.EXPECT().Recv().Return(
		&ethpb.SyncedResponse{
			Synced:      true,
			GenesisTime: genesis,
		},
		nil,
	)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := v.WaitForSynced(ctx)
	want := cancelledCtx
	if !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %v, received %v", want, err)
	}
}

func TestWaitForSynced_StreamSetupFails(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	client := mock.NewMockBeaconNodeValidatorClient(ctrl)

	v := validator{
		keyManager:      testKeyManager,
		validatorClient: client,
	}
	clientStream := mock.NewMockBeaconNodeValidator_WaitForSyncedClient(ctrl)
	client.EXPECT().WaitForSynced(
		gomock.Any(),
		&ptypes.Empty{},
	).Return(clientStream, errors.New("failed stream"))
	err := v.WaitForSynced(context.Background())
	want := "could not setup beacon chain Synced streaming client"
	if !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %v, received %v", want, err)
	}
}

func TestWaitForSynced_ReceiveErrorFromStream(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	client := mock.NewMockBeaconNodeValidatorClient(ctrl)

	v := validator{
		keyManager:      testKeyManager,
		validatorClient: client,
	}
	clientStream := mock.NewMockBeaconNodeValidator_WaitForSyncedClient(ctrl)
	client.EXPECT().WaitForSynced(
		gomock.Any(),
		&ptypes.Empty{},
	).Return(clientStream, nil)
	clientStream.EXPECT().Recv().Return(
		nil,
		errors.New("fails"),
	)
	err := v.WaitForSynced(context.Background())
	want := "could not receive Synced from stream"
	if !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %v, received %v", want, err)
	}
}

func TestWaitActivation_ContextCanceled(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	client := mock.NewMockBeaconNodeValidatorClient(ctrl)

	v := validator{
		keyManager:      testKeyManager,
		validatorClient: client,
	}
	clientStream := mock.NewMockBeaconNodeValidator_WaitForActivationClient(ctrl)

	client.EXPECT().WaitForActivation(
		gomock.Any(),
		&ethpb.ValidatorActivationRequest{
			PublicKeys: publicKeys(v.keyManager),
		},
	).Return(clientStream, nil)
	clientStream.EXPECT().Recv().Return(
		&ethpb.ValidatorActivationResponse{},
		nil,
	)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := v.WaitForActivation(ctx)
	want := cancelledCtx
	if !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %v, received %v", want, err)
	}
}

func TestWaitActivation_StreamSetupFails(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	client := mock.NewMockBeaconNodeValidatorClient(ctrl)

	v := validator{
		keyManager:      testKeyManager,
		validatorClient: client,
	}
	clientStream := mock.NewMockBeaconNodeValidator_WaitForActivationClient(ctrl)
	client.EXPECT().WaitForActivation(
		gomock.Any(),
		&ethpb.ValidatorActivationRequest{
			PublicKeys: publicKeys(v.keyManager),
		},
	).Return(clientStream, errors.New("failed stream"))
	err := v.WaitForActivation(context.Background())
	want := "could not setup validator WaitForActivation streaming client"
	if !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %v, received %v", want, err)
	}
}

func TestWaitActivation_ReceiveErrorFromStream(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	client := mock.NewMockBeaconNodeValidatorClient(ctrl)

	v := validator{
		keyManager:      testKeyManager,
		validatorClient: client,
	}
	clientStream := mock.NewMockBeaconNodeValidator_WaitForActivationClient(ctrl)
	client.EXPECT().WaitForActivation(
		gomock.Any(),
		&ethpb.ValidatorActivationRequest{
			PublicKeys: publicKeys(v.keyManager),
		},
	).Return(clientStream, nil)
	clientStream.EXPECT().Recv().Return(
		nil,
		errors.New("fails"),
	)
	err := v.WaitForActivation(context.Background())
	want := "could not receive validator activation from stream"
	if !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %v, received %v", want, err)
	}
}

func TestWaitActivation_LogsActivationEpochOK(t *testing.T) {
	hook := logTest.NewGlobal()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	client := mock.NewMockBeaconNodeValidatorClient(ctrl)

	v := validator{
		keyManager:      testKeyManager,
		validatorClient: client,
		genesisTime:     1,
	}
	resp := generateMockStatusResponse(publicKeys(v.keyManager))
	resp.Statuses[0].Status.Status = ethpb.ValidatorStatus_ACTIVE
	clientStream := mock.NewMockBeaconNodeValidator_WaitForActivationClient(ctrl)
	client.EXPECT().WaitForActivation(
		gomock.Any(),
		&ethpb.ValidatorActivationRequest{
			PublicKeys: publicKeys(v.keyManager),
		},
	).Return(clientStream, nil)
	clientStream.EXPECT().Recv().Return(
		resp,
		nil,
	)
	if err := v.WaitForActivation(context.Background()); err != nil {
		t.Errorf("Could not wait for activation: %v", err)
	}
	testutil.AssertLogsContain(t, hook, "Validator activated")
}

func TestCanonicalHeadSlot_FailedRPC(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	client := mock.NewMockBeaconChainClient(ctrl)
	v := validator{
		keyManager:   testKeyManager,
		beaconClient: client,
		genesisTime:  1,
	}
	client.EXPECT().GetChainHead(
		gomock.Any(),
		gomock.Any(),
	).Return(nil, errors.New("failed"))
	if _, err := v.CanonicalHeadSlot(context.Background()); !strings.Contains(err.Error(), "failed") {
		t.Errorf("Wanted: %v, received: %v", "failed", err)
	}
}

func TestCanonicalHeadSlot_OK(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	client := mock.NewMockBeaconChainClient(ctrl)
	v := validator{
		keyManager:   testKeyManager,
		beaconClient: client,
	}
	client.EXPECT().GetChainHead(
		gomock.Any(),
		gomock.Any(),
	).Return(&ethpb.ChainHead{HeadSlot: 0}, nil)
	headSlot, err := v.CanonicalHeadSlot(context.Background())
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if headSlot != 0 {
		t.Errorf("Mismatch slots, wanted: %v, received: %v", 0, headSlot)
	}
}

func TestWaitMultipleActivation_LogsActivationEpochOK(t *testing.T) {
	hook := logTest.NewGlobal()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	client := mock.NewMockBeaconNodeValidatorClient(ctrl)

	v := validator{
		keyManager:      testKeyManagerThreeValidators,
		validatorClient: client,
		genesisTime:     1,
	}
	publicKeys := publicKeys(v.keyManager)
	resp := generateMockStatusResponse(publicKeys)
	resp.Statuses[0].Status.Status = ethpb.ValidatorStatus_ACTIVE
	resp.Statuses[1].Status.Status = ethpb.ValidatorStatus_ACTIVE
	clientStream := mock.NewMockBeaconNodeValidator_WaitForActivationClient(ctrl)
	client.EXPECT().WaitForActivation(
		gomock.Any(),
		&ethpb.ValidatorActivationRequest{
			PublicKeys: publicKeys,
		},
	).Return(clientStream, nil)
	clientStream.EXPECT().Recv().Return(
		resp,
		nil,
	)
	if err := v.WaitForActivation(context.Background()); err != nil {
		t.Errorf("Could not wait for activation: %v", err)
	}
	testutil.AssertLogsContain(t, hook, "Validator activated")
}

func TestWaitActivation_NotAllValidatorsActivatedOK(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	client := mock.NewMockBeaconNodeValidatorClient(ctrl)

	v := validator{
		keyManager:      testKeyManagerThreeValidators,
		validatorClient: client,
		genesisTime:     1,
	}
	resp := generateMockStatusResponse(publicKeys(v.keyManager))
	resp.Statuses[0].Status.Status = ethpb.ValidatorStatus_ACTIVE
	clientStream := mock.NewMockBeaconNodeValidator_WaitForActivationClient(ctrl)
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
	if err := v.WaitForActivation(context.Background()); err != nil {
		t.Errorf("Could not wait for activation: %v", err)
	}
}

func TestWaitSync_ContextCanceled(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	n := mock.NewMockNodeClient(ctrl)

	v := validator{
		node: n,
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	n.EXPECT().GetSyncStatus(
		gomock.Any(),
		gomock.Any(),
	).Return(&ethpb.SyncStatus{Syncing: true}, nil)

	err := v.WaitForSync(ctx)
	want := cancelledCtx
	if !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %v, received %v", want, err)
	}
}

func TestWaitSync_NotSyncing(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	n := mock.NewMockNodeClient(ctrl)

	v := validator{
		node: n,
	}

	n.EXPECT().GetSyncStatus(
		gomock.Any(),
		gomock.Any(),
	).Return(&ethpb.SyncStatus{Syncing: false}, nil)

	err := v.WaitForSync(context.Background())
	if err != nil {
		t.Fatal(err)
	}
}

func TestWaitSync_Syncing(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	n := mock.NewMockNodeClient(ctrl)

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

	err := v.WaitForSync(context.Background())
	if err != nil {
		t.Fatal(err)
	}
}

func TestUpdateDuties_DoesNothingWhenNotEpochStart_AlreadyExistingAssignments(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	client := mock.NewMockBeaconNodeValidatorClient(ctrl)

	slot := uint64(1)
	v := validator{
		keyManager:      testKeyManager,
		validatorClient: client,
		duties: &ethpb.DutiesResponse{
			Duties: []*ethpb.DutiesResponse_Duty{
				{
					Committee:      []uint64{},
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

	if err := v.UpdateDuties(context.Background(), slot); err != nil {
		t.Errorf("Could not update assignments: %v", err)
	}
}

func TestUpdateDuties_ReturnsError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	client := mock.NewMockBeaconNodeValidatorClient(ctrl)

	v := validator{
		keyManager:      testKeyManager,
		validatorClient: client,
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

	if err := v.UpdateDuties(context.Background(), params.BeaconConfig().SlotsPerEpoch); err != expected {
		t.Errorf("Bad error; want=%v got=%v", expected, err)
	}
	if v.duties != nil {
		t.Error("Assignments should have been cleared on failure")
	}
}

func TestUpdateDuties_OK(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	client := mock.NewMockBeaconNodeValidatorClient(ctrl)

	slot := params.BeaconConfig().SlotsPerEpoch
	resp := &ethpb.DutiesResponse{
		Duties: []*ethpb.DutiesResponse_Duty{
			{
				AttesterSlot:   params.BeaconConfig().SlotsPerEpoch,
				ValidatorIndex: 200,
				CommitteeIndex: 100,
				Committee:      []uint64{0, 1, 2, 3},
				PublicKey:      []byte("testPubKey_1"),
				ProposerSlots:  []uint64{params.BeaconConfig().SlotsPerEpoch + 1},
			},
		},
	}
	v := validator{
		keyManager:      testKeyManager,
		validatorClient: client,
	}
	client.EXPECT().GetDuties(
		gomock.Any(),
		gomock.Any(),
	).Return(resp, nil)

	client.EXPECT().GetDuties(
		gomock.Any(),
		gomock.Any(),
	).Return(resp, nil)

	client.EXPECT().SubscribeCommitteeSubnets(
		gomock.Any(),
		gomock.Any(),
	).Return(nil, nil)

	if err := v.UpdateDuties(context.Background(), slot); err != nil {
		t.Fatalf("Could not update assignments: %v", err)
	}
	if v.duties.Duties[0].ProposerSlots[0] != params.BeaconConfig().SlotsPerEpoch+1 {
		t.Errorf(
			"Unexpected validator assignments. want=%v got=%v",
			params.BeaconConfig().SlotsPerEpoch+1,
			v.duties.Duties[0].ProposerSlots[0],
		)
	}
	if v.duties.Duties[0].AttesterSlot != params.BeaconConfig().SlotsPerEpoch {
		t.Errorf(
			"Unexpected validator assignments. want=%v got=%v",
			params.BeaconConfig().SlotsPerEpoch,
			v.duties.Duties[0].AttesterSlot,
		)
	}
	if v.duties.Duties[0].CommitteeIndex != resp.Duties[0].CommitteeIndex {
		t.Errorf(
			"Unexpected validator assignments. want=%v got=%v",
			resp.Duties[0].CommitteeIndex,
			v.duties.Duties[0].CommitteeIndex,
		)
	}
	if v.duties.Duties[0].ValidatorIndex != resp.Duties[0].ValidatorIndex {
		t.Errorf(
			"Unexpected validator assignments. want=%v got=%v",
			resp.Duties[0].ValidatorIndex,
			v.duties.Duties[0].ValidatorIndex,
		)
	}
}

func TestUpdateProtections_OK(t *testing.T) {
	pubKey1 := [48]byte{1}
	pubKey2 := [48]byte{2}
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	client := mock.NewMockBeaconNodeValidatorClient(ctrl)
	db := dbTest.SetupDB(t, [][48]byte{pubKey1, pubKey2})

	newMap := make(map[uint64]uint64)
	newMap[0] = params.BeaconConfig().FarFutureEpoch
	newMap[1] = 0
	newMap[2] = 1
	history := &slashpb.AttestationHistory{
		TargetToSource:     newMap,
		LatestEpochWritten: 2,
	}

	newMap2 := make(map[uint64]uint64)
	newMap2[0] = params.BeaconConfig().FarFutureEpoch
	newMap2[1] = params.BeaconConfig().FarFutureEpoch
	newMap2[2] = params.BeaconConfig().FarFutureEpoch
	newMap2[3] = 2
	history2 := &slashpb.AttestationHistory{
		TargetToSource:     newMap,
		LatestEpochWritten: 3,
	}

	histories := make(map[[48]byte]*slashpb.AttestationHistory)
	histories[pubKey1] = history
	histories[pubKey2] = history2
	if err := db.SaveAttestationHistoryForPubKeys(context.Background(), histories); err != nil {
		t.Fatal(err)
	}

	slot := params.BeaconConfig().SlotsPerEpoch
	duties := &ethpb.DutiesResponse{
		CurrentEpochDuties: []*ethpb.DutiesResponse_Duty{
			{
				AttesterSlot:   slot,
				ValidatorIndex: 200,
				CommitteeIndex: 100,
				Committee:      []uint64{0, 1, 2, 3},
				PublicKey:      pubKey1[:],
			},
			{
				AttesterSlot:   slot,
				ValidatorIndex: 201,
				CommitteeIndex: 100,
				Committee:      []uint64{0, 1, 2, 3},
				PublicKey:      pubKey2[:],
			},
		},
	}
	v := validator{
		db:              db,
		keyManager:      testKeyManager,
		validatorClient: client,
		duties:          duties,
	}

	if err := v.UpdateProtections(context.Background(), slot); err != nil {
		t.Fatalf("Could not update assignments: %v", err)
	}
	if !reflect.DeepEqual(v.attesterHistoryByPubKey[pubKey1], history) {
		t.Fatalf("Expected retrieved history to be equal to %v, received %v", history, v.attesterHistoryByPubKey[pubKey1])
	}
	if !reflect.DeepEqual(v.attesterHistoryByPubKey[pubKey2], history2) {
		t.Fatalf("Expected retrieved history to be equal to %v, received %v", history2, v.attesterHistoryByPubKey[pubKey2])
	}
}

func TestSaveProtections_OK(t *testing.T) {
	pubKey1 := [48]byte{1}
	pubKey2 := [48]byte{2}
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	client := mock.NewMockBeaconNodeValidatorClient(ctrl)
	db := dbTest.SetupDB(t, [][48]byte{pubKey1, pubKey2})

	cleanHistories, err := db.AttestationHistoryForPubKeys(context.Background(), [][48]byte{pubKey1, pubKey2})
	if err != nil {
		t.Fatal(err)
	}
	v := validator{
		db:                      db,
		keyManager:              testKeyManager,
		validatorClient:         client,
		attesterHistoryByPubKey: cleanHistories,
	}

	history1 := cleanHistories[pubKey1]
	history1 = markAttestationForTargetEpoch(history1, 0, 1)

	history2 := cleanHistories[pubKey1]
	history2 = markAttestationForTargetEpoch(history1, 2, 3)

	cleanHistories[pubKey1] = history1
	cleanHistories[pubKey2] = history2

	v.attesterHistoryByPubKey = cleanHistories
	if err := v.SaveProtections(context.Background()); err != nil {
		t.Fatalf("Could not update assignments: %v", err)
	}
	savedHistories, err := db.AttestationHistoryForPubKeys(context.Background(), [][48]byte{pubKey1, pubKey2})
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(savedHistories[pubKey1], history1) {
		t.Fatalf("Expected retrieved history to be equal to %v, received %v", history1, v.attesterHistoryByPubKey[pubKey1])
	}
	if !reflect.DeepEqual(savedHistories[pubKey2], history2) {
		t.Fatalf("Expected retrieved history to be equal to %v, received %v", history2, v.attesterHistoryByPubKey[pubKey2])
	}
}

func TestRolesAt_OK(t *testing.T) {
	v, m, finish := setup(t)
	defer finish()

	sks := make([]bls.SecretKey, 4)
	sks[0] = bls.RandKey()
	sks[1] = bls.RandKey()
	sks[2] = bls.RandKey()
	sks[3] = bls.RandKey()
	v.keyManager = keymanager.NewDirect(sks)
	v.duties = &ethpb.DutiesResponse{
		Duties: []*ethpb.DutiesResponse_Duty{
			{
				CommitteeIndex: 1,
				AttesterSlot:   1,
				PublicKey:      sks[0].PublicKey().Marshal(),
			},
			{
				CommitteeIndex: 2,
				ProposerSlots:  []uint64{1},
				PublicKey:      sks[1].PublicKey().Marshal(),
			},
			{
				CommitteeIndex: 1,
				AttesterSlot:   2,
				PublicKey:      sks[2].PublicKey().Marshal(),
			},
			{
				CommitteeIndex: 2,
				AttesterSlot:   1,
				ProposerSlots:  []uint64{1, 5},
				PublicKey:      sks[3].PublicKey().Marshal(),
			},
		},
	}

	m.validatorClient.EXPECT().DomainData(
		gomock.Any(), // ctx
		gomock.Any(), // epoch
	).Return(&ethpb.DomainResponse{}, nil /*err*/)

	m.validatorClient.EXPECT().DomainData(
		gomock.Any(), // ctx
		gomock.Any(), // epoch
	).Return(&ethpb.DomainResponse{}, nil /*err*/)

	roleMap, err := v.RolesAt(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}

	if roleMap[bytesutil.ToBytes48(sks[0].PublicKey().Marshal())][0] != roleAttester {
		t.Errorf("Unexpected validator role. want: roleProposer")
	}
	if roleMap[bytesutil.ToBytes48(sks[1].PublicKey().Marshal())][0] != roleProposer {
		t.Errorf("Unexpected validator role. want: roleAttester")
	}
	if roleMap[bytesutil.ToBytes48(sks[2].PublicKey().Marshal())][0] != roleUnknown {
		t.Errorf("Unexpected validator role. want: UNKNOWN")
	}
	if roleMap[bytesutil.ToBytes48(sks[3].PublicKey().Marshal())][0] != roleProposer {
		t.Errorf("Unexpected validator role. want: roleProposer")
	}
	if roleMap[bytesutil.ToBytes48(sks[3].PublicKey().Marshal())][1] != roleAttester {
		t.Errorf("Unexpected validator role. want: roleAttester")
	}
	if roleMap[bytesutil.ToBytes48(sks[3].PublicKey().Marshal())][2] != roleAggregator {
		t.Errorf("Unexpected validator role. want: roleAggregator")
	}
}

func TestRolesAt_DoesNotAssignProposer_Slot0(t *testing.T) {
	v, m, finish := setup(t)
	defer finish()

	sks := make([]bls.SecretKey, 3)
	sks[0] = bls.RandKey()
	sks[1] = bls.RandKey()
	sks[2] = bls.RandKey()
	v.keyManager = keymanager.NewDirect(sks)
	v.duties = &ethpb.DutiesResponse{
		Duties: []*ethpb.DutiesResponse_Duty{
			{
				CommitteeIndex: 1,
				AttesterSlot:   0,
				ProposerSlots:  []uint64{0},
				PublicKey:      sks[0].PublicKey().Marshal(),
			},
			{
				CommitteeIndex: 2,
				AttesterSlot:   4,
				ProposerSlots:  nil,
				PublicKey:      sks[1].PublicKey().Marshal(),
			},
			{
				CommitteeIndex: 1,
				AttesterSlot:   3,
				ProposerSlots:  nil,
				PublicKey:      sks[2].PublicKey().Marshal(),
			},
		},
	}

	m.validatorClient.EXPECT().DomainData(
		gomock.Any(), // ctx
		gomock.Any(), // epoch
	).Return(&ethpb.DomainResponse{}, nil /*err*/)

	roleMap, err := v.RolesAt(context.Background(), 0)
	if err != nil {
		t.Fatal(err)
	}

	if roleMap[bytesutil.ToBytes48(sks[0].PublicKey().Marshal())][0] != roleAttester {
		t.Errorf("Unexpected validator role. want: roleProposer")
	}
	if roleMap[bytesutil.ToBytes48(sks[1].PublicKey().Marshal())][0] != roleUnknown {
		t.Errorf("Unexpected validator role. want: roleAttester")
	}
	if roleMap[bytesutil.ToBytes48(sks[2].PublicKey().Marshal())][0] != roleUnknown {
		t.Errorf("Unexpected validator role. want: UNKNOWN")
	}
}

func TestCheckAndLogValidatorStatus_OK(t *testing.T) {
	nonexistentIndex := ^uint64(0)
	type statusTest struct {
		name   string
		status *ethpb.ValidatorActivationResponse_Status
		log    string
		active bool
	}
	pubKeys := [][]byte{
		bytesutil.Uint64ToBytes(0),
		bytesutil.Uint64ToBytes(1),
		bytesutil.Uint64ToBytes(2),
		bytesutil.Uint64ToBytes(3),
	}
	tests := []statusTest{
		{
			name: "UNKNOWN_STATUS, no deposit found yet",
			status: &ethpb.ValidatorActivationResponse_Status{
				PublicKey: pubKeys[0],
				Index:     nonexistentIndex,
				Status: &ethpb.ValidatorStatusResponse{
					Status: ethpb.ValidatorStatus_UNKNOWN_STATUS,
				},
			},
			log: "Waiting for deposit to be observed by beacon node",
		},
		{
			name: "DEPOSITED, deposit found",
			status: &ethpb.ValidatorActivationResponse_Status{
				PublicKey: pubKeys[0],
				Index:     nonexistentIndex,
				Status: &ethpb.ValidatorStatusResponse{
					Status:                 ethpb.ValidatorStatus_DEPOSITED,
					DepositInclusionSlot:   50,
					Eth1DepositBlockNumber: 400,
				},
			},
			log: "Deposit for validator received but not processed into the beacon state\" eth1DepositBlockNumber=400 expectedInclusionSlot=50",
		},
		{
			name: "DEPOSITED into state",
			status: &ethpb.ValidatorActivationResponse_Status{
				PublicKey: pubKeys[0],
				Index:     30,
				Status: &ethpb.ValidatorStatusResponse{
					Status:                    ethpb.ValidatorStatus_DEPOSITED,
					PositionInActivationQueue: 30,
				},
			},
			log: "Deposit processed, entering activation queue after finalization\" index=30 positionInActivationQueue=30",
		},
		{
			name: "PENDING",
			status: &ethpb.ValidatorActivationResponse_Status{
				PublicKey: pubKeys[0],
				Index:     50,
				Status: &ethpb.ValidatorStatusResponse{
					Status:                    ethpb.ValidatorStatus_PENDING,
					ActivationEpoch:           params.BeaconConfig().FarFutureEpoch,
					PositionInActivationQueue: 6,
				},
			},
			log: "Waiting to be assigned activation epoch\" index=50 positionInActivationQueue=6",
		},
		{
			name: "PENDING",
			status: &ethpb.ValidatorActivationResponse_Status{
				PublicKey: pubKeys[0],
				Index:     89,
				Status: &ethpb.ValidatorStatusResponse{
					Status:                    ethpb.ValidatorStatus_PENDING,
					ActivationEpoch:           60,
					PositionInActivationQueue: 5,
				},
			},
			log: "Waiting for activation\" activationEpoch=60 index=89",
		},
		{
			name: "EXITED",
			status: &ethpb.ValidatorActivationResponse_Status{
				PublicKey: pubKeys[0],
				Status: &ethpb.ValidatorStatusResponse{
					Status: ethpb.ValidatorStatus_EXITED,
				},
			},
			log: "Validator exited",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			hook := logTest.NewGlobal()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			client := mock.NewMockBeaconNodeValidatorClient(ctrl)
			v := validator{
				keyManager:      testKeyManager,
				validatorClient: client,
				duties: &ethpb.DutiesResponse{
					Duties: []*ethpb.DutiesResponse_Duty{
						{
							CommitteeIndex: 1,
						},
					},
				},
			}

			active := v.checkAndLogValidatorStatus([]*ethpb.ValidatorActivationResponse_Status{test.status})
			if active != test.active {
				t.Fatalf("expected key to be active, expected %t, received %t", test.active, active)
			}

			testutil.AssertLogsContain(t, hook, test.log)
		})
	}
}
