package rpc

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"testing"
	"time"

	mock "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	mockPOW "github.com/prysmaticlabs/prysm/beacon-chain/powchain/testing"
	mockSync "github.com/prysmaticlabs/prysm/beacon-chain/sync/initial-sync/testing"
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
	chainService := &mock.ChainService{
		Genesis: time.Now(),
	}
	rpcService := NewService(context.Background(), &Config{
		Port:                "7348",
		CertFlag:            "alice.crt",
		KeyFlag:             "alice.key",
		SyncService:         &mockSync.Sync{IsSyncing: false},
		BlockReceiver:       chainService,
		AttestationReceiver: chainService,
		HeadFetcher:         chainService,
		GenesisTimeFetcher:  chainService,
		POWChainService:     &mockPOW.POWChain{},
		StateNotifier:       chainService.StateNotifier(),
	})

	rpcService.Start()

	testutil.AssertLogsContain(t, hook, "listening on port")

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
	chainService := &mock.ChainService{Genesis: time.Now()}
	rpcService := NewService(context.Background(), &Config{
		Port:                "7777",
		SyncService:         &mockSync.Sync{IsSyncing: false},
		BlockReceiver:       chainService,
		GenesisTimeFetcher:  chainService,
		AttestationReceiver: chainService,
		HeadFetcher:         chainService,
		POWChainService:     &mockPOW.POWChain{},
		StateNotifier:       chainService.StateNotifier(),
	})

	rpcService.Start()

	testutil.AssertLogsContain(t, hook, fmt.Sprint("listening on port"))
	testutil.AssertLogsContain(t, hook, "You are using an insecure gRPC connection")

	rpcService.Stop()
}
