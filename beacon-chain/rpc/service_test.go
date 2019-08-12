package rpc

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/gogo/protobuf/proto"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
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

func (ms *mockOperationService) AttestationPool(_ context.Context, headRoot []byte, expectedSlot uint64) ([]*ethpb.Attestation, error) {
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

type mockChainService struct {
	blockFeed            *event.Feed
	stateFeed            *event.Feed
	attestationFeed      *event.Feed
	stateInitializedFeed *event.Feed
	headBlock            *ethpb.BeaconBlock
	headState            *pb.BeaconState
}

func (ms *mockChainService) StateInitializedFeed() *event.Feed {
	return ms.stateInitializedFeed
}

func (ms *mockChainService) ReceiveBlock(ctx context.Context, block *ethpb.BeaconBlock) error {
	return nil
}

func (ms *mockChainService) ReceiveAttestation(ctx context.Context, att *ethpb.Attestation) error {
	return nil
}

func (ms *mockChainService) CanonicalBlockFeed() *event.Feed {
	return new(event.Feed)
}

func (ms *mockChainService) CanonicalRoot(slot uint64) []byte {
	return []byte{'A'}
}

func (ms *mockChainService) FinalizedState(ctx context.Context) (*pb.BeaconState, error) {
	return nil, nil
}

func (ms *mockChainService) FinalizedBlock() (*ethpb.BeaconBlock, error) {
	return nil, nil

}

func (ms *mockChainService) FinalizedCheckpt() *ethpb.Checkpoint {
	return nil
}

func (ms *mockChainService) JustifiedCheckpt() *ethpb.Checkpoint {
	return nil
}

func (ms *mockChainService) HeadSlot() uint64 {
	return 0
}

func (ms *mockChainService) HeadRoot() []byte {
	return nil
}

func (ms *mockChainService) HeadBlock() (*ethpb.BeaconBlock, error) {
	return ms.headBlock, nil
}

func (ms *mockChainService) HeadState() (*pb.BeaconState, error) {
	return ms.headState, nil
}

func newMockChainService() *mockChainService {
	return &mockChainService{
		blockFeed:            new(event.Feed),
		stateFeed:            new(event.Feed),
		attestationFeed:      new(event.Feed),
		stateInitializedFeed: new(event.Feed),
	}
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
	rpcService := NewRPCService(context.Background(), &Config{
		Port:        "7348",
		CertFlag:    "alice.crt",
		KeyFlag:     "alice.key",
		SyncService: &mockSyncService{},
	})

	rpcService.Start()

	testutil.AssertLogsContain(t, hook, "Starting service")
	testutil.AssertLogsContain(t, hook, "Listening on port")

	rpcService.Stop()
	testutil.AssertLogsContain(t, hook, "Stopping service")

}

func TestRPC_BadEndpoint(t *testing.T) {
	hook := logTest.NewGlobal()

	rpcService := NewRPCService(context.Background(), &Config{
		Port:        "ralph merkle!!!",
		SyncService: &mockSyncService{},
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
	rpcService := NewRPCService(context.Background(), &Config{
		Port:        "7777",
		SyncService: &mockSyncService{},
	})

	rpcService.Start()

	testutil.AssertLogsContain(t, hook, "Starting service")
	testutil.AssertLogsContain(t, hook, fmt.Sprint("Listening on port"))
	testutil.AssertLogsContain(t, hook, "You are using an insecure gRPC connection")

	rpcService.Stop()
	testutil.AssertLogsContain(t, hook, "Stopping service")
}
