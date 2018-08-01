package rpcclient

import (
	"context"
	"fmt"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/testutil"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func TestLifecycle(t *testing.T) {
	hook := logTest.NewGlobal()
	rpcClientService := NewRPCClient(context.Background(), &Config{Endpoint: "merkle tries"})

	rpcClientService.Start()

	testutil.AssertLogsContain(t, hook, "Starting service")
	testutil.AssertLogsContain(t, hook, fmt.Sprintf("Could not connect to beacon node via RPC endpoint: %s", rpcClientService.endpoint))

	rpcClientService.Stop()
	testutil.AssertLogsContain(t, hook, "Stopping service")
}
