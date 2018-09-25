package rpc

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/event"
	"github.com/golang/mock/gomock"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/prysmaticlabs/prysm/beacon-chain/internal"
	"github.com/prysmaticlabs/prysm/beacon-chain/types"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/sirupsen/logrus"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func init() {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetOutput(ioutil.Discard)
}

type mockPOWChainService struct{}

func (m *mockPOWChainService) LatestBlockHash() common.Hash {
	return common.BytesToHash([]byte{})
}

type faultyChainService struct{}

func (f *faultyChainService) CanonicalHead() (*types.Block, error) {
	return nil, errors.New("failed")
}

func (f *faultyChainService) CanonicalCrystallizedState() *types.CrystallizedState {
	return nil
}

func (f *faultyChainService) CanonicalBlockFeed() *event.Feed {
	return nil
}

func (f *faultyChainService) CanonicalCrystallizedStateFeed() *event.Feed {
	return nil
}

type mockChainService struct {
	blockFeed       *event.Feed
	stateFeed       *event.Feed
	attestationFeed *event.Feed
}

type mockAttestationService struct{}

func (m *mockAttestationService) IncomingAttestationFeed() *event.Feed {
	return new(event.Feed)
}

func (m *mockAttestationService) ContainsAttestation(bitfield []byte, h [32]byte) (bool, error) {
	return true, nil
}

func (m *mockChainService) CurrentCrystallizedState() *types.CrystallizedState {
	cState, err := types.NewGenesisCrystallizedState()
	if err != nil {
		return nil
	}
	return cState
}

func (m *mockChainService) IncomingBlockFeed() *event.Feed {
	return new(event.Feed)
}

func (m *mockChainService) CanonicalBlockFeed() *event.Feed {
	return m.blockFeed
}

func (m *mockChainService) CanonicalCrystallizedStateFeed() *event.Feed {
	return m.stateFeed
}

func (m *mockChainService) CanonicalHead() (*types.Block, error) {
	data := &pbp2p.BeaconBlock{SlotNumber: 5}
	return types.NewBlock(data), nil
}

func (m *mockChainService) CanonicalCrystallizedState() *types.CrystallizedState {
	data := &pbp2p.CrystallizedState{}
	return types.NewCrystallizedState(data)
}

func newMockChainService() *mockChainService {
	return &mockChainService{
		blockFeed:       new(event.Feed),
		stateFeed:       new(event.Feed),
		attestationFeed: new(event.Feed),
	}
}

func TestLifecycle(t *testing.T) {
	hook := logTest.NewGlobal()
	cs := newMockChainService()
	rpcService := NewRPCService(context.Background(), &Config{
		Port:             "7348",
		CertFlag:         "alice.crt",
		KeyFlag:          "alice.key",
		CanonicalFetcher: cs,
	})

	rpcService.Start()

	testutil.AssertLogsContain(t, hook, "Starting service")
	testutil.AssertLogsContain(t, hook, fmt.Sprintf("RPC server listening on port :%s", rpcService.port))

	rpcService.Stop()
	testutil.AssertLogsContain(t, hook, "Stopping service")
}

func TestBadEndpoint(t *testing.T) {
	hook := logTest.NewGlobal()
	cs := newMockChainService()
	rpcService := NewRPCService(context.Background(), &Config{
		Port:             "ralph merkle!!!",
		CanonicalFetcher: cs,
	})

	rpcService.Start()

	testutil.AssertLogsContain(t, hook, "Starting service")
	testutil.AssertLogsContain(t, hook, fmt.Sprintf("Could not listen to port :%s", rpcService.port))

	rpcService.Stop()
	testutil.AssertLogsContain(t, hook, "Stopping service")
}

func TestInsecureEndpoint(t *testing.T) {
	hook := logTest.NewGlobal()
	cs := newMockChainService()
	rpcService := NewRPCService(context.Background(), &Config{
		Port:             "7777",
		CanonicalFetcher: cs,
	})

	rpcService.Start()

	testutil.AssertLogsContain(t, hook, "Starting service")
	testutil.AssertLogsContain(t, hook, fmt.Sprintf("RPC server listening on port :%s", rpcService.port))
	testutil.AssertLogsContain(t, hook, "You are using an insecure gRPC connection")

	rpcService.Stop()
	testutil.AssertLogsContain(t, hook, "Stopping service")
}

func TestCanonicalHead(t *testing.T) {
	mockChain := &mockChainService{}
	rpcService := NewRPCService(context.Background(), &Config{
		Port:             "6372",
		CanonicalFetcher: mockChain,
		ChainService:     mockChain,
		POWChainService:  &mockPOWChainService{},
	})
	if _, err := rpcService.CanonicalHead(context.Background(), &empty.Empty{}); err != nil {
		t.Errorf("Could not call CanonicalHead correctly: %v", err)
	}

	rpcService = NewRPCService(context.Background(), &Config{
		Port:             "6372",
		CanonicalFetcher: &faultyChainService{},
		ChainService:     &mockChainService{},
		POWChainService:  &mockPOWChainService{},
	})
	if _, err := rpcService.CanonicalHead(context.Background(), &empty.Empty{}); err == nil {
		t.Error("Expected error from faulty chain service, received nil")
	}
}

func TestGenesisTimeAndCanonicalState(t *testing.T) {
	mockChain := &mockChainService{}
	rpcService := NewRPCService(context.Background(), &Config{
		Port:             "6372",
		CanonicalFetcher: mockChain,
		ChainService:     mockChain,
		POWChainService:  &mockPOWChainService{},
	})
	res, err := rpcService.GenesisTimeAndCanonicalState(context.Background(), &empty.Empty{})
	if err != nil {
		t.Errorf("Could not call GenesisTimeAndCanonicalState correctly: %v", err)
	}
	genesis := types.NewGenesisBlock()
	if res.GenesisTimestamp.String() != genesis.Proto().GetTimestamp().String() {
		t.Errorf("Received different genesis timestamp, wanted: %v, received: %v", genesis.Proto().GetTimestamp(), res.GenesisTimestamp)
	}
}

func TestProposeBlock(t *testing.T) {
	mockChain := &mockChainService{}
	rpcService := NewRPCService(context.Background(), &Config{
		Port:             "6372",
		CanonicalFetcher: mockChain,
		ChainService:     mockChain,
		POWChainService:  &mockPOWChainService{},
	})
	req := &pb.ProposeRequest{
		SlotNumber: 5,
		ParentHash: []byte("parent-hash"),
		Timestamp:  ptypes.TimestampNow(),
	}
	if _, err := rpcService.ProposeBlock(context.Background(), req); err != nil {
		t.Errorf("Could not propose block correctly: %v", err)
	}
}

func TestLatestCrystallizedStateContextClosed(t *testing.T) {
	hook := logTest.NewGlobal()
	cs := newMockChainService()
	rpcService := NewRPCService(context.Background(), &Config{
		Port:             "8777",
		SubscriptionBuf:  0,
		CanonicalFetcher: cs,
	})
	exitRoutine := make(chan bool)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockStream := internal.NewMockBeaconService_LatestCrystallizedStateServer(ctrl)
	go func(tt *testing.T) {
		if err := rpcService.LatestCrystallizedState(&empty.Empty{}, mockStream); err != nil {
			tt.Errorf("Could not call RPC method: %v", err)
		}
		<-exitRoutine
	}(t)
	rpcService.cancel()
	exitRoutine <- true
	testutil.AssertLogsContain(t, hook, "RPC context closed, exiting goroutine")
}

func TestLatestCrystallizedState(t *testing.T) {
	hook := logTest.NewGlobal()
	cs := newMockChainService()
	rpcService := NewRPCService(context.Background(), &Config{
		Port:             "8773",
		SubscriptionBuf:  0,
		CanonicalFetcher: cs,
	})
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	exitRoutine := make(chan bool)

	mockStream := internal.NewMockBeaconService_LatestCrystallizedStateServer(ctrl)
	mockStream.EXPECT().Send(&pbp2p.CrystallizedState{}).Return(errors.New("something wrong"))
	// Tests a faulty stream.
	go func(tt *testing.T) {
		if err := rpcService.LatestCrystallizedState(&empty.Empty{}, mockStream); err.Error() != "something wrong" {
			tt.Errorf("Faulty stream should throw correct error, wanted 'something wrong', got %v", err)
		}
		<-exitRoutine
	}(t)
	rpcService.canonicalStateChan <- types.NewCrystallizedState(&pbp2p.CrystallizedState{})

	mockStream = internal.NewMockBeaconService_LatestCrystallizedStateServer(ctrl)
	mockStream.EXPECT().Send(&pbp2p.CrystallizedState{}).Return(nil)

	// Tests a good stream.
	go func(tt *testing.T) {
		if err := rpcService.LatestCrystallizedState(&empty.Empty{}, mockStream); err != nil {
			tt.Errorf("Could not call RPC method: %v", err)
		}
		<-exitRoutine
	}(t)
	rpcService.canonicalStateChan <- types.NewCrystallizedState(&pbp2p.CrystallizedState{})
	testutil.AssertLogsContain(t, hook, "Sending crystallized state to RPC clients")
	rpcService.cancel()
	exitRoutine <- true
}

func TestAttestHead(t *testing.T) {
	mockChain := &mockChainService{}
	mockAttestationService := &mockAttestationService{}
	rpcService := NewRPCService(context.Background(), &Config{
		Port:               "6372",
		ChainService:       mockChain,
		AttestationService: mockAttestationService,
	})
	req := &pb.AttestRequest{
		Attestation: &pbp2p.AggregatedAttestation{
			Slot:           999,
			ShardId:        1,
			ShardBlockHash: []byte{'a'},
		},
	}
	if _, err := rpcService.AttestHead(context.Background(), req); err != nil {
		t.Errorf("Could not attest head correctly: %v", err)
	}
}

func TestLatestAttestationContextClosed(t *testing.T) {
	hook := logTest.NewGlobal()
	mockAttestationService := &mockAttestationService{}
	rpcService := NewRPCService(context.Background(), &Config{Port: "8777", SubscriptionBuf: 0, AttestationService: mockAttestationService})
	exitRoutine := make(chan bool)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockStream := internal.NewMockBeaconService_LatestAttestationServer(ctrl)
	go func(tt *testing.T) {
		if err := rpcService.LatestAttestation(&empty.Empty{}, mockStream); err != nil {
			tt.Errorf("Could not call RPC method: %v", err)
		}
		<-exitRoutine
	}(t)
	rpcService.cancel()
	exitRoutine <- true
	testutil.AssertLogsContain(t, hook, "RPC context closed, exiting goroutine")
}

func TestLatestAttestation(t *testing.T) {
	hook := logTest.NewGlobal()
	attestationService := &mockAttestationService{}
	rpcService := NewRPCService(context.Background(), &Config{Port: "8777", SubscriptionBuf: 0, AttestationService: attestationService})
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	exitRoutine := make(chan bool)

	mockStream := internal.NewMockBeaconService_LatestAttestationServer(ctrl)
	mockStream.EXPECT().Send(&pbp2p.AggregatedAttestation{}).Return(errors.New("something wrong"))
	// Tests a faulty stream.
	go func(tt *testing.T) {
		if err := rpcService.LatestAttestation(&empty.Empty{}, mockStream); err.Error() != "something wrong" {
			tt.Errorf("Faulty stream should throw correct error, wanted 'something wrong', got %v", err)
		}
		<-exitRoutine
	}(t)
	rpcService.incomingAttestation <- &pbp2p.AggregatedAttestation{}

	mockStream = internal.NewMockBeaconService_LatestAttestationServer(ctrl)
	mockStream.EXPECT().Send(&pbp2p.AggregatedAttestation{}).Return(nil)

	// Tests a good stream.
	go func(tt *testing.T) {
		if err := rpcService.LatestAttestation(&empty.Empty{}, mockStream); err != nil {
			tt.Errorf("Could not call RPC method: %v", err)
		}
		<-exitRoutine
	}(t)
	rpcService.incomingAttestation <- &pbp2p.AggregatedAttestation{}
	testutil.AssertLogsContain(t, hook, "Sending attestation to RPC clients")
	rpcService.cancel()
	exitRoutine <- true
}

func TestValidatorSlotAndResponsibility(t *testing.T) {
	mockChain := &mockChainService{}
	rpcService := NewRPCService(context.Background(), &Config{
		Port:         "6372",
		ChainService: mockChain,
	})
	req := &pb.PublicKey{
		PublicKey: []byte{},
	}
	if _, err := rpcService.ValidatorSlotAndResponsibility(context.Background(), req); err != nil {
		t.Errorf("Could not get validator slot: %v", err)
	}
}

func TestValidatorIndex(t *testing.T) {
	mockChain := &mockChainService{}
	rpcService := NewRPCService(context.Background(), &Config{
		Port:         "6372",
		ChainService: mockChain,
	})
	req := &pb.PublicKey{
		PublicKey: []byte{},
	}
	if _, err := rpcService.ValidatorIndex(context.Background(), req); err != nil {
		t.Errorf("Could not get validator index: %v", err)
	}
}

func TestValidatorShardID(t *testing.T) {
	mockChain := &mockChainService{}
	rpcService := NewRPCService(context.Background(), &Config{
		Port:         "6372",
		ChainService: mockChain,
	})
	req := &pb.PublicKey{
		PublicKey: []byte{},
	}
	if _, err := rpcService.ValidatorShardID(context.Background(), req); err != nil {
		t.Errorf("Could not get validator shard ID: %v", err)
	}
}
