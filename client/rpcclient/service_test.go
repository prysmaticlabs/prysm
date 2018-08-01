package rpcclient

import (
	"context"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/testutil"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func TestLifecycle(t *testing.T) {
	hook := logTest.NewGlobal()
	rpcClientService := NewRPCClient(context.Background(), &Config{Endpoint: "merkle tries"})

	rpcClientService.Start()

	testutil.AssertLogsContain(t, hook, "Starting service")
	testutil.AssertLogsContain(t, hook, "Dialing beacon node RPC endpoint")

	rpcClientService.Stop()
	testutil.AssertLogsContain(t, hook, "Stopping service")
}
