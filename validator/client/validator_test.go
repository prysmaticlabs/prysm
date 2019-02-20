package client

import (
	"context"
	"errors"
	"io"
	"io/ioutil"
	"testing"
	"time"

	ptypes "github.com/gogo/protobuf/types"
	"github.com/golang/mock/gomock"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
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

func TestWaitForChainStart_SetsChainStartGenesisTime(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	client := internal.NewMockBeaconServiceClient(ctrl)

	v := validator{
		key:          validatorKey,
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
	v.WaitForChainStart(context.Background())
	if v.genesisTime != genesis {
		t.Errorf("Expected chain start time to equal %d, received %d", genesis, v.genesisTime)
	}
	if v.ticker == nil {
		t.Error("Expected ticker to be set, received nil")
	}
}

func TestWaitForChainStart_ContextCanceled(t *testing.T) {
	hook := logTest.NewGlobal()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	client := internal.NewMockBeaconServiceClient(ctrl)

	v := validator{
		key:          validatorKey,
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
	v.WaitForChainStart(ctx)
	testutil.AssertLogsContain(t, hook, "Context has been canceled")
}

func TestWaitForChainStart_StreamSetupFails(t *testing.T) {
	hook := logTest.NewGlobal()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	client := internal.NewMockBeaconServiceClient(ctrl)

	v := validator{
		key:          validatorKey,
		beaconClient: client,
	}
	clientStream := internal.NewMockBeaconService_WaitForChainStartClient(ctrl)
	client.EXPECT().WaitForChainStart(
		gomock.Any(),
		&ptypes.Empty{},
	).Return(clientStream, errors.New("failed stream"))
	v.WaitForChainStart(context.Background())
	testutil.AssertLogsContain(t, hook, "Could not setup beacon chain ChainStart streaming client")
}

func TestWaitForChainStart_ReceiveErrorFromStream(t *testing.T) {
	hook := logTest.NewGlobal()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	client := internal.NewMockBeaconServiceClient(ctrl)

	v := validator{
		key:          validatorKey,
		beaconClient: client,
	}
	genesis := uint64(time.Unix(0, 0).Unix())
	clientStream := internal.NewMockBeaconService_WaitForChainStartClient(ctrl)
	client.EXPECT().WaitForChainStart(
		gomock.Any(),
		&ptypes.Empty{},
	).Return(clientStream, nil)
	clientStream.EXPECT().Recv().Return(
		nil,
		errors.New("fails"),
	)
	clientStream.EXPECT().Recv().Return(
		&pb.ChainStartResponse{
			Started:     true,
			GenesisTime: genesis,
		},
		io.EOF,
	)
	v.WaitForChainStart(context.Background())
	testutil.AssertLogsContain(t, hook, "Could not receive ChainStart from stream")
}

func TestUpdateAssignments_DoesNothingWhenNotEpochStartAndAlreadyExistingAssignments(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	client := internal.NewMockValidatorServiceClient(ctrl)

	slot := uint64(1)
	v := validator{
		key:             validatorKey,
		validatorClient: client,
		assignment: &pb.CommitteeAssignmentResponse{
			Committee: []uint64{0, 1, 2, 3},
			Slot:      20,
		},
	}
	client.EXPECT().NextEpochCommitteeAssignment(
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
		key:             validatorKey,
		validatorClient: client,
	}

	expected := errors.New("bad")

	client.EXPECT().NextEpochCommitteeAssignment(
		gomock.Any(),
		gomock.Any(),
	).Return(nil, expected)

	if err := v.UpdateAssignments(context.Background(), params.BeaconConfig().SlotsPerEpoch); err != expected {
		t.Errorf("Bad error; want=%v got=%v", expected, err)
	}
}

func TestUpdateAssignments_DoesUpdateAssignments(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	client := internal.NewMockValidatorServiceClient(ctrl)

	slot := params.BeaconConfig().SlotsPerEpoch
	resp := &pb.CommitteeAssignmentResponse{
		Slot:       params.BeaconConfig().SlotsPerEpoch,
		Shard:      100,
		Committee:  []uint64{0, 1, 2, 3},
		IsProposer: true,
	}
	v := validator{
		key:             validatorKey,
		validatorClient: client,
	}
	client.EXPECT().NextEpochCommitteeAssignment(
		gomock.Any(),
		gomock.Any(),
	).Return(resp, nil)

	if err := v.UpdateAssignments(context.Background(), slot); err != nil {
		t.Fatalf("Could not update assignments: %v", err)
	}

	if v.assignment.Slot != params.BeaconConfig().SlotsPerEpoch {
		t.Errorf("Unexpected validator assignments. want=%v got=%v", params.BeaconConfig().SlotsPerEpoch, v.assignment.Slot)
	}
	if v.assignment.Shard != resp.Shard {
		t.Errorf("Unexpected validator assignments. want=%v got=%v", resp.Shard, v.assignment.Slot)
	}
	if !v.assignment.IsProposer {
		t.Errorf("Unexpected validator assignments. want: proposer=true")
	}
}
