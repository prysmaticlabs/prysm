package beacon

import (
	"bytes"
	"context"
	"errors"
	"io"
	"io/ioutil"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
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

	mockServiceClient.EXPECT().CurrentAssignmentsAndGenesisTime(
		gomock.Any(),
		gomock.Any(),
	).Return(&pb.CurrentAssignmentsResponse{
		GenesisTimestamp: ptypes.TimestampNow(),
	}, nil)

	attesterStream := internal.NewMockBeaconService_LatestAttestationClient(fc.ctrl)
	mockServiceClient.EXPECT().LatestAttestation(
		gomock.Any(),
		&empty.Empty{},
	).Return(attesterStream, nil)
	attesterStream.EXPECT().Recv().Return(&pbp2p.AggregatedAttestation{}, io.EOF)

	cycleStream := internal.NewMockBeaconService_ValidatorAssignmentsClient(fc.ctrl)
	mockServiceClient.EXPECT().ValidatorAssignments(
		gomock.Any(),
		gomock.Any(),
	).Return(cycleStream, nil)
	cycleStream.EXPECT().Recv().Return(&pb.ValidatorAssignmentResponse{}, io.EOF)

	return mockServiceClient
}

func TestLifecycle(t *testing.T) {
	hook := logTest.NewGlobal()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	b := NewBeaconValidator(context.Background(), []byte{}, &mockLifecycleClient{ctrl})
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
	if !bytes.Equal(b.PublicKey(), []byte{}) {
		t.Error("Incorrect public key")
	}
	b.slotAlignmentDuration = time.Millisecond * 10
	b.Start()
	time.Sleep(time.Millisecond * 10)
	testutil.AssertLogsContain(t, hook, "Starting service")
	b.Stop()
	testutil.AssertLogsContain(t, hook, "Stopping service")
}

func TestCurrentBeaconSlot(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	b := NewBeaconValidator(context.Background(), []byte{}, &mockLifecycleClient{ctrl})
	b.genesisTimestamp = time.Now()
	if b.CurrentBeaconSlot() != 0 {
		t.Errorf("Expected us to be in the 0th slot, received %v", b.CurrentBeaconSlot())
	}
}

func TestWaitForAssignmentProposer(t *testing.T) {
	hook := logTest.NewGlobal()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	b := NewBeaconValidator(context.Background(), []byte{}, &mockClient{ctrl})

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

	b.role = pb.ValidatorRole_PROPOSER
	b.genesisTimestamp = time.Now()
	b.assignedSlot = 0
	timeChan <- time.Now()
	b.cancel()
	exitRoutine <- true

	testutil.AssertLogsContain(t, hook, "Assigned proposal slot number reached")
}

func TestWaitForAssignmentProposerError(t *testing.T) {
	hook := logTest.NewGlobal()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	b := NewBeaconValidator(context.Background(), []byte{}, &mockClient{ctrl})

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

	b.role = pb.ValidatorRole_PROPOSER
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
	b := NewBeaconValidator(context.Background(), []byte{}, &mockClient{ctrl})

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

	b.role = pb.ValidatorRole_ATTESTER
	b.genesisTimestamp = time.Now()
	b.assignedSlot = 0
	timeChan <- time.Now()
	b.cancel()
	exitRoutine <- true

	testutil.AssertLogsContain(t, hook, "Assigned attest slot number reached")
}

func TestWaitForAssignmentAttesterError(t *testing.T) {
	hook := logTest.NewGlobal()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	b := NewBeaconValidator(context.Background(), []byte{}, &mockClient{ctrl})

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

	b.role = pb.ValidatorRole_ATTESTER
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
	b := NewBeaconValidator(context.Background(), []byte{}, &mockClient{ctrl})

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

	// Creating a faulty stream will trigger error.
	mockServiceClient = internal.NewMockBeaconServiceClient(ctrl)
	mockServiceClient.EXPECT().LatestAttestation(
		gomock.Any(),
		gomock.Any(),
	).Return(stream, errors.New("stream creation failed"))

	b.listenForProcessedAttestations(mockServiceClient)
	testutil.AssertLogsContain(t, hook, "stream creation failed")

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

func TestListenForAssignmentProposer(t *testing.T) {
	hook := logTest.NewGlobal()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	b := NewBeaconValidator(context.Background(), []byte{'A'}, &mockClient{ctrl})

	// Create mock for the stream returned by LatestAttestation.
	stream := internal.NewMockBeaconService_ValidatorAssignmentsClient(ctrl)

	// Testing proposer assignment.
	assignedSlot := b.CurrentCycleStartSlot() + 2
	stream.EXPECT().Recv().Return(&pb.ValidatorAssignmentResponse{Assignments: []*pb.Assignment{{
		PublicKey:    &pb.PublicKey{PublicKey: []byte{'A'}},
		ShardId:      2,
		AssignedSlot: assignedSlot,
		Role:         pb.ValidatorRole_PROPOSER}}}, nil)
	stream.EXPECT().Recv().Return(&pb.ValidatorAssignmentResponse{}, io.EOF)

	mockServiceValidator := internal.NewMockBeaconServiceClient(ctrl)
	mockServiceValidator.EXPECT().ValidatorAssignments(
		gomock.Any(),
		gomock.Any(),
	).Return(stream, nil)

	b.listenForAssignmentChange(mockServiceValidator)

	testutil.AssertLogsContain(t, hook, "Validator with pub key 0xA re-assigned to shard ID 2 for PROPOSER duty")

	// Creating a faulty stream will trigger error.
	mockServiceValidator = internal.NewMockBeaconServiceClient(ctrl)
	mockServiceValidator.EXPECT().ValidatorAssignments(
		gomock.Any(),
		gomock.Any(),
	).Return(stream, errors.New("stream creation failed"))

	b.listenForAssignmentChange(mockServiceValidator)
	testutil.AssertLogsContain(t, hook, "stream creation failed")
	testutil.AssertLogsContain(t, hook, "could not fetch validator assigned slot and responsibility from beacon node")

	// Test that the routine exits when context is closed
	stream = internal.NewMockBeaconService_ValidatorAssignmentsClient(ctrl)
	stream.EXPECT().Recv().Return(&pb.ValidatorAssignmentResponse{}, nil)

	//mockServiceClient = internal.NewMockBeaconServiceClient(ctrl)
	mockServiceValidator = internal.NewMockBeaconServiceClient(ctrl)
	mockServiceValidator.EXPECT().ValidatorAssignments(
		gomock.Any(),
		gomock.Any(),
	).Return(stream, nil)
	b.cancel()

	b.listenForAssignmentChange(mockServiceValidator)
	testutil.AssertLogsContain(t, hook, "Context has been canceled so shutting down the loop")
}
