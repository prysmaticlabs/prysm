package rpc

import (
	"context"
	"fmt"
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/types"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

type mockChainService struct{}

func (ms *mockChainService) BeaconHashHeightChan() <-chan *pb.BeaconHashHeightResponse {
	return make(chan *pb.BeaconHashHeightResponse, 0)
}

func (ms *mockChainService) CrystallizedStateChan() <-chan *types.CrystallizedState {
	return make(chan *types.CrystallizedState, 0)
}

func TestLifecycle(t *testing.T) {
	hook := logTest.NewGlobal()
	rpcService := NewRPCService(context.Background(), &Config{Port: "9999"}, &mockChainService{})

	rpcService.Start()

	testutil.AssertLogsContain(t, hook, "Starting service")
	testutil.AssertLogsContain(t, hook, fmt.Sprintf("RPC server listening on port :%s", rpcService.port))

	rpcService.Stop()
	testutil.AssertLogsContain(t, hook, "Stopping service")
}

func TestBadEndpoint(t *testing.T) {
	hook := logTest.NewGlobal()
	rpcService := NewRPCService(context.Background(), &Config{Port: "ralph merkle!!!"}, &mockChainService{})

	rpcService.Start()

	testutil.AssertLogsContain(t, hook, "Starting service")
	testutil.AssertLogsContain(t, hook, fmt.Sprintf("Could not listen to port :%s", rpcService.port))

	rpcService.Stop()
	testutil.AssertLogsContain(t, hook, "Stopping service")
}
