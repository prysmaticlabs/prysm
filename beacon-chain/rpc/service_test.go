package rpc

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/event"
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

type mockChainService struct {
	blockFeed *event.Feed
	stateFeed *event.Feed
}

func newMockChainService() *mockChainService {
	return &mockChainService{
		blockFeed: new(event.Feed),
		stateFeed: new(event.Feed),
	}
}

func (m *mockChainService) CanonicalBlockFeed() *event.Feed {
	return m.blockFeed
}

func (m *mockChainService) CanonicalCrystallizedStateFeed() *event.Feed {
	return m.stateFeed
}

func (m *mockChainService) CanonicalHead() (*types.Block, error) {
	data := &pbp2p.BeaconBlock{SlotNumber: 5}
	return types.NewBlock(data), nil
}

func (m *mockChainService) CanonicalCrystallizedState() *types.CrystallizedState {
	data := &pbp2p.CrystallizedState{}
	return types.NewCrystallizedState(data)
}

type faultyChainService struct{}

func (f *faultyChainService) CanonicalBlockFeed() *event.Feed {
	return nil
}

func (f *faultyChainService) CanonicalCrystallizedStateFeed() *event.Feed {
	return nil
}

func (f *faultyChainService) CanonicalHead() (*types.Block, error) {
	return nil, errors.New("oops failed")
}

func (f *faultyChainService) CanonicalCrystallizedState() *types.CrystallizedState {
	return nil
}

func TestLifecycle(t *testing.T) {
	hook := logTest.NewGlobal()
	cs := newMockChainService()
	rpcService := NewRPCService(context.Background(), &Config{
		Port:             "7348",
		CertFlag:         "alice.crt",
		KeyFlag:          "alice.key",
		CanonicalFetcher: cs,
	})

	rpcService.Start()

	testutil.AssertLogsContain(t, hook, "Starting service")
	testutil.AssertLogsContain(t, hook, fmt.Sprintf("RPC server listening on port :%s", rpcService.port))

	rpcService.Stop()
	testutil.AssertLogsContain(t, hook, "Stopping service")
}

func TestBadEndpoint(t *testing.T) {
	hook := logTest.NewGlobal()
	cs := newMockChainService()
	rpcService := NewRPCService(context.Background(), &Config{
		Port:             "ralph merkle!!!",
		CanonicalFetcher: cs,
	})

	rpcService.Start()

	testutil.AssertLogsContain(t, hook, "Starting service")
	testutil.AssertLogsContain(t, hook, fmt.Sprintf("Could not listen to port :%s", rpcService.port))

	rpcService.Stop()
	testutil.AssertLogsContain(t, hook, "Stopping service")
}

func TestInsecureEndpoint(t *testing.T) {
	hook := logTest.NewGlobal()
	cs := newMockChainService()
	rpcService := NewRPCService(context.Background(), &Config{
		Port:             "7777",
		CanonicalFetcher: cs,
	})

	rpcService.Start()

	testutil.AssertLogsContain(t, hook, "Starting service")
	testutil.AssertLogsContain(t, hook, fmt.Sprintf("RPC server listening on port :%s", rpcService.port))
	testutil.AssertLogsContain(t, hook, "You are using an insecure gRPC connection")

	rpcService.Stop()
	testutil.AssertLogsContain(t, hook, "Stopping service")
}

func TestRPCMethods(t *testing.T) {
	cs := newMockChainService()
	rpcService := NewRPCService(context.Background(), &Config{
		Port:             "7362",
		CanonicalFetcher: cs,
	})
	if _, err := rpcService.ProposeBlock(context.Background(), nil); err == nil {
		t.Error("Wanted error: unimplemented, received nil")
	}
	if _, err := rpcService.SignBlock(context.Background(), nil); err == nil {
		t.Error("Wanted error: unimplemented, received nil")
	}
}

func TestCanonicalHeadAndState(t *testing.T) {
	cs := newMockChainService()
	rpcService := NewRPCService(context.Background(), &Config{
		Port:             "7362",
		CanonicalFetcher: cs,
	})
	if _, err := rpcService.CanonicalHeadAndState(context.Background(), nil); err != nil {
		t.Errorf("Unexpected error when calling CanonicalHeadAndState: %v", err)
	}

	faulty := &faultyChainService{}
	rpcService = NewRPCService(context.Background(), &Config{
		Port:             "7362",
		CanonicalFetcher: faulty,
	})
	_, err := rpcService.CanonicalHeadAndState(context.Background(), nil)
	if !strings.Contains(err.Error(), "could not fetch canonical") {
		t.Errorf("Expected: could not fetch canonical, received %v", err)
	}
}

func TestFetchShuffledValidatorIndices(t *testing.T) {
	cs := newMockChainService()
	rpcService := NewRPCService(context.Background(), &Config{
		Port:             "6372",
		CanonicalFetcher: cs,
	})
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
	cs := newMockChainService()
	rpcService := NewRPCService(context.Background(), &Config{
		Port:             "6663",
		SubscriptionBuf:  0,
		CanonicalFetcher: cs,
	})
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
	cs := newMockChainService()
	rpcService := NewRPCService(context.Background(), &Config{
		Port:             "7771",
		SubscriptionBuf:  0,
		CanonicalFetcher: cs,
	})
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	exitRoutine := make(chan bool)

	mockStream := internal.NewMockBeaconService_LatestBeaconBlockServer(ctrl)
	mockStream.EXPECT().Send(&pbp2p.BeaconBlock{}).Return(errors.New("something wrong"))
	// Tests a faulty stream.
	go func(tt *testing.T) {
		if err := rpcService.LatestBeaconBlock(&empty.Empty{}, mockStream); err.Error() != "something wrong" {
			tt.Errorf("Faulty stream should throw correct error, wanted 'something wrong', got %v", err)
		}
		<-exitRoutine
	}(t)
	rpcService.canonicalBlockChan <- types.NewBlock(&pbp2p.BeaconBlock{})

	mockStream = internal.NewMockBeaconService_LatestBeaconBlockServer(ctrl)
	mockStream.EXPECT().Send(&pbp2p.BeaconBlock{}).Return(nil)

	// Tests a good stream.
	go func(tt *testing.T) {
		if err := rpcService.LatestBeaconBlock(&empty.Empty{}, mockStream); err != nil {
			tt.Errorf("Could not call RPC method: %v", err)
		}
		<-exitRoutine
	}(t)
	rpcService.canonicalBlockChan <- types.NewBlock(&pbp2p.BeaconBlock{})
	testutil.AssertLogsContain(t, hook, "Sending latest canonical block to RPC clients")
	rpcService.cancel()
	exitRoutine <- true
}

func TestLatestCrystallizedStateContextClosed(t *testing.T) {
	hook := logTest.NewGlobal()
	cs := newMockChainService()
	rpcService := NewRPCService(context.Background(), &Config{
		Port:             "8777",
		SubscriptionBuf:  0,
		CanonicalFetcher: cs,
	})
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
	cs := newMockChainService()
	rpcService := NewRPCService(context.Background(), &Config{
		Port:             "8773",
		SubscriptionBuf:  0,
		CanonicalFetcher: cs,
	})
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	exitRoutine := make(chan bool)

	mockStream := internal.NewMockBeaconService_LatestCrystallizedStateServer(ctrl)
	mockStream.EXPECT().Send(&pbp2p.CrystallizedState{}).Return(errors.New("something wrong"))
	// Tests a faulty stream.
	go func(tt *testing.T) {
		if err := rpcService.LatestCrystallizedState(&empty.Empty{}, mockStream); err.Error() != "something wrong" {
			tt.Errorf("Faulty stream should throw correct error, wanted 'something wrong', got %v", err)
		}
		<-exitRoutine
	}(t)
	rpcService.canonicalStateChan <- types.NewCrystallizedState(&pbp2p.CrystallizedState{})

	mockStream = internal.NewMockBeaconService_LatestCrystallizedStateServer(ctrl)
	mockStream.EXPECT().Send(&pbp2p.CrystallizedState{}).Return(nil)

	// Tests a good stream.
	go func(tt *testing.T) {
		if err := rpcService.LatestCrystallizedState(&empty.Empty{}, mockStream); err != nil {
			tt.Errorf("Could not call RPC method: %v", err)
		}
		<-exitRoutine
	}(t)
	rpcService.canonicalStateChan <- types.NewCrystallizedState(&pbp2p.CrystallizedState{})
	testutil.AssertLogsContain(t, hook, "Sending crystallized state to RPC clients")
	rpcService.cancel()
	exitRoutine <- true
}
