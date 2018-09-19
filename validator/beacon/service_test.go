package beacon

import (
	"context"
	"errors"
	"io"
	"io/ioutil"
	"testing"
	"time"

	gomock "github.com/golang/mock/gomock"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/empty"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/validator/internal"
	"github.com/sirupsen/logrus"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func init() {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetOutput(ioutil.Discard)
}

type mockClient struct {
	ctrl *gomock.Controller
}

func (fc *mockClient) BeaconServiceClient() pb.BeaconServiceClient {
	mockServiceClient := internal.NewMockBeaconServiceClient(fc.ctrl)

	stateStream := internal.NewMockBeaconService_LatestCrystallizedStateClient(fc.ctrl)
	stateStream.EXPECT().Recv().Return(&pbp2p.CrystallizedState{}, io.EOF)
	attesterStream := internal.NewMockBeaconService_LatestAttestationClient(fc.ctrl)
	attesterStream.EXPECT().Recv().Return(&pbp2p.AggregatedAttestation{}, io.EOF)

	mockServiceClient.EXPECT().LatestAttestation(
		gomock.Any(),
		&empty.Empty{},
	).Return(attesterStream, nil)

	mockServiceClient.EXPECT().LatestCrystallizedState(
		gomock.Any(),
		&empty.Empty{},
	).Return(stateStream, nil)
	return mockServiceClient
}

type mockLifecycleClient struct {
	ctrl *gomock.Controller
}

func (fc *mockLifecycleClient) BeaconServiceClient() pb.BeaconServiceClient {
	mockServiceClient := internal.NewMockBeaconServiceClient(fc.ctrl)

	stateStream := internal.NewMockBeaconService_LatestCrystallizedStateClient(fc.ctrl)
	stateStream.EXPECT().Recv().Return(&pbp2p.CrystallizedState{}, io.EOF)

	mockServiceClient.EXPECT().LatestCrystallizedState(
		gomock.Any(),
		&empty.Empty{},
	).Return(stateStream, nil)

	validator1 := &pbp2p.ValidatorRecord{WithdrawalAddress: []byte("0x0"), StartDynasty: 1, EndDynasty: 10}
	validator2 := &pbp2p.ValidatorRecord{WithdrawalAddress: []byte("0x1"), StartDynasty: 1, EndDynasty: 10}
	validator3 := &pbp2p.ValidatorRecord{WithdrawalAddress: []byte{}, StartDynasty: 1, EndDynasty: 10}
	crystallized := &pbp2p.CrystallizedState{
		Validators:     []*pbp2p.ValidatorRecord{validator1, validator2, validator3},
		CurrentDynasty: 5,
	}

	mockServiceClient.EXPECT().GenesisTimeAndCanonicalState(
		gomock.Any(),
		gomock.Any(),
	).Return(&pb.GenesisTimeAndStateResponse{
		LatestCrystallizedState: crystallized,
		GenesisTimestamp:        ptypes.TimestampNow(),
	}, nil)

	attesterStream := internal.NewMockBeaconService_LatestAttestationClient(fc.ctrl)
	mockServiceClient.EXPECT().LatestAttestation(
		gomock.Any(),
		&empty.Empty{},
	).Return(attesterStream, nil)
	attesterStream.EXPECT().Recv().Return(&pbp2p.AggregatedAttestation{}, io.EOF)
	return mockServiceClient
}

func TestLifecycle(t *testing.T) {
	hook := logTest.NewGlobal()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	b := NewBeaconValidator(context.Background(), &mockLifecycleClient{ctrl})
	// Testing basic feeds.
	if b.AttesterAssignmentFeed() == nil {
		t.Error("AttesterAssignmentFeed empty")
	}
	if b.ProposerAssignmentFeed() == nil {
		t.Error("ProposerAssignmentFeed empty")
	}
	if b.ProcessedAttestationFeed() == nil {
		t.Error("ProcessedAttestationFeed empty")
	}
	b.Start()
	// TODO: find a better way to test this. The problem is that start is non-blocking, and it ends
	// before the for loops of its inner goroutines begin.
	time.Sleep(time.Millisecond * 10)
	testutil.AssertLogsContain(t, hook, "Starting service")
	b.Stop()
	testutil.AssertLogsContain(t, hook, "Stopping service")
}

func TestWaitForAssignmentProposer(t *testing.T) {
	hook := logTest.NewGlobal()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	b := NewBeaconValidator(context.Background(), &mockClient{ctrl})

	mockServiceClient := internal.NewMockBeaconServiceClient(ctrl)
	mockServiceClient.EXPECT().CanonicalHead(
		gomock.Any(),
		gomock.Any(),
	).Return(nil, nil)

	exitRoutine := make(chan bool)
	timeChan := make(chan time.Time)
	go func() {
		b.waitForAssignment(timeChan, mockServiceClient)
		<-exitRoutine
	}()

	b.responsibility = "proposer"
	b.assignedSlot = 40
	b.currentSlot = 40
	timeChan <- time.Now()
	b.cancel()
	exitRoutine <- true

	testutil.AssertLogsContain(t, hook, "New beacon node slot interval")
}

func TestWaitForAssignmentProposerError(t *testing.T) {
	hook := logTest.NewGlobal()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	b := NewBeaconValidator(context.Background(), &mockClient{ctrl})

	mockServiceClient := internal.NewMockBeaconServiceClient(ctrl)
	mockServiceClient.EXPECT().CanonicalHead(
		gomock.Any(),
		gomock.Any(),
	).Return(nil, errors.New("failed"))

	exitRoutine := make(chan bool)
	timeChan := make(chan time.Time)
	go func() {
		b.waitForAssignment(timeChan, mockServiceClient)
		<-exitRoutine
	}()

	b.responsibility = "proposer"
	b.assignedSlot = 40
	b.currentSlot = 40
	timeChan <- time.Now()
	b.cancel()
	exitRoutine <- true

	testutil.AssertLogsContain(t, hook, "failed")
}

func TestWaitForAssignmentAttester(t *testing.T) {
	hook := logTest.NewGlobal()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	b := NewBeaconValidator(context.Background(), &mockClient{ctrl})

	mockServiceClient := internal.NewMockBeaconServiceClient(ctrl)
	mockServiceClient.EXPECT().CanonicalHead(
		gomock.Any(),
		gomock.Any(),
	).Return(nil, nil)

	exitRoutine := make(chan bool)
	timeChan := make(chan time.Time)
	go func() {
		b.waitForAssignment(timeChan, mockServiceClient)
		<-exitRoutine
	}()

	b.responsibility = "attester"
	b.assignedSlot = 40
	b.currentSlot = 40
	timeChan <- time.Now()
	b.cancel()
	exitRoutine <- true

	testutil.AssertLogsContain(t, hook, "New beacon node slot interval")
}

func TestWaitForAssignmentAttesterError(t *testing.T) {
	hook := logTest.NewGlobal()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	b := NewBeaconValidator(context.Background(), &mockClient{ctrl})

	mockServiceClient := internal.NewMockBeaconServiceClient(ctrl)
	mockServiceClient.EXPECT().CanonicalHead(
		gomock.Any(),
		gomock.Any(),
	).Return(nil, errors.New("failed"))

	exitRoutine := make(chan bool)
	timeChan := make(chan time.Time)
	go func() {
		b.waitForAssignment(timeChan, mockServiceClient)
		<-exitRoutine
	}()

	b.responsibility = "attester"
	b.assignedSlot = 40
	b.currentSlot = 40
	timeChan <- time.Now()
	b.cancel()
	exitRoutine <- true

	testutil.AssertLogsContain(t, hook, "failed")
}

func TestListenForCrystallizedStates(t *testing.T) {
	hook := logTest.NewGlobal()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	b := NewBeaconValidator(context.Background(), &mockClient{ctrl})

	// Creating a faulty stream will trigger error.
	stream := internal.NewMockBeaconService_LatestCrystallizedStateClient(ctrl)
	mockServiceClient := internal.NewMockBeaconServiceClient(ctrl)
	mockServiceClient.EXPECT().LatestCrystallizedState(
		gomock.Any(),
		gomock.Any(),
	).Return(stream, errors.New("stream creation failed"))

	b.listenForCrystallizedStates(mockServiceClient)

	testutil.AssertLogsContain(t, hook, "stream creation failed")

	// Stream recv error should trigger error log.
	stream = internal.NewMockBeaconService_LatestCrystallizedStateClient(ctrl)
	stream.EXPECT().Recv().Return(nil, errors.New("recv error"))
	stream.EXPECT().Recv().Return(&pbp2p.CrystallizedState{}, io.EOF)

	mockServiceClient = internal.NewMockBeaconServiceClient(ctrl)
	mockServiceClient.EXPECT().LatestCrystallizedState(
		gomock.Any(),
		gomock.Any(),
	).Return(stream, nil)

	b.listenForCrystallizedStates(mockServiceClient)

	testutil.AssertLogsContain(t, hook, "recv error")

	// If the current validator is not found within the active validators list, log a debug message.
	validator := &pbp2p.ValidatorRecord{WithdrawalAddress: []byte("0x01"), StartDynasty: 1, EndDynasty: 10}
	stream = internal.NewMockBeaconService_LatestCrystallizedStateClient(ctrl)
	stream.EXPECT().Recv().Return(&pbp2p.CrystallizedState{Validators: []*pbp2p.ValidatorRecord{validator}, CurrentDynasty: 5}, nil)
	stream.EXPECT().Recv().Return(&pbp2p.CrystallizedState{}, io.EOF)

	mockServiceClient = internal.NewMockBeaconServiceClient(ctrl)
	mockServiceClient.EXPECT().LatestCrystallizedState(
		gomock.Any(),
		gomock.Any(),
	).Return(stream, nil)

	b.listenForCrystallizedStates(mockServiceClient)

	testutil.AssertLogsContain(t, hook, "Validator index not found in latest crystallized state's active validator list")

	// If the validator is the last index in the shuffled validator indices, it should be assigned
	// to be a proposer.
	validator1 := &pbp2p.ValidatorRecord{WithdrawalAddress: []byte("0x0"), StartDynasty: 1, EndDynasty: 10}
	validator2 := &pbp2p.ValidatorRecord{WithdrawalAddress: []byte("0x1"), StartDynasty: 1, EndDynasty: 10}
	validator3 := &pbp2p.ValidatorRecord{WithdrawalAddress: []byte{}, StartDynasty: 1, EndDynasty: 10}
	stream = internal.NewMockBeaconService_LatestCrystallizedStateClient(ctrl)
	stream.EXPECT().Recv().Return(&pbp2p.CrystallizedState{Validators: []*pbp2p.ValidatorRecord{validator1, validator2, validator3}, CurrentDynasty: 5}, nil)
	stream.EXPECT().Recv().Return(&pbp2p.CrystallizedState{}, io.EOF)

	mockServiceClient = internal.NewMockBeaconServiceClient(ctrl)
	mockServiceClient.EXPECT().LatestCrystallizedState(
		gomock.Any(),
		gomock.Any(),
	).Return(stream, nil)

	b.listenForCrystallizedStates(mockServiceClient)

	testutil.AssertLogsContain(t, hook, "Validator selected as proposer")
}

func TestListenForProcessedAttestations(t *testing.T) {
	hook := logTest.NewGlobal()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	b := NewBeaconValidator(context.Background(), &mockClient{ctrl})

	// Create mock for the stream returned by LatestAttestation.
	stream := internal.NewMockBeaconService_LatestAttestationClient(ctrl)

	// Testing if an attestation is received,triggering a log.
	stream.EXPECT().Recv().Return(&pbp2p.AggregatedAttestation{Slot: 10}, nil)
	stream.EXPECT().Recv().Return(&pbp2p.AggregatedAttestation{}, io.EOF)

	mockServiceClient := internal.NewMockBeaconServiceClient(ctrl)
	mockServiceClient.EXPECT().LatestAttestation(
		gomock.Any(),
		gomock.Any(),
	).Return(stream, nil)

	b.listenForProcessedAttestations(mockServiceClient)

	testutil.AssertLogsContain(t, hook, "Latest attestation slot number")

	// Testing an error coming from the stream.
	stream = internal.NewMockBeaconService_LatestAttestationClient(ctrl)
	stream.EXPECT().Recv().Return(&pbp2p.AggregatedAttestation{}, errors.New("stream error"))
	stream.EXPECT().Recv().Return(&pbp2p.AggregatedAttestation{}, io.EOF)

	mockServiceClient = internal.NewMockBeaconServiceClient(ctrl)
	mockServiceClient.EXPECT().LatestAttestation(
		gomock.Any(),
		gomock.Any(),
	).Return(stream, nil)

	b.listenForProcessedAttestations(mockServiceClient)

	testutil.AssertLogsContain(t, hook, "stream error")

	// Creating a faulty stream will trigger error.
	mockServiceClient = internal.NewMockBeaconServiceClient(ctrl)
	mockServiceClient.EXPECT().LatestAttestation(
		gomock.Any(),
		gomock.Any(),
	).Return(stream, errors.New("stream creation failed"))

	b.listenForProcessedAttestations(mockServiceClient)
	testutil.AssertLogsContain(t, hook, "stream creation failed")
	testutil.AssertLogsContain(t, hook, "Could not receive latest attestation from stream")
}
