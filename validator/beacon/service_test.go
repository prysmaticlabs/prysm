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

	attesterStream := internal.NewMockBeaconService_LatestAttestationClient(fc.ctrl)
	attesterStream.EXPECT().Recv().Return(&pbp2p.AggregatedAttestation{}, io.EOF)

	mockServiceClient.EXPECT().LatestAttestation(
		gomock.Any(),
		&empty.Empty{},
	).Return(attesterStream, nil)

	return mockServiceClient
}

type mockLifecycleClient struct {
	ctrl *gomock.Controller
}

func (fc *mockLifecycleClient) BeaconServiceClient() pb.BeaconServiceClient {
	mockServiceClient := internal.NewMockBeaconServiceClient(fc.ctrl)

	mockServiceClient.EXPECT().GenesisTimeAndCanonicalState(
		gomock.Any(),
		gomock.Any(),
	).Return(&pb.GenesisTimeAndStateResponse{
		GenesisTimestamp: ptypes.TimestampNow(),
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
	time.Sleep(time.Millisecond * 10)
	testutil.AssertLogsContain(t, hook, "Starting service")
	b.Stop()
	testutil.AssertLogsContain(t, hook, "Stopping service")
}

func TestCurrentBeaconSlot(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	b := NewBeaconValidator(context.Background(), &mockLifecycleClient{ctrl})
	b.genesisTimestamp = time.Now()
	if b.CurrentBeaconSlot() != 0 {
		t.Errorf("Expected us to be in the 0th slot, received %v", b.CurrentBeaconSlot())
	}
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
	b.genesisTimestamp = time.Now()
	b.assignedSlot = 0
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
	b.genesisTimestamp = time.Now()
	b.assignedSlot = 0
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
	b.genesisTimestamp = time.Now()
	b.assignedSlot = 0
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
	b.genesisTimestamp = time.Now()
	b.assignedSlot = 0
	timeChan <- time.Now()
	b.cancel()
	exitRoutine <- true

	testutil.AssertLogsContain(t, hook, "failed")
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

	// Test that the routine exits when context is closed
	stream = internal.NewMockBeaconService_LatestAttestationClient(ctrl)

	stream.EXPECT().Recv().Return(&pbp2p.AggregatedAttestation{}, nil)

	mockServiceClient = internal.NewMockBeaconServiceClient(ctrl)
	mockServiceClient.EXPECT().LatestAttestation(
		gomock.Any(),
		gomock.Any(),
	).Return(stream, nil)
	b.cancel()

	b.listenForProcessedAttestations(mockServiceClient)
	testutil.AssertLogsContain(t, hook, "Context has been canceled so shutting down the loop")
}
