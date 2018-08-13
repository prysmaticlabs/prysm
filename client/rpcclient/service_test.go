package rpcclient

import (
	"context"
	"testing"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func TestLifecycle(t *testing.T) {
	hook := logTest.NewGlobal()
	rpcClientService := NewRPCClient(
		context.Background(),
		&Config{
			Endpoint: "merkle tries",
			CertFlag: "alice.crt",
		},
	)
	rpcClientService.Start()
	testutil.AssertLogsContain(t, hook, "Starting service")
	rpcClientService.Stop()
	testutil.AssertLogsContain(t, hook, "Stopping service")
}

func TestInsecure(t *testing.T) {
	hook := logTest.NewGlobal()
	rpcClientService := NewRPCClient(
		context.Background(),
		&Config{
			Endpoint: "merkle tries",
		},
	)
	rpcClientService.Start()
	testutil.AssertLogsContain(t, hook, "Starting service")
	testutil.AssertLogsContain(t, hook, "You are using an insecure gRPC connection")
	rpcClientService.Stop()
	testutil.AssertLogsContain(t, hook, "Stopping service")
}

func TestBeaconServiceClient(t *testing.T) {
	rpcClientService := NewRPCClient(
		context.Background(),
		&Config{
			Endpoint: "merkle tries",
		},
	)
	rpcClientService.conn = nil
	client := rpcClientService.BeaconServiceClient()
	if _, ok := client.(pb.BeaconServiceClient); !ok {
		t.Error("Beacon service client function does not implement interface")
	}
}
