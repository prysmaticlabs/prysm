package rpc

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	ptypes "github.com/gogo/protobuf/types"
	"github.com/golang/mock/gomock"
	mockClient "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	mockRPC "github.com/prysmaticlabs/prysm/beacon-chain/rpc/testing"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/event"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/trieutil"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

var closedContext = "context closed"
var mockSig [96]byte
var mockCreds [32]byte

type faultyPOWChainService struct {
	chainStartFeed *event.Feed
	hashesByHeight map[int][]byte
}

func (f *faultyPOWChainService) HasChainStarted() bool {
	return false
}
func (f *faultyPOWChainService) ETH2GenesisTime() (uint64, *big.Int) {
	return 0, big.NewInt(0)
}

func (f *faultyPOWChainService) ChainStartFeed() *event.Feed {
	return f.chainStartFeed
}
func (f *faultyPOWChainService) LatestBlockHeight() *big.Int {
	return big.NewInt(0)
}

func (f *faultyPOWChainService) BlockExists(_ context.Context, hash common.Hash) (bool, *big.Int, error) {
	if f.hashesByHeight == nil {
		return false, big.NewInt(1), errors.New("failed")
	}

	return true, big.NewInt(1), nil
}

func (f *faultyPOWChainService) BlockHashByHeight(_ context.Context, height *big.Int) (common.Hash, error) {
	return [32]byte{}, errors.New("failed")
}

func (f *faultyPOWChainService) BlockTimeByHeight(_ context.Context, height *big.Int) (uint64, error) {
	return 0, errors.New("failed")
}

func (f *faultyPOWChainService) BlockNumberByTimestamp(_ context.Context, _ uint64) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (f *faultyPOWChainService) DepositRoot() [32]byte {
	return [32]byte{}
}

func (f *faultyPOWChainService) DepositTrie() *trieutil.MerkleTrie {
	return &trieutil.MerkleTrie{}
}

func (f *faultyPOWChainService) ChainStartDeposits() []*ethpb.Deposit {
	return []*ethpb.Deposit{}
}

func (f *faultyPOWChainService) ChainStartDepositHashes() ([][]byte, error) {
	return [][]byte{}, errors.New("hashing failed")
}

func (f *faultyPOWChainService) ChainStartETH1Data() *ethpb.Eth1Data {
	return &ethpb.Eth1Data{}
}

type mockPOWChainService struct {
	chainStartFeed      *event.Feed
	latestBlockNumber   *big.Int
	hashesByHeight      map[int][]byte
	blockTimeByHeight   map[int]uint64
	blockNumberByHeight map[uint64]*big.Int
	eth1Data            *ethpb.Eth1Data
	genesisEth1Block    *big.Int
}

func (m *mockPOWChainService) HasChainStarted() bool {
	return true
}

func (m *mockPOWChainService) ETH2GenesisTime() (uint64, *big.Int) {
	blk := m.genesisEth1Block
	if blk == nil {
		blk = big.NewInt(0)
	}
	return uint64(time.Unix(0, 0).Unix()), blk
}
func (m *mockPOWChainService) ChainStartFeed() *event.Feed {
	return m.chainStartFeed
}
func (m *mockPOWChainService) LatestBlockHeight() *big.Int {
	return m.latestBlockNumber
}

func (m *mockPOWChainService) DepositTrie() *trieutil.MerkleTrie {
	return &trieutil.MerkleTrie{}
}

func (m *mockPOWChainService) BlockExists(_ context.Context, hash common.Hash) (bool, *big.Int, error) {
	// Reverse the map of heights by hash.
	heightsByHash := make(map[[32]byte]int)
	for k, v := range m.hashesByHeight {
		h := bytesutil.ToBytes32(v)
		heightsByHash[h] = k
	}
	val, ok := heightsByHash[hash]
	if !ok {
		return false, nil, fmt.Errorf("could not fetch height for hash: %#x", hash)
	}
	return true, big.NewInt(int64(val)), nil
}

func (m *mockPOWChainService) BlockHashByHeight(_ context.Context, height *big.Int) (common.Hash, error) {
	k := int(height.Int64())
	val, ok := m.hashesByHeight[k]
	if !ok {
		return [32]byte{}, fmt.Errorf("could not fetch hash for height: %v", height)
	}
	return bytesutil.ToBytes32(val), nil
}

func (m *mockPOWChainService) BlockTimeByHeight(_ context.Context, height *big.Int) (uint64, error) {
	h := int(height.Int64())
	return m.blockTimeByHeight[h], nil
}

func (m *mockPOWChainService) BlockNumberByTimestamp(_ context.Context, time uint64) (*big.Int, error) {
	return m.blockNumberByHeight[time], nil
}

func (m *mockPOWChainService) DepositRoot() [32]byte {
	root := []byte("depositroot")
	return bytesutil.ToBytes32(root)
}

func (m *mockPOWChainService) ChainStartDeposits() []*ethpb.Deposit {
	return []*ethpb.Deposit{}
}

func (m *mockPOWChainService) ChainStartDepositHashes() ([][]byte, error) {
	return [][]byte{}, nil
}

func (m *mockPOWChainService) ChainStartETH1Data() *ethpb.Eth1Data {
	return m.eth1Data
}

func TestWaitForChainStart_ContextClosed(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	beaconServer := &BeaconServer{
		ctx: ctx,
		powChainService: &faultyPOWChainService{
			chainStartFeed: new(event.Feed),
		},
		chainService: &mockClient.ChainService{},
	}
	exitRoutine := make(chan bool)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockStream := mockRPC.NewMockBeaconService_WaitForChainStartServer(ctrl)
	go func(tt *testing.T) {
		if err := beaconServer.WaitForChainStart(&ptypes.Empty{}, mockStream); !strings.Contains(err.Error(), closedContext) {
			tt.Errorf("Could not call RPC method: %v", err)
		}
		<-exitRoutine
	}(t)
	cancel()
	exitRoutine <- true
}

func TestWaitForChainStart_AlreadyStarted(t *testing.T) {
	beaconServer := &BeaconServer{
		ctx: context.Background(),
		powChainService: &mockPOWChainService{
			chainStartFeed: new(event.Feed),
		},
		chainService: &mockClient.ChainService{},
	}
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockStream := mockRPC.NewMockBeaconService_WaitForChainStartServer(ctrl)
	mockStream.EXPECT().Send(
		&pb.ChainStartResponse{
			Started:     true,
			GenesisTime: uint64(time.Unix(0, 0).Unix()),
		},
	).Return(nil)
	if err := beaconServer.WaitForChainStart(&ptypes.Empty{}, mockStream); err != nil {
		t.Errorf("Could not call RPC method: %v", err)
	}
}

func TestWaitForChainStart_NotStartedThenLogFired(t *testing.T) {
	hook := logTest.NewGlobal()
	beaconServer := &BeaconServer{
		ctx:            context.Background(),
		chainStartChan: make(chan time.Time, 1),
		powChainService: &faultyPOWChainService{
			chainStartFeed: new(event.Feed),
		},
		chainService: &mockClient.ChainService{},
	}
	exitRoutine := make(chan bool)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockStream := mockRPC.NewMockBeaconService_WaitForChainStartServer(ctrl)
	mockStream.EXPECT().Send(
		&pb.ChainStartResponse{
			Started:     true,
			GenesisTime: uint64(time.Unix(0, 0).Unix()),
		},
	).Return(nil)
	go func(tt *testing.T) {
		if err := beaconServer.WaitForChainStart(&ptypes.Empty{}, mockStream); err != nil {
			tt.Errorf("Could not call RPC method: %v", err)
		}
		<-exitRoutine
	}(t)
	beaconServer.chainStartChan <- time.Unix(0, 0)
	exitRoutine <- true
	testutil.AssertLogsContain(t, hook, "Sending ChainStart log and genesis time to connected validator clients")
}
