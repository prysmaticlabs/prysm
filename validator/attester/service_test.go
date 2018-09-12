package attester

import (
	"context"
	"errors"
	"io/ioutil"
	"testing"

	"github.com/ethereum/go-ethereum/event"
	"github.com/golang/mock/gomock"
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

func (mc *mockClient) AttesterServiceClient() pb.AttesterServiceClient {
	return internal.NewMockAttesterServiceClient(mc.ctrl)
}

type mockAssigner struct{}

func (m *mockAssigner) AttesterAssignmentFeed() *event.Feed {
	return new(event.Feed)
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
	testutil.AssertLogsContain(t, hook, "Starting service")
	att.Stop()
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

	mockServiceClient := internal.NewMockAttesterServiceClient(ctrl)
	mockServiceClient.EXPECT().AttestHead(
		gomock.Any(),
		gomock.Any(),
	).Return(&pb.AttestResponse{
		AttestationHash: []byte{'A'},
	}, nil)

	doneChan := make(chan struct{})
	exitRoutine := make(chan bool)
	go func() {
		att.run(doneChan, mockServiceClient)
		<-exitRoutine
	}()

	att.assignmentChan <- &pbp2p.BeaconBlock{SlotNumber: 999}

	testutil.AssertLogsContain(t, hook, "Performing attester responsibility")
	doneChan <- struct{}{}
	exitRoutine <- true
	testutil.AssertLogsContain(t, hook, "Attester context closed")
}

func TestAttesterMarshalError(t *testing.T) {
	hook := logTest.NewGlobal()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	cfg := &Config{
		AssignmentBuf: 0,
		Assigner:      &mockAssigner{},
		Client:        &mockClient{ctrl},
	}
	p := NewAttester(context.Background(), cfg)

	mockServiceClient := internal.NewMockAttesterServiceClient(ctrl)

	doneChan := make(chan struct{})
	exitRoutine := make(chan bool)
	go func() {
		p.run(doneChan, mockServiceClient)
		<-exitRoutine
	}()

	p.assignmentChan <- nil
	testutil.AssertLogsContain(t, hook, "Could not marshal latest beacon block")
	doneChan <- struct{}{}
	exitRoutine <- true
	testutil.AssertLogsContain(t, hook, "Attester context closed")
}

func TestAttesterErrorLoop(t *testing.T) {
	hook := logTest.NewGlobal()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	cfg := &Config{
		AssignmentBuf: 0,
		Assigner:      &mockAssigner{},
		Client:        &mockClient{ctrl},
	}
	p := NewAttester(context.Background(), cfg)

	mockServiceClient := internal.NewMockAttesterServiceClient(ctrl)

	// Expect call to throw an error.
	mockServiceClient.EXPECT().AttestHead(
		gomock.Any(),
		gomock.Any(),
	).Return(nil, errors.New("could not attest head"))

	doneChan := make(chan struct{})
	exitRoutine := make(chan bool)
	go func() {
		p.run(doneChan, mockServiceClient)
		<-exitRoutine
	}()

	p.assignmentChan <- &pbp2p.BeaconBlock{SlotNumber: 999}
	testutil.AssertLogsContain(t, hook, "Performing attester responsibility")
	testutil.AssertLogsContain(t, hook, "could not attest head")
	doneChan <- struct{}{}
	exitRoutine <- true
	testutil.AssertLogsContain(t, hook, "Attester context closed")
}
