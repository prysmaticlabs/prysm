package rpc

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/gogo/protobuf/proto"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
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

type TestLogger struct {
	logrus.FieldLogger
	testMap map[string]interface{}
}

func (t *TestLogger) Errorf(format string, args ...interface{}) {
	t.testMap["error"] = true
}

type mockOperationService struct {
	pendingAttestations []*pb.Attestation
}

func (ms *mockOperationService) IncomingAttFeed() *event.Feed {
	return new(event.Feed)
}

func (ms *mockOperationService) IncomingExitFeed() *event.Feed {
	return new(event.Feed)
}

func (ms *mockOperationService) HandleAttestations(_ context.Context, _ proto.Message) error {
	return nil
}

func (ms *mockOperationService) PendingAttestations() ([]*pb.Attestation, error) {
	if ms.pendingAttestations != nil {
		return ms.pendingAttestations, nil
	}
	return []*pb.Attestation{
		{
			AggregationBitfield: []byte("A"),
			Data: &pb.AttestationData{
				Slot: params.BeaconConfig().GenesisSlot + params.BeaconConfig().SlotsPerEpoch,
			},
		},
		{
			AggregationBitfield: []byte("B"),
			Data: &pb.AttestationData{
				Slot: params.BeaconConfig().GenesisSlot + params.BeaconConfig().SlotsPerEpoch,
			},
		},
		{
			AggregationBitfield: []byte("C"),
			Data: &pb.AttestationData{
				Slot: params.BeaconConfig().GenesisSlot + params.BeaconConfig().SlotsPerEpoch,
			},
		},
	}, nil
}

type mockChainService struct {
	blockFeed            *event.Feed
	stateFeed            *event.Feed
	attestationFeed      *event.Feed
	stateInitializedFeed *event.Feed
}

func (m *mockChainService) StateInitializedFeed() *event.Feed {
	return m.stateInitializedFeed
}

func (m *mockChainService) ReceiveBlock(ctx context.Context, block *pb.BeaconBlock) (*pb.BeaconState, error) {
	return &pb.BeaconState{}, nil
}

func (m *mockChainService) ApplyForkChoiceRule(ctx context.Context, block *pb.BeaconBlock, computedState *pb.BeaconState) error {
	return nil
}

func (m *mockChainService) CanonicalBlockFeed() *event.Feed {
	return new(event.Feed)
}

func newMockChainService() *mockChainService {
	return &mockChainService{
		blockFeed:            new(event.Feed),
		stateFeed:            new(event.Feed),
		attestationFeed:      new(event.Feed),
		stateInitializedFeed: new(event.Feed),
	}
}

func TestLifecycle_OK(t *testing.T) {
	hook := logTest.NewGlobal()
	rpcService := NewRPCService(context.Background(), &Config{
		Port:     "7348",
		CertFlag: "alice.crt",
		KeyFlag:  "alice.key",
	})

	rpcService.Start()

	testutil.AssertLogsContain(t, hook, "Starting service")
	testutil.AssertLogsContain(t, hook, fmt.Sprintf("RPC server listening on port :%s", rpcService.port))

	rpcService.Stop()
	testutil.AssertLogsContain(t, hook, "Stopping service")

}

func TestRPC_BadEndpoint(t *testing.T) {
	fl := logrus.WithField("prefix", "rpc")

	log = &TestLogger{
		FieldLogger: fl,
		testMap:     make(map[string]interface{}),
	}

	hook := logTest.NewLocal(fl.Logger)

	rpcService := NewRPCService(context.Background(), &Config{
		Port: "ralph merkle!!!",
	})

	if _, ok := log.(*TestLogger).testMap["error"]; ok {
		t.Fatal("Error in Start() occurred before expected")
	}

	rpcService.Start()

	if _, ok := log.(*TestLogger).testMap["error"]; !ok {
		t.Fatal("No error occurred. Expected Start() to output an error")
	}

	testutil.AssertLogsContain(t, hook, "Starting service")

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
		Port: "7777",
	})

	rpcService.Start()

	testutil.AssertLogsContain(t, hook, "Starting service")
	testutil.AssertLogsContain(t, hook, fmt.Sprintf("RPC server listening on port :%s", rpcService.port))
	testutil.AssertLogsContain(t, hook, "You are using an insecure gRPC connection")

	rpcService.Stop()
	testutil.AssertLogsContain(t, hook, "Stopping service")
}
