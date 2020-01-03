package client

import (
	"context"
	"errors"
	"io/ioutil"
	"strings"
	"testing"
	"time"

	ptypes "github.com/gogo/protobuf/types"
	"github.com/golang/mock/gomock"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/validator/internal"
	"github.com/prysmaticlabs/prysm/validator/keymanager"
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
	keys, _ := km.FetchValidatingKeys()
	res := make([][]byte, len(keys))
	for i := range keys {
		res[i] = keys[i][:]
	}
	return res
}

func generateMockStatusResponse(pubkeys [][]byte) *pb.ValidatorActivationResponse {
	multipleStatus := make([]*pb.ValidatorActivationResponse_Status, len(pubkeys))
	for i, key := range pubkeys {
		multipleStatus[i] = &pb.ValidatorActivationResponse_Status{
			PublicKey: key,
			Status: &pb.ValidatorStatusResponse{
				Status: pb.ValidatorStatus_UNKNOWN_STATUS,
			},
		}
	}
	return &pb.ValidatorActivationResponse{Statuses: multipleStatus}
}

func TestWaitForChainStart_SetsChainStartGenesisTime(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	client := internal.NewMockValidatorServiceClient(ctrl)

	v := validator{
		keyManager:      testKeyManager,
		validatorClient: client,
	}
	genesis := uint64(time.Unix(1, 0).Unix())
	clientStream := internal.NewMockValidatorService_WaitForChainStartClient(ctrl)
	client.EXPECT().WaitForChainStart(
		gomock.Any(),
		&ptypes.Empty{},
	).Return(clientStream, nil)
	clientStream.EXPECT().Recv().Return(
		&pb.ChainStartResponse{
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
	client := internal.NewMockValidatorServiceClient(ctrl)

	v := validator{
		keyManager:      testKeyManager,
		validatorClient: client,
	}
	genesis := uint64(time.Unix(0, 0).Unix())
	clientStream := internal.NewMockValidatorService_WaitForChainStartClient(ctrl)
	client.EXPECT().WaitForChainStart(
		gomock.Any(),
		&ptypes.Empty{},
	).Return(clientStream, nil)
	clientStream.EXPECT().Recv().Return(
		&pb.ChainStartResponse{
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
	client := internal.NewMockValidatorServiceClient(ctrl)

	v := validator{
		keyManager:      testKeyManager,
		validatorClient: client,
	}
	clientStream := internal.NewMockValidatorService_WaitForChainStartClient(ctrl)
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
	client := internal.NewMockValidatorServiceClient(ctrl)

	v := validator{
		keyManager:      testKeyManager,
		validatorClient: client,
	}
	clientStream := internal.NewMockValidatorService_WaitForChainStartClient(ctrl)
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

func TestWaitActivation_ContextCanceled(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	client := internal.NewMockValidatorServiceClient(ctrl)

	v := validator{
		keyManager:      testKeyManager,
		validatorClient: client,
	}
	clientStream := internal.NewMockValidatorService_WaitForActivationClient(ctrl)

	client.EXPECT().WaitForActivation(
		gomock.Any(),
		&pb.ValidatorActivationRequest{
			PublicKeys: publicKeys(v.keyManager),
		},
	).Return(clientStream, nil)
	clientStream.EXPECT().Recv().Return(
		&pb.ValidatorActivationResponse{
			ActivatedPublicKeys: publicKeys(v.keyManager),
		},
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
	client := internal.NewMockValidatorServiceClient(ctrl)

	v := validator{
		keyManager:      testKeyManager,
		validatorClient: client,
	}
	clientStream := internal.NewMockValidatorService_WaitForActivationClient(ctrl)
	client.EXPECT().WaitForActivation(
		gomock.Any(),
		&pb.ValidatorActivationRequest{
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
	client := internal.NewMockValidatorServiceClient(ctrl)

	v := validator{
		keyManager:      testKeyManager,
		validatorClient: client,
	}
	clientStream := internal.NewMockValidatorService_WaitForActivationClient(ctrl)
	client.EXPECT().WaitForActivation(
		gomock.Any(),
		&pb.ValidatorActivationRequest{
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
	client := internal.NewMockValidatorServiceClient(ctrl)

	v := validator{
		keyManager:      testKeyManager,
		validatorClient: client,
		genesisTime:     1,
	}
	resp := generateMockStatusResponse(publicKeys(v.keyManager))
	resp.Statuses[0].Status.Status = pb.ValidatorStatus_ACTIVE
	clientStream := internal.NewMockValidatorService_WaitForActivationClient(ctrl)
	client.EXPECT().WaitForActivation(
		gomock.Any(),
		&pb.ValidatorActivationRequest{
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
	client := internal.NewMockValidatorServiceClient(ctrl)
	v := validator{
		keyManager:      testKeyManager,
		validatorClient: client,
		genesisTime:     1,
	}
	client.EXPECT().CanonicalHead(
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
	client := internal.NewMockValidatorServiceClient(ctrl)
	v := validator{
		keyManager:      testKeyManager,
		validatorClient: client,
	}
	client.EXPECT().CanonicalHead(
		gomock.Any(),
		gomock.Any(),
	).Return(&ethpb.BeaconBlock{Slot: 0}, nil)
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
	client := internal.NewMockValidatorServiceClient(ctrl)

	v := validator{
		keyManager:      testKeyManagerThreeValidators,
		validatorClient: client,
		genesisTime:     1,
	}
	resp := generateMockStatusResponse(publicKeys(v.keyManager))
	resp.Statuses[0].Status.Status = pb.ValidatorStatus_ACTIVE
	resp.Statuses[1].Status.Status = pb.ValidatorStatus_ACTIVE
	clientStream := internal.NewMockValidatorService_WaitForActivationClient(ctrl)
	client.EXPECT().WaitForActivation(
		gomock.Any(),
		&pb.ValidatorActivationRequest{
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
func TestWaitActivation_NotAllValidatorsActivatedOK(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	client := internal.NewMockValidatorServiceClient(ctrl)

	v := validator{
		keyManager:      testKeyManagerThreeValidators,
		validatorClient: client,
		pubkeys:         publicKeys(keyMapThreeValidators),
		genesisTime:     1,
	}
	resp := generateMockStatusResponse(publicKeys(v.keyManager))
	resp.Statuses[0].Status.Status = pb.ValidatorStatus_ACTIVE
	clientStream := internal.NewMockValidatorService_WaitForActivationClient(ctrl)
	client.EXPECT().WaitForActivation(
		gomock.Any(),
		gomock.Any(),
	).Return(clientStream, nil)
	clientStream.EXPECT().Recv().Return(
		&pb.ValidatorActivationResponse{
			ActivatedPublicKeys: make([][]byte, 0),
		},
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
	n := internal.NewMockNodeClient(ctrl)

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
	n := internal.NewMockNodeClient(ctrl)

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
	n := internal.NewMockNodeClient(ctrl)

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

func TestUpdateAssignments_DoesNothingWhenNotEpochStartAndAlreadyExistingAssignments(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	client := internal.NewMockValidatorServiceClient(ctrl)

	slot := uint64(1)
	v := validator{
		keyManager:      testKeyManager,
		validatorClient: client,
		assignments: &pb.AssignmentResponse{
			ValidatorAssignment: []*pb.AssignmentResponse_ValidatorAssignment{
				{
					Committee:      []uint64{},
					AttesterSlot:   10,
					CommitteeIndex: 20,
				},
			},
		},
	}
	client.EXPECT().CommitteeAssignment(
		gomock.Any(),
		gomock.Any(),
	).Times(0)

	if err := v.UpdateAssignments(context.Background(), slot); err != nil {
		t.Errorf("Could not update assignments: %v", err)
	}
}

func TestUpdateAssignments_ReturnsError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	client := internal.NewMockValidatorServiceClient(ctrl)

	v := validator{
		keyManager:      testKeyManager,
		validatorClient: client,
		assignments: &pb.AssignmentResponse{
			ValidatorAssignment: []*pb.AssignmentResponse_ValidatorAssignment{
				{
					CommitteeIndex: 1,
				},
			},
		},
	}

	expected := errors.New("bad")

	client.EXPECT().CommitteeAssignment(
		gomock.Any(),
		gomock.Any(),
	).Return(nil, expected)

	if err := v.UpdateAssignments(context.Background(), params.BeaconConfig().SlotsPerEpoch); err != expected {
		t.Errorf("Bad error; want=%v got=%v", expected, err)
	}
	if v.assignments != nil {
		t.Error("Assignments should have been cleared on failure")
	}
}

func TestUpdateAssignments_OK(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	client := internal.NewMockValidatorServiceClient(ctrl)

	slot := params.BeaconConfig().SlotsPerEpoch
	resp := &pb.AssignmentResponse{
		ValidatorAssignment: []*pb.AssignmentResponse_ValidatorAssignment{
			{
				AttesterSlot:   params.BeaconConfig().SlotsPerEpoch,
				CommitteeIndex: 100,
				Committee:      []uint64{0, 1, 2, 3},
				PublicKey:      []byte("testPubKey_1"),
				ProposerSlot:   params.BeaconConfig().SlotsPerEpoch + 1,
			},
		},
	}
	v := validator{
		keyManager:      testKeyManager,
		validatorClient: client,
	}
	client.EXPECT().CommitteeAssignment(
		gomock.Any(),
		gomock.Any(),
	).Return(resp, nil)

	indexResp := &pb.ValidatorIndexResponse{}
	client.EXPECT().ValidatorIndex(
		gomock.Any(),
		gomock.Any(),
	).Return(indexResp, nil)

	if err := v.UpdateAssignments(context.Background(), slot); err != nil {
		t.Fatalf("Could not update assignments: %v", err)
	}
	if v.assignments.ValidatorAssignment[0].ProposerSlot != params.BeaconConfig().SlotsPerEpoch+1 {
		t.Errorf("Unexpected validator assignments. want=%v got=%v", params.BeaconConfig().SlotsPerEpoch+1, v.assignments.ValidatorAssignment[0].ProposerSlot)
	}
	if v.assignments.ValidatorAssignment[0].AttesterSlot != params.BeaconConfig().SlotsPerEpoch {
		t.Errorf("Unexpected validator assignments. want=%v got=%v", params.BeaconConfig().SlotsPerEpoch, v.assignments.ValidatorAssignment[0].AttesterSlot)
	}
	if v.assignments.ValidatorAssignment[0].CommitteeIndex != resp.ValidatorAssignment[0].CommitteeIndex {
		t.Errorf("Unexpected validator assignments. want=%v got=%v", resp.ValidatorAssignment[0].CommitteeIndex, v.assignments.ValidatorAssignment[0].CommitteeIndex)
	}
}

func TestRolesAt_OK(t *testing.T) {
	v, m, finish := setup(t)
	defer finish()

	sks := make([]*bls.SecretKey, 4)
	sks[0] = bls.RandKey()
	sks[1] = bls.RandKey()
	sks[2] = bls.RandKey()
	sks[3] = bls.RandKey()
	v.keyManager = keymanager.NewDirect(sks)
	v.assignments = &pb.AssignmentResponse{
		ValidatorAssignment: []*pb.AssignmentResponse_ValidatorAssignment{
			{
				CommitteeIndex: 1,
				AttesterSlot:   1,
				PublicKey:      sks[0].PublicKey().Marshal(),
			},
			{
				CommitteeIndex: 2,
				ProposerSlot:   1,
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
				ProposerSlot:   1,
				PublicKey:      sks[3].PublicKey().Marshal(),
			},
		},
	}

	m.validatorClient.EXPECT().DomainData(
		gomock.Any(), // ctx
		gomock.Any(), // epoch
	).Return(&pb.DomainResponse{}, nil /*err*/)
	m.validatorClient.EXPECT().DomainData(
		gomock.Any(), // ctx
		gomock.Any(), // epoch
	).Return(&pb.DomainResponse{}, nil /*err*/)

	roleMap, err := v.RolesAt(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}

	if roleMap[bytesutil.ToBytes48(sks[0].PublicKey().Marshal())][0] != pb.ValidatorRole_ATTESTER {
		t.Errorf("Unexpected validator role. want: ValidatorRole_PROPOSER")
	}
	if roleMap[bytesutil.ToBytes48(sks[1].PublicKey().Marshal())][0] != pb.ValidatorRole_PROPOSER {
		t.Errorf("Unexpected validator role. want: ValidatorRole_ATTESTER")
	}
	if roleMap[bytesutil.ToBytes48(sks[2].PublicKey().Marshal())][0] != pb.ValidatorRole_UNKNOWN {
		t.Errorf("Unexpected validator role. want: UNKNOWN")
	}
	if roleMap[bytesutil.ToBytes48(sks[3].PublicKey().Marshal())][0] != pb.ValidatorRole_PROPOSER {
		t.Errorf("Unexpected validator role. want: ValidatorRole_PROPOSER")
	}
	if roleMap[bytesutil.ToBytes48(sks[3].PublicKey().Marshal())][1] != pb.ValidatorRole_ATTESTER {
		t.Errorf("Unexpected validator role. want: ValidatorRole_ATTESTER")
	}
	if roleMap[bytesutil.ToBytes48(sks[3].PublicKey().Marshal())][2] != pb.ValidatorRole_AGGREGATOR {
		t.Errorf("Unexpected validator role. want: ValidatorRole_AGGREGATOR")
	}
}
