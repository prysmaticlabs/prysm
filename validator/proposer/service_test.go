package proposer

import (
	"context"
	"errors"
	"fmt"
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

func (mc *mockClient) ProposerServiceClient() pb.ProposerServiceClient {
	return internal.NewMockProposerServiceClient(mc.ctrl)
}

type mockAssigner struct{}

func (m *mockAssigner) ProposerAssignmentFeed() *event.Feed {
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
	p := NewProposer(context.Background(), cfg)
	p.Start()
	testutil.AssertLogsContain(t, hook, "Starting service")
	p.Stop()
	testutil.AssertLogsContain(t, hook, "Stopping service")
}

func TestProposerLoop(t *testing.T) {
	hook := logTest.NewGlobal()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	cfg := &Config{
		AssignmentBuf: 0,
		Assigner:      &mockAssigner{},
		Client:        &mockClient{ctrl},
	}
	p := NewProposer(context.Background(), cfg)

	mockServiceClient := internal.NewMockProposerServiceClient(ctrl)

	// Expect first call to go through correctly.
	mockServiceClient.EXPECT().ProposeBlock(
		gomock.Any(),
		gomock.Any(),
	).Return(&pb.ProposeResponse{
		BlockHash: []byte("hi"),
	}, nil)

	doneChan := make(chan struct{})
	exitRoutine := make(chan bool)
	go func() {
		p.run(doneChan, mockServiceClient)
		<-exitRoutine
	}()
	p.assignmentChan <- &pbp2p.BeaconBlock{}
	testutil.AssertLogsContain(t, hook, "Performing proposer responsibility")
	testutil.AssertLogsContain(t, hook, fmt.Sprintf("Block proposed successfully with hash 0x%x", []byte("hi")))
	doneChan <- struct{}{}
	exitRoutine <- true
	testutil.AssertLogsContain(t, hook, "Proposer context closed")
}

func TestProposerErrorLoop(t *testing.T) {
	hook := logTest.NewGlobal()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	cfg := &Config{
		AssignmentBuf: 0,
		Assigner:      &mockAssigner{},
		Client:        &mockClient{ctrl},
	}
	p := NewProposer(context.Background(), cfg)

	mockServiceClient := internal.NewMockProposerServiceClient(ctrl)

	// Expect call to throw an error.
	mockServiceClient.EXPECT().ProposeBlock(
		gomock.Any(),
		gomock.Any(),
	).Return(nil, errors.New("bad block proposed"))

	doneChan := make(chan struct{})
	exitRoutine := make(chan bool)
	go func() {
		p.run(doneChan, mockServiceClient)
		<-exitRoutine
	}()
	p.assignmentChan <- &pbp2p.BeaconBlock{}
	testutil.AssertLogsContain(t, hook, "Performing proposer responsibility")
	testutil.AssertLogsContain(t, hook, "bad block proposed")
	doneChan <- struct{}{}
	exitRoutine <- true
	testutil.AssertLogsContain(t, hook, "Proposer context closed")
}
