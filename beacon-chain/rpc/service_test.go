package rpc

import (
	"context"
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/prysmaticlabs/prysm/beacon-chain/internal"
	"github.com/prysmaticlabs/prysm/beacon-chain/types"
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

func (m *mockAnnouncer) CanonicalBlockAnnouncement() <-chan *types.Block {
	return m.blockChan
}

func (m *mockAnnouncer) CanonicalCrystallizedStateAnnouncement() <-chan *types.CrystallizedState {
	return m.crystallizedStateChan
}

func TestLifecycle(t *testing.T) {
	hook := logTest.NewGlobal()
	rpcService := NewRPCService(context.Background(), &Config{Port: "9999", CertFlag: "alice.crt", KeyFlag: "alice.key"}, &mockAnnouncer{})

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
	rpcService := NewRPCService(context.Background(), &Config{Port: "9999"}, &mockAnnouncer{})

	rpcService.Start()

	testutil.AssertLogsContain(t, hook, "Starting service")
	testutil.AssertLogsContain(t, hook, fmt.Sprintf("RPC server listening on port :%s", rpcService.port))
	testutil.AssertLogsContain(t, hook, "You are using an insecure gRPC connection")

	rpcService.Stop()
	testutil.AssertLogsContain(t, hook, "Stopping service")
}

func TestRPCMethods(t *testing.T) {
	rpcService := NewRPCService(context.Background(), &Config{Port: "9999"}, &mockAnnouncer{})
	if _, err := rpcService.ProposeBlock(context.Background(), nil); err == nil {
		t.Error("Wanted error: unimplemented, received nil")
	}
	if _, err := rpcService.SignBlock(context.Background(), nil); err == nil {
		t.Error("Wanted error: unimplemented, received nil")
	}
}

func TestFetchShuffledValidatorIndices(t *testing.T) {
	rpcService := NewRPCService(context.Background(), &Config{Port: "9999"}, &mockAnnouncer{})
	res, err := rpcService.FetchShuffledValidatorIndices(context.Background(), &pb.ShuffleRequest{})
	if err != nil {
		t.Fatalf("Could not call RPC method: %v", err)
	}
	if len(res.ShuffledValidatorIndices) != 100 {
		t.Errorf("Expected 100 validators in the shuffled indices, received %d", len(res.ShuffledValidatorIndices))
	}
}

func TestLatestBeaconBlockClosedContext(t *testing.T) {
	hook := logTest.NewGlobal()
	rpcService := NewRPCService(context.Background(), &Config{Port: "9999"}, &mockAnnouncer{})
	exitRoutine := make(chan bool)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockStream := internal.NewMockBeaconService_LatestBeaconBlockServer(ctrl)
	go func() {
		if err := rpcService.LatestBeaconBlock(&empty.Empty{}, mockStream); err != nil {
			t.Fatalf("Could not call RPC method: %v", err)
		}
		<-exitRoutine
	}()
	rpcService.cancel()
	exitRoutine <- true
	testutil.AssertLogsContain(t, hook, "RPC context closed, exiting goroutine")
}
