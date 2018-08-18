package rpc

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/prysmaticlabs/prysm/beacon-chain/internal"
	"github.com/prysmaticlabs/prysm/beacon-chain/types"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/sirupsen/logrus"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func init() {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetOutput(ioutil.Discard)
}

type mockAnnouncer struct {
	blockChan             chan *types.Block
	crystallizedStateChan chan *types.CrystallizedState
}

func newMockAnnouncer() *mockAnnouncer {
	return &mockAnnouncer{
		blockChan:             make(chan *types.Block),
		crystallizedStateChan: make(chan *types.CrystallizedState),
	}
}

func (m *mockAnnouncer) CanonicalBlockEvent() <-chan *types.Block {
	return m.blockChan
}

func (m *mockAnnouncer) CanonicalCrystallizedStateEvent() <-chan *types.CrystallizedState {
	return m.crystallizedStateChan
}

func TestLifecycle(t *testing.T) {
	hook := logTest.NewGlobal()
	rpcService := NewRPCService(context.Background(), &Config{Port: "7348", CertFlag: "alice.crt", KeyFlag: "alice.key"}, &mockAnnouncer{})

	rpcService.Start()

	testutil.AssertLogsContain(t, hook, "Starting service")
	testutil.AssertLogsContain(t, hook, fmt.Sprintf("RPC server listening on port :%s", rpcService.port))

	rpcService.Stop()
	testutil.AssertLogsContain(t, hook, "Stopping service")
}

func TestBadEndpoint(t *testing.T) {
	hook := logTest.NewGlobal()
	rpcService := NewRPCService(context.Background(), &Config{Port: "ralph merkle!!!"}, &mockAnnouncer{})

	rpcService.Start()

	testutil.AssertLogsContain(t, hook, "Starting service")
	testutil.AssertLogsContain(t, hook, fmt.Sprintf("Could not listen to port :%s", rpcService.port))

	rpcService.Stop()
	testutil.AssertLogsContain(t, hook, "Stopping service")
}

func TestInsecureEndpoint(t *testing.T) {
	hook := logTest.NewGlobal()
	rpcService := NewRPCService(context.Background(), &Config{Port: "7777"}, &mockAnnouncer{})

	rpcService.Start()

	testutil.AssertLogsContain(t, hook, "Starting service")
	testutil.AssertLogsContain(t, hook, fmt.Sprintf("RPC server listening on port :%s", rpcService.port))
	testutil.AssertLogsContain(t, hook, "You are using an insecure gRPC connection")

	rpcService.Stop()
	testutil.AssertLogsContain(t, hook, "Stopping service")
}

func TestRPCMethods(t *testing.T) {
	rpcService := NewRPCService(context.Background(), &Config{Port: "7362"}, &mockAnnouncer{})
	if _, err := rpcService.ProposeBlock(context.Background(), nil); err == nil {
		t.Error("Wanted error: unimplemented, received nil")
	}
	if _, err := rpcService.SignBlock(context.Background(), nil); err == nil {
		t.Error("Wanted error: unimplemented, received nil")
	}
}

func TestFetchShuffledValidatorIndices(t *testing.T) {
	rpcService := NewRPCService(context.Background(), &Config{Port: "6372"}, &mockAnnouncer{})
	res, err := rpcService.FetchShuffledValidatorIndices(context.Background(), &pb.ShuffleRequest{})
	if err != nil {
		t.Fatalf("Could not call RPC method: %v", err)
	}
	if len(res.ShuffledValidatorIndices) != 100 {
		t.Errorf("Expected 100 validators in the shuffled indices, received %d", len(res.ShuffledValidatorIndices))
	}
}

func TestLatestBeaconBlockContextClosed(t *testing.T) {
	hook := logTest.NewGlobal()
	rpcService := NewRPCService(context.Background(), &Config{Port: "6663"}, &mockAnnouncer{})
	exitRoutine := make(chan bool)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockStream := internal.NewMockBeaconService_LatestBeaconBlockServer(ctrl)
	go func(tt *testing.T) {
		if err := rpcService.LatestBeaconBlock(&empty.Empty{}, mockStream); err != nil {
			tt.Errorf("Could not call RPC method: %v", err)
		}
		<-exitRoutine
	}(t)
	rpcService.cancel()
	exitRoutine <- true
	testutil.AssertLogsContain(t, hook, "RPC context closed, exiting goroutine")
}

func TestLatestBeaconBlock(t *testing.T) {
	hook := logTest.NewGlobal()
	announcer := newMockAnnouncer()
	rpcService := NewRPCService(context.Background(), &Config{Port: "7771"}, announcer)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStream := internal.NewMockBeaconService_LatestBeaconBlockServer(ctrl)
	mockStream.EXPECT().Send(&pbp2p.BeaconBlock{}).Return(errors.New("something wrong"))
	// Tests a faulty stream.
	go func(tt *testing.T) {
		if err := rpcService.LatestBeaconBlock(&empty.Empty{}, mockStream); err.Error() != "something wrong" {
			tt.Errorf("Faulty stream should throw correct error, wanted 'something wrong', got %v", err)
		}
	}(t)
	announcer.blockChan <- types.NewBlock(&pbp2p.BeaconBlock{})

	mockStream = internal.NewMockBeaconService_LatestBeaconBlockServer(ctrl)
	mockStream.EXPECT().Send(&pbp2p.BeaconBlock{}).Return(nil)

	// Tests a good stream.
	go func(tt *testing.T) {
		if err := rpcService.LatestBeaconBlock(&empty.Empty{}, mockStream); err != nil {
			tt.Errorf("Could not call RPC method: %v", err)
		}
	}(t)
	announcer.blockChan <- types.NewBlock(&pbp2p.BeaconBlock{})
	testutil.AssertLogsContain(t, hook, "Sending latest canonical block to RPC clients")
	rpcService.cancel()
}

func TestLatestCrystallizedStateContextClosed(t *testing.T) {
	hook := logTest.NewGlobal()
	rpcService := NewRPCService(context.Background(), &Config{Port: "8777"}, &mockAnnouncer{})
	exitRoutine := make(chan bool)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockStream := internal.NewMockBeaconService_LatestCrystallizedStateServer(ctrl)
	go func(tt *testing.T) {
		if err := rpcService.LatestCrystallizedState(&empty.Empty{}, mockStream); err != nil {
			tt.Errorf("Could not call RPC method: %v", err)
		}
		<-exitRoutine
	}(t)
	rpcService.cancel()
	exitRoutine <- true
	testutil.AssertLogsContain(t, hook, "RPC context closed, exiting goroutine")
}

func TestLatestCrystallizedState(t *testing.T) {
	hook := logTest.NewGlobal()
	announcer := newMockAnnouncer()
	rpcService := NewRPCService(context.Background(), &Config{Port: "8773"}, announcer)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStream := internal.NewMockBeaconService_LatestCrystallizedStateServer(ctrl)
	mockStream.EXPECT().Send(&pbp2p.CrystallizedState{}).Return(errors.New("something wrong"))
	// Tests a faulty stream.
	go func(tt *testing.T) {
		if err := rpcService.LatestCrystallizedState(&empty.Empty{}, mockStream); err.Error() != "something wrong" {
			tt.Errorf("Faulty stream should throw correct error, wanted 'something wrong', got %v", err)
		}
	}(t)
	announcer.crystallizedStateChan <- types.NewCrystallizedState(&pbp2p.CrystallizedState{})

	mockStream = internal.NewMockBeaconService_LatestCrystallizedStateServer(ctrl)
	mockStream.EXPECT().Send(&pbp2p.CrystallizedState{}).Return(nil)

	// Tests a good stream.
	go func(tt *testing.T) {
		if err := rpcService.LatestCrystallizedState(&empty.Empty{}, mockStream); err != nil {
			tt.Errorf("Could not call RPC method: %v", err)
		}
	}(t)
	announcer.crystallizedStateChan <- types.NewCrystallizedState(&pbp2p.CrystallizedState{})
	testutil.AssertLogsContain(t, hook, "Sending crystallized state to RPC clients")
	rpcService.cancel()
}
