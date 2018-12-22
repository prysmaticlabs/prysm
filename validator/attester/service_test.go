package attester

import (
	"context"
	"errors"
	"io/ioutil"
	"testing"

	"github.com/golang/mock/gomock"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	"github.com/prysmaticlabs/prysm/shared/event"
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

func (mc *mockClient) AttesterServiceClient() pb.AttesterServiceClient {
	return internal.NewMockAttesterServiceClient(mc.ctrl)
}

func (mc *mockClient) ValidatorServiceClient() pb.ValidatorServiceClient {
	return internal.NewMockValidatorServiceClient(mc.ctrl)
}

type mockAssigner struct{}

func (m *mockAssigner) AttesterAssignmentFeed() *event.Feed {
	return new(event.Feed)
}

func (m *mockAssigner) PublicKey() []byte {
	return []byte{}
}

func TestLifecycle(t *testing.T) {
	hook := logTest.NewGlobal()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	cfg := &Config{
		AssignmentBuf: 0,
		Assigner:      &mockAssigner{},
		Client:        &mockClient{ctrl},
	}
	att := NewAttester(context.Background(), cfg)
	att.Start()
	att.Stop()

	testutil.AssertLogsContain(t, hook, "Starting service")
	testutil.AssertLogsContain(t, hook, "Stopping service")
}

func TestAttesterLoop(t *testing.T) {
	hook := logTest.NewGlobal()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	cfg := &Config{
		AssignmentBuf: 0,
		Assigner:      &mockAssigner{},
		Client:        &mockClient{ctrl},
	}
	att := NewAttester(context.Background(), cfg)

	mockServiceValidator := internal.NewMockValidatorServiceClient(ctrl)
	mockServiceValidator.EXPECT().ValidatorShardID(
		gomock.Any(),
		gomock.Any(),
	).Return(&pb.ShardIDResponse{
		ShardId: 100,
	}, nil)
	mockServiceValidator.EXPECT().ValidatorIndex(
		gomock.Any(),
		gomock.Any(),
	).Return(&pb.IndexResponse{
		Index: 0,
	}, nil)

	mockServiceAttester := internal.NewMockAttesterServiceClient(ctrl)
	mockServiceAttester.EXPECT().AttestHead(
		gomock.Any(),
		gomock.Any(),
	).Return(&pb.AttestResponse{
		AttestationHash: []byte{'A'},
	}, nil)

	exitRoutine := make(chan bool)
	go func() {
		att.run(mockServiceAttester, mockServiceValidator)
		<-exitRoutine
	}()
	att.assignmentChan <- &pbp2p.BeaconBlock{Slot: 33}
	att.cancel()
	exitRoutine <- true

	testutil.AssertLogsContain(t, hook, "Performing attester responsibility")
	testutil.AssertLogsContain(t, hook, "Attester context closed")
}

func TestAttesterMarshalError(t *testing.T) {
	hook := logTest.NewGlobal()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	ctx := context.Background()
	cfg := &Config{
		AssignmentBuf: 0,
		Assigner:      &mockAssigner{},
		Client:        &mockClient{ctrl},
	}
	att := NewAttester(ctx, cfg)

	mockServiceAttester := internal.NewMockAttesterServiceClient(ctrl)

	mockServiceValidator := internal.NewMockValidatorServiceClient(ctrl)

	exitRoutine := make(chan bool)
	go func() {
		att.run(mockServiceAttester, mockServiceValidator)
		<-exitRoutine
	}()

	att.assignmentChan <- nil
	att.cancel()
	exitRoutine <- true

	testutil.AssertLogsContain(t, hook, "could not marshal nil latest beacon block")
	testutil.AssertLogsContain(t, hook, "Attester context closed")
}

func TestAttesterErrorCantAttestHead(t *testing.T) {
	hook := logTest.NewGlobal()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	cfg := &Config{
		AssignmentBuf: 0,
		Assigner:      &mockAssigner{},
		Client:        &mockClient{ctrl},
	}
	att := NewAttester(context.Background(), cfg)

	mockServiceValidator := internal.NewMockValidatorServiceClient(ctrl)
	mockServiceValidator.EXPECT().ValidatorShardID(
		gomock.Any(),
		gomock.Any(),
	).Return(&pb.ShardIDResponse{
		ShardId: 100,
	}, nil)
	mockServiceValidator.EXPECT().ValidatorIndex(
		gomock.Any(),
		gomock.Any(),
	).Return(&pb.IndexResponse{
		Index: 0,
	}, nil)

	mockServiceAttester := internal.NewMockAttesterServiceClient(ctrl)
	// Expect call to throw an error.
	mockServiceAttester.EXPECT().AttestHead(
		gomock.Any(),
		gomock.Any(),
	).Return(nil, errors.New("could not attest head"))

	exitRoutine := make(chan bool)
	go func() {
		att.run(mockServiceAttester, mockServiceValidator)
		<-exitRoutine
	}()

	att.assignmentChan <- &pbp2p.BeaconBlock{Slot: 999}
	att.cancel()
	exitRoutine <- true

	testutil.AssertLogsContain(t, hook, "Performing attester responsibility")
	testutil.AssertLogsContain(t, hook, "could not attest head")
	testutil.AssertLogsContain(t, hook, "Attester context closed")
}

func TestAttesterErrorCantGetShardID(t *testing.T) {
	hook := logTest.NewGlobal()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	cfg := &Config{
		AssignmentBuf: 0,
		Assigner:      &mockAssigner{},
		Client:        &mockClient{ctrl},
	}
	att := NewAttester(context.Background(), cfg)

	mockServiceValidator := internal.NewMockValidatorServiceClient(ctrl)
	mockServiceValidator.EXPECT().ValidatorShardID(
		gomock.Any(),
		gomock.Any(),
	).Return(nil, errors.New("could not get attester Shard ID"))

	mockServiceAttester := internal.NewMockAttesterServiceClient(ctrl)

	exitRoutine := make(chan bool)
	go func() {
		att.run(mockServiceAttester, mockServiceValidator)
		<-exitRoutine
	}()

	att.assignmentChan <- &pbp2p.BeaconBlock{Slot: 999}
	att.cancel()
	exitRoutine <- true

	testutil.AssertLogsContain(t, hook, "Performing attester responsibility")
	testutil.AssertLogsContain(t, hook, "could not get attester Shard ID")
	testutil.AssertLogsContain(t, hook, "Attester context closed")
}

func TestAttesterErrorCantGetAttesterIndex(t *testing.T) {
	hook := logTest.NewGlobal()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	cfg := &Config{
		AssignmentBuf: 0,
		Assigner:      &mockAssigner{},
		Client:        &mockClient{ctrl},
	}
	att := NewAttester(context.Background(), cfg)

	mockServiceValidator := internal.NewMockValidatorServiceClient(ctrl)
	mockServiceValidator.EXPECT().ValidatorShardID(
		gomock.Any(),
		gomock.Any(),
	).Return(&pb.ShardIDResponse{
		ShardId: 100,
	}, nil)
	mockServiceValidator.EXPECT().ValidatorIndex(
		gomock.Any(),
		gomock.Any(),
	).Return(nil, errors.New("could not get attester index"))

	mockServiceAttester := internal.NewMockAttesterServiceClient(ctrl)

	exitRoutine := make(chan bool)
	go func() {
		att.run(mockServiceAttester, mockServiceValidator)
		<-exitRoutine
	}()

	att.assignmentChan <- &pbp2p.BeaconBlock{Slot: 999}
	att.cancel()
	exitRoutine <- true

	testutil.AssertLogsContain(t, hook, "Performing attester responsibility")
	testutil.AssertLogsContain(t, hook, "could not get attester index")
	testutil.AssertLogsContain(t, hook, "Attester context closed")
}
