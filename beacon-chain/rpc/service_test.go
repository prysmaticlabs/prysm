package rpc

import (
	"context"
	"fmt"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/testutil"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func TestLifecycle(t *testing.T) {
	hook := logTest.NewGlobal()
	rpcService := NewRPCService(context.Background(), &Config{Endpoint: "9999"})

	rpcService.Start()

	testutil.AssertLogsContain(t, hook, "Starting service")
	testutil.AssertLogsContain(t, hook, fmt.Sprintf("RPC server listening on port :%s", rpcService.endpoint))

	rpcService.Stop()
	testutil.AssertLogsContain(t, hook, "Stopping service")
}

func TestBadEndpoint(t *testing.T) {
	hook := logTest.NewGlobal()
	rpcService := NewRPCService(context.Background(), &Config{Endpoint: "ralph merkle!!!"})

	rpcService.Start()

	testutil.AssertLogsContain(t, hook, "Starting service")
	testutil.AssertLogsContain(t, hook, fmt.Sprintf("Could not listen to port :%s", rpcService.endpoint))

	rpcService.Stop()
	testutil.AssertLogsContain(t, hook, "Stopping service")
}
