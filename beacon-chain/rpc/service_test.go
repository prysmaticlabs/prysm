package rpc

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/gogo/protobuf/proto"
	mock "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/event"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/sirupsen/logrus"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func init() {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetOutput(ioutil.Discard)
}

type mockOperationService struct {
	pendingAttestations []*ethpb.Attestation
}

func (ms *mockOperationService) IncomingAttFeed() *event.Feed {
	return new(event.Feed)
}

func (ms *mockOperationService) IncomingExitFeed() *event.Feed {
	return new(event.Feed)
}

func (ms *mockOperationService) HandleAttestation(_ context.Context, _ proto.Message) error {
	return nil
}

func (ms *mockOperationService) IsAttCanonical(_ context.Context, att *ethpb.Attestation) (bool, error) {
	return true, nil
}

func (ms *mockOperationService) AttestationPool(_ context.Context, expectedSlot uint64) ([]*ethpb.Attestation, error) {
	if ms.pendingAttestations != nil {
		return ms.pendingAttestations, nil
	}
	return []*ethpb.Attestation{
		{
			AggregationBits: []byte{0xC0},
			Data: &ethpb.AttestationData{
				Crosslink: &ethpb.Crosslink{
					Shard:    params.BeaconConfig().SlotsPerEpoch,
					DataRoot: params.BeaconConfig().ZeroHash[:],
				},
			},
		},
		{
			AggregationBits: []byte{0xC1},
			Data: &ethpb.AttestationData{
				Crosslink: &ethpb.Crosslink{
					Shard:    params.BeaconConfig().SlotsPerEpoch,
					DataRoot: params.BeaconConfig().ZeroHash[:],
				},
			},
		},
		{
			AggregationBits: []byte{0xC2},
			Data: &ethpb.AttestationData{
				Crosslink: &ethpb.Crosslink{
					Shard:    params.BeaconConfig().SlotsPerEpoch,
					DataRoot: params.BeaconConfig().ZeroHash[:],
				},
			},
		},
	}, nil
}

type mockSyncService struct {
}

func (ms *mockSyncService) Status() error {
	return nil
}

func (ms *mockSyncService) Syncing() bool {
	return false
}

func TestLifecycle_OK(t *testing.T) {
	hook := logTest.NewGlobal()
	rpcService := NewService(context.Background(), &Config{
		Port:         "7348",
		CertFlag:     "alice.crt",
		KeyFlag:      "alice.key",
		SyncService:  &mockSyncService{},
		ChainService: &mock.ChainService{},
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
		Port:         "ralph merkle!!!",
		SyncService:  &mockSyncService{},
		ChainService: &mock.ChainService{},
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
		Port:         "7777",
		SyncService:  &mockSyncService{},
		ChainService: &mock.ChainService{},
	})

	rpcService.Start()

	testutil.AssertLogsContain(t, hook, "Starting service")
	testutil.AssertLogsContain(t, hook, fmt.Sprint("Listening on port"))
	testutil.AssertLogsContain(t, hook, "You are using an insecure gRPC connection")

	rpcService.Stop()
	testutil.AssertLogsContain(t, hook, "Stopping service")
}
