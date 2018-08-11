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
	rpcService := NewRPCService(context.Background(), &Config{Port: "9999", CertFlag: "alice.crt", KeyFlag: "alice.key"})

	rpcService.Start()

	testutil.AssertLogsContain(t, hook, "Starting service")
	testutil.AssertLogsContain(t, hook, fmt.Sprintf("RPC server listening on port :%s", rpcService.port))

	rpcService.Stop()
	testutil.AssertLogsContain(t, hook, "Stopping service")
}

func TestBadEndpoint(t *testing.T) {
	hook := logTest.NewGlobal()
	rpcService := NewRPCService(context.Background(), &Config{Port: "ralph merkle!!!"})

	rpcService.Start()

	testutil.AssertLogsContain(t, hook, "Starting service")
	testutil.AssertLogsContain(t, hook, fmt.Sprintf("Could not listen to port :%s", rpcService.port))

	rpcService.Stop()
	testutil.AssertLogsContain(t, hook, "Stopping service")
}

func TestInsecureEndpoint(t *testing.T) {
	hook := logTest.NewGlobal()
	rpcService := NewRPCService(context.Background(), &Config{Port: "9999"})

	rpcService.Start()

	testutil.AssertLogsContain(t, hook, "Starting service")
	testutil.AssertLogsContain(t, hook, fmt.Sprintf("RPC server listening on port :%s", rpcService.port))
	testutil.AssertLogsContain(t, hook, "You are using an insecure gRPC connection")

	rpcService.Stop()
	testutil.AssertLogsContain(t, hook, "Stopping service")
}

func TestRPCMethods(t *testing.T) {
	s := &Service{}
	if _, err := s.FetchShuffledValidatorIndices(context.Background(), nil); err == nil {
		t.Error("Wanted error: unimplemented, received nil")
	}
	if _, err := s.ProposeBlock(context.Background(), nil); err == nil {
		t.Error("Wanted error: unimplemented, received nil")
	}
	if _, err := s.SignBlock(context.Background(), nil); err == nil {
		t.Error("Wanted error: unimplemented, received nil")
	}
	if err := s.LatestBeaconBlock(nil, nil); err == nil {
		t.Error("Wanted error: unimplemented, received nil")
	}
	if err := s.LatestCrystallizedState(nil, nil); err == nil {
		t.Error("Wanted error: unimplemented, received nil")
	}
}
