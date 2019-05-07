package client

import (
	"context"
	"encoding/hex"
	"errors"
	"io/ioutil"
	"strings"
	"testing"
	"time"

	ptypes "github.com/gogo/protobuf/types"
	"github.com/golang/mock/gomock"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	"github.com/prysmaticlabs/prysm/shared/keystore"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/validator/internal"
	"github.com/sirupsen/logrus"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func init() {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetOutput(ioutil.Discard)
}

var _ = Validator(&validator{})

const cancelledCtx = "context has been canceled"

func publicKeys(keys map[string]*keystore.Key) [][]byte {
	pks := make([][]byte, 0, len(keys))
	for _, value := range keys {
		pks = append(pks, value.PublicKey.Marshal())
	}
	return pks
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
	client := internal.NewMockBeaconServiceClient(ctrl)

	v := validator{
		keys:         keyMap,
		beaconClient: client,
	}
	genesis := uint64(time.Unix(0, 0).Unix())
	clientStream := internal.NewMockBeaconService_WaitForChainStartClient(ctrl)
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
	client := internal.NewMockBeaconServiceClient(ctrl)

	v := validator{
		keys:         keyMap,
		beaconClient: client,
	}
	genesis := uint64(time.Unix(0, 0).Unix())
	clientStream := internal.NewMockBeaconService_WaitForChainStartClient(ctrl)
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
	client := internal.NewMockBeaconServiceClient(ctrl)

	v := validator{
		keys:         keyMap,
		beaconClient: client,
	}
	clientStream := internal.NewMockBeaconService_WaitForChainStartClient(ctrl)
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
	client := internal.NewMockBeaconServiceClient(ctrl)

	v := validator{
		keys:         keyMap,
		beaconClient: client,
	}
	clientStream := internal.NewMockBeaconService_WaitForChainStartClient(ctrl)
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
		keys:            keyMap,
		pubkeys:         make([][]byte, 0),
		validatorClient: client,
	}
	v.pubkeys = publicKeys(v.keys)
	clientStream := internal.NewMockValidatorService_WaitForActivationClient(ctrl)

	client.EXPECT().WaitForActivation(
		gomock.Any(),
		&pb.ValidatorActivationRequest{
			PublicKeys: publicKeys(v.keys),
		},
	).Return(clientStream, nil)
	clientStream.EXPECT().Recv().Return(
		&pb.ValidatorActivationResponse{
			ActivatedPublicKeys: publicKeys(v.keys),
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
		keys:            keyMap,
		pubkeys:         make([][]byte, 0),
		validatorClient: client,
	}
	v.pubkeys = publicKeys(v.keys)
	clientStream := internal.NewMockValidatorService_WaitForActivationClient(ctrl)
	client.EXPECT().WaitForActivation(
		gomock.Any(),
		&pb.ValidatorActivationRequest{
			PublicKeys: publicKeys(v.keys),
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
		keys:            keyMap,
		pubkeys:         make([][]byte, 0),
		validatorClient: client,
	}
	v.pubkeys = publicKeys(v.keys)
	clientStream := internal.NewMockValidatorService_WaitForActivationClient(ctrl)
	client.EXPECT().WaitForActivation(
		gomock.Any(),
		&pb.ValidatorActivationRequest{
			PublicKeys: publicKeys(v.keys),
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
		keys:            keyMap,
		pubkeys:         make([][]byte, 0),
		validatorClient: client,
	}
	v.pubkeys = publicKeys(v.keys)
	resp := generateMockStatusResponse(v.pubkeys)
	resp.Statuses[0].Status.Status = pb.ValidatorStatus_ACTIVE
	clientStream := internal.NewMockValidatorService_WaitForActivationClient(ctrl)
	client.EXPECT().WaitForActivation(
		gomock.Any(),
		&pb.ValidatorActivationRequest{
			PublicKeys: publicKeys(v.keys),
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
	client := internal.NewMockBeaconServiceClient(ctrl)
	v := validator{
		keys:         keyMap,
		beaconClient: client,
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
	client := internal.NewMockBeaconServiceClient(ctrl)
	v := validator{
		keys:         keyMap,
		beaconClient: client,
	}
	client.EXPECT().CanonicalHead(
		gomock.Any(),
		gomock.Any(),
	).Return(&pbp2p.BeaconBlock{Slot: params.BeaconConfig().GenesisSlot}, nil)
	headSlot, err := v.CanonicalHeadSlot(context.Background())
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if headSlot != params.BeaconConfig().GenesisSlot {
		t.Errorf("Mismatch slots, wanted: %v, received: %v", params.BeaconConfig().GenesisSlot, headSlot)
	}
}
func TestWaitMultipleActivation_LogsActivationEpochOK(t *testing.T) {
	hook := logTest.NewGlobal()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	client := internal.NewMockValidatorServiceClient(ctrl)

	v := validator{
		keys:            keyMapThreeValidators,
		pubkeys:         make([][]byte, 0),
		validatorClient: client,
	}
	v.pubkeys = publicKeys(v.keys)
	resp := generateMockStatusResponse(v.pubkeys)
	resp.Statuses[0].Status.Status = pb.ValidatorStatus_ACTIVE
	resp.Statuses[1].Status.Status = pb.ValidatorStatus_ACTIVE
	clientStream := internal.NewMockValidatorService_WaitForActivationClient(ctrl)
	client.EXPECT().WaitForActivation(
		gomock.Any(),
		&pb.ValidatorActivationRequest{
			PublicKeys: v.pubkeys,
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
		keys:            keyMapThreeValidators,
		validatorClient: client,
		pubkeys:         publicKeys(keyMapThreeValidators),
	}
	resp := generateMockStatusResponse(v.pubkeys)
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

func TestUpdateAssignments_DoesNothingWhenNotEpochStartAndAlreadyExistingAssignments(t *testing.T) {
	// TODO(2167): Unskip this test.
	t.Skip()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	client := internal.NewMockValidatorServiceClient(ctrl)

	slot := uint64(1)
	v := validator{
		keys:            keyMap,
		validatorClient: client,
		assignments: &pb.CommitteeAssignmentResponse{
			Assignment: []*pb.CommitteeAssignmentResponse_CommitteeAssignment{
				{
					Committee: []uint64{},
					Slot:      10,
					Shard:     20,
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
		keys:            keyMap,
		validatorClient: client,
		assignments: &pb.CommitteeAssignmentResponse{
			Assignment: []*pb.CommitteeAssignmentResponse_CommitteeAssignment{
				{
					Shard: 1,
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
	resp := &pb.CommitteeAssignmentResponse{
		Assignment: []*pb.CommitteeAssignmentResponse_CommitteeAssignment{
			{
				Slot:       params.BeaconConfig().SlotsPerEpoch,
				Shard:      100,
				Committee:  []uint64{0, 1, 2, 3},
				IsProposer: true,
				PublicKey:  []byte("testPubKey_1"),
			},
		},
	}
	v := validator{
		keys:            keyMap,
		validatorClient: client,
	}
	client.EXPECT().CommitteeAssignment(
		gomock.Any(),
		gomock.Any(),
	).Return(resp, nil)

	if err := v.UpdateAssignments(context.Background(), slot); err != nil {
		t.Fatalf("Could not update assignments: %v", err)
	}

	if v.assignments.Assignment[0].Slot != params.BeaconConfig().SlotsPerEpoch {
		t.Errorf("Unexpected validator assignments. want=%v got=%v", params.BeaconConfig().SlotsPerEpoch, v.assignments.Assignment[0].Slot)
	}
	if v.assignments.Assignment[0].Shard != resp.Assignment[0].Shard {
		t.Errorf("Unexpected validator assignments. want=%v got=%v", resp.Assignment[0].Shard, v.assignments.Assignment[0].Slot)
	}
	if !v.assignments.Assignment[0].IsProposer {
		t.Errorf("Unexpected validator assignments. want: proposer=true")
	}
}

func TestRolesAt_OK(t *testing.T) {

	v := validator{
		assignments: &pb.CommitteeAssignmentResponse{
			Assignment: []*pb.CommitteeAssignmentResponse_CommitteeAssignment{
				{
					Shard:      1,
					Slot:       1,
					IsProposer: true,
					PublicKey:  []byte("pk1"),
				},
				{
					Shard:     2,
					Slot:      1,
					PublicKey: []byte("pk2"),
				},
				{
					Shard:     1,
					Slot:      2,
					PublicKey: []byte("pk3"),
				},
			},
		},
	}
	roleMap := v.RolesAt(1)
	if roleMap[hex.EncodeToString([]byte("pk1"))] != pb.ValidatorRole_PROPOSER {
		t.Errorf("Unexpected validator role. want: ValidatorRole_PROPOSER")
	}
	if roleMap[hex.EncodeToString([]byte("pk2"))] != pb.ValidatorRole_ATTESTER {
		t.Errorf("Unexpected validator role. want: ValidatorRole_ATTESTER")
	}
	if roleMap[hex.EncodeToString([]byte("pk3"))] != pb.ValidatorRole_UNKNOWN {
		t.Errorf("Unexpected validator role. want: UNKNOWN")
	}

}
