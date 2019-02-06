package client

import (
	"context"
	"errors"
	"io"
	"io/ioutil"
	"testing"
	"time"

	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/sirupsen/logrus"

	ptypes "github.com/gogo/protobuf/types"

	"github.com/golang/mock/gomock"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/validator/internal"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func init() {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetOutput(ioutil.Discard)
}

var _ = Validator(&validator{})

// TODO(1323): remove this
type fakeAttestationPool struct{}

func (f *fakeAttestationPool) PendingAttestations() []*pbp2p.Attestation {
	return nil
}

var fakePubKey = []byte{1}

func TestWaitForChainStart_SetsChainStartGenesisTime(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	client := internal.NewMockBeaconServiceClient(ctrl)

	v := validator{
		pubKey:       fakePubKey,
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
		pubKey:       fakePubKey,
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
		pubKey:       fakePubKey,
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
		pubKey:       fakePubKey,
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

func TestUpdateAssignments_DoesNothingWhenNotEpochStart(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	client := internal.NewMockValidatorServiceClient(ctrl)

	slot := uint64(1)
	v := validator{
		pubKey:          fakePubKey,
		validatorClient: client,
	}
	client.EXPECT().ValidatorEpochAssignments(
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
		pubKey:          fakePubKey,
		validatorClient: client,
	}

	expected := errors.New("bad")

	client.EXPECT().ValidatorEpochAssignments(
		gomock.Any(),
		gomock.Any(),
	).Return(nil, expected)

	err := v.UpdateAssignments(context.Background(), params.BeaconConfig().EpochLength)
	if err != expected {
		t.Errorf("Bad error; want=%v got=%v", expected, err)
	}
}

func TestUpdateAssignments_DoesUpdateAssignments(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	client := internal.NewMockValidatorServiceClient(ctrl)

	slot := params.BeaconConfig().EpochLength
	resp := &pb.ValidatorEpochAssignmentsResponse{
		Assignment: &pb.Assignment{
			ProposerSlot: 67,
			AttesterSlot: 78,
		},
	}
	v := validator{
		pubKey:          fakePubKey,
		validatorClient: client,
	}
	client.EXPECT().ValidatorEpochAssignments(
		gomock.Any(),
		gomock.Any(),
	).Return(resp, nil)

	if err := v.UpdateAssignments(context.Background(), slot); err != nil {
		t.Fatalf("Could not update assignments: %v", err)
	}

	if v.assignment.ProposerSlot != 67 {
		t.Errorf("Unexpected validator assignments. want=%v got=%v", 67, v.assignment.ProposerSlot)
	}
	if v.assignment.AttesterSlot != 78 {
		t.Errorf("Unexpected validator assignments. want=%v got=%v", 78, v.assignment.AttesterSlot)
	}
}
