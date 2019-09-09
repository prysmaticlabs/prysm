package rpc

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"testing"

	mock "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	mockSync "github.com/prysmaticlabs/prysm/beacon-chain/sync/testing"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/sirupsen/logrus"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func init() {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetOutput(ioutil.Discard)
}

func TestLifecycle_OK(t *testing.T) {
	hook := logTest.NewGlobal()
	rpcService := NewService(context.Background(), &Config{
		Port:                "7348",
		CertFlag:            "alice.crt",
		KeyFlag:             "alice.key",
		SyncService:         &mockSync.Sync{IsSyncing: false},
		BlockReceiver:       &mock.ChainService{},
		AttestationReceiver: &mock.ChainService{},
		HeadFetcher:         &mock.ChainService{},
		StateFeedListener:   &mock.ChainService{},
	})

	rpcService.Start()

	testutil.AssertLogsContain(t, hook, "Starting service")
	testutil.AssertLogsContain(t, hook, "Listening on port")

	rpcService.Stop()
	testutil.AssertLogsContain(t, hook, "Stopping service")

}

func TestRPC_BadEndpoint(t *testing.T) {
	hook := logTest.NewGlobal()

	rpcService := NewService(context.Background(), &Config{
		Port:                "ralph merkle!!!",
		SyncService:         &mockSync.Sync{IsSyncing: false},
		BlockReceiver:       &mock.ChainService{},
		AttestationReceiver: &mock.ChainService{},
		HeadFetcher:         &mock.ChainService{},
		StateFeedListener:   &mock.ChainService{},
	})

	testutil.AssertLogsDoNotContain(t, hook, "Could not listen to port in Start()")
	testutil.AssertLogsDoNotContain(t, hook, "Could not load TLS keys")
	testutil.AssertLogsDoNotContain(t, hook, "Could not serve gRPC")

	rpcService.Start()

	testutil.AssertLogsContain(t, hook, "Starting service")
	testutil.AssertLogsContain(t, hook, "Could not listen to port in Start()")

	rpcService.Stop()
}

func TestStatus_CredentialError(t *testing.T) {
	credentialErr := errors.New("credentialError")
	s := &Service{credentialError: credentialErr}

	if err := s.Status(); err != s.credentialError {
		t.Errorf("Wanted: %v, got: %v", s.credentialError, s.Status())
	}
}

func TestRPC_InsecureEndpoint(t *testing.T) {
	hook := logTest.NewGlobal()
	rpcService := NewService(context.Background(), &Config{
		Port:                "7777",
		SyncService:         &mockSync.Sync{IsSyncing: false},
		BlockReceiver:       &mock.ChainService{},
		AttestationReceiver: &mock.ChainService{},
		HeadFetcher:         &mock.ChainService{},
		StateFeedListener:   &mock.ChainService{},
	})

	rpcService.Start()

	testutil.AssertLogsContain(t, hook, "Starting service")
	testutil.AssertLogsContain(t, hook, fmt.Sprint("Listening on port"))
	testutil.AssertLogsContain(t, hook, "You are using an insecure gRPC connection")

	rpcService.Stop()
	testutil.AssertLogsContain(t, hook, "Stopping service")
}
