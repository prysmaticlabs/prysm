// Package testing provides useful mocks for an eth1 powchain
// service as needed by unit tests for the beacon node.
package testing

import (
	"context"
	"fmt"
	"math/big"
	"net/http/httptest"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	gethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/prysmaticlabs/prysm/v3/async/event"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/execution/types"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	"github.com/prysmaticlabs/prysm/v3/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
)

// Chain defines a properly functioning mock for the powchain service.
type Chain struct {
	ChainFeed         *event.Feed
	LatestBlockNumber *big.Int
	HashesByHeight    map[int][]byte
	TimesByHeight     map[int]uint64
	BlockNumberByTime map[uint64]*big.Int
	Eth1Data          *ethpb.Eth1Data
	GenesisEth1Block  *big.Int
	GenesisState      state.BeaconState
	CurrEndpoint      string
	CurrError         error
	Endpoints         []string
	Errors            []error
}

// GenesisTime represents a static past date - JAN 01 2000.
var GenesisTime = time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC).Unix()

// New creates a new mock chain with empty block info.
func New() *Chain {
	return &Chain{
		HashesByHeight:    make(map[int][]byte),
		TimesByHeight:     make(map[int]uint64),
		BlockNumberByTime: make(map[uint64]*big.Int),
	}
}

// GenesisExecutionChainInfo --
func (m *Chain) GenesisExecutionChainInfo() (uint64, *big.Int) {
	blk := m.GenesisEth1Block
	if blk == nil {
		blk = big.NewInt(GenesisTime)
	}
	return uint64(GenesisTime), blk
}

// BlockExists --
func (m *Chain) BlockExists(_ context.Context, hash common.Hash) (bool, *big.Int, error) {
	// Reverse the map of heights by hash.
	heightsByHash := make(map[[32]byte]int, len(m.HashesByHeight))
	for k, v := range m.HashesByHeight {
		h := bytesutil.ToBytes32(v)
		heightsByHash[h] = k
	}
	val, ok := heightsByHash[hash]
	if !ok {
		return false, nil, fmt.Errorf("could not fetch height for hash: %#x", hash)
	}
	return true, big.NewInt(int64(val)), nil
}

// BlockHashByHeight --
func (m *Chain) BlockHashByHeight(_ context.Context, height *big.Int) (common.Hash, error) {
	k := int(height.Int64())
	val, ok := m.HashesByHeight[k]
	if !ok {
		return [32]byte{}, fmt.Errorf("could not fetch hash for height: %v", height)
	}
	return bytesutil.ToBytes32(val), nil
}

// BlockTimeByHeight --
func (m *Chain) BlockTimeByHeight(_ context.Context, height *big.Int) (uint64, error) {
	h := int(height.Int64())
	return m.TimesByHeight[h], nil
}

// BlockByTimestamp --
func (m *Chain) BlockByTimestamp(_ context.Context, time uint64) (*types.HeaderInfo, error) {
	var chosenTime uint64
	var chosenNumber *big.Int
	for t, num := range m.BlockNumberByTime {
		if t > chosenTime && t <= time {
			chosenNumber = num
			chosenTime = t
		}
	}
	return &types.HeaderInfo{Number: chosenNumber, Time: chosenTime}, nil
}

// ChainStartEth1Data --
func (m *Chain) ChainStartEth1Data() *ethpb.Eth1Data {
	return m.Eth1Data
}

// PreGenesisState --
func (m *Chain) PreGenesisState() state.BeaconState {
	return m.GenesisState
}

// ClearPreGenesisData --
func (*Chain) ClearPreGenesisData() {
	// no-op
}

func (*Chain) ExecutionClientConnected() bool {
	return true
}

func (m *Chain) ExecutionClientEndpoint() string {
	return m.CurrEndpoint
}

func (m *Chain) ExecutionClientConnectionErr() error {
	return m.CurrError
}

func (m *Chain) ETH1Endpoints() []string {
	return m.Endpoints
}

func (m *Chain) ETH1ConnectionErrors() []error {
	return m.Errors
}

// RPCClient defines the mock rpc client.
type RPCClient struct {
	Backend *backends.SimulatedBackend
}

func (*RPCClient) Close() {}

func (*RPCClient) CallContext(_ context.Context, _ interface{}, _ string, _ ...interface{}) error {
	return nil
}

// BatchCall --
func (r *RPCClient) BatchCall(b []rpc.BatchElem) error {
	if r.Backend == nil {
		return nil
	}

	for _, e := range b {
		num, err := hexutil.DecodeBig(e.Args[0].(string))
		if err != nil {
			return err
		}
		h, err := r.Backend.HeaderByNumber(context.Background(), num)
		if err != nil {
			return err
		}
		*e.Result.(*gethTypes.Header) = *h

	}
	return nil
}

// InsertBlock adds provided block info into the chain.
func (m *Chain) InsertBlock(height int, time uint64, hash []byte) *Chain {
	m.HashesByHeight[height] = hash
	m.TimesByHeight[height] = time
	m.BlockNumberByTime[time] = big.NewInt(int64(height))
	return m
}

func SetupRPCServer() (*rpc.Server, string, error) {
	srv := rpc.NewServer()
	if err := srv.RegisterName("eth", &testETHRPC{}); err != nil {
		return nil, "", err
	}
	if err := srv.RegisterName("net", &testETHRPC{}); err != nil {
		return nil, "", err
	}
	hs := httptest.NewUnstartedServer(srv)
	hs.Start()
	return srv, hs.URL, nil
}

type testETHRPC struct{}

func (*testETHRPC) NoArgsRets() {}

func (*testETHRPC) ChainId(_ context.Context) *hexutil.Big {
	return (*hexutil.Big)(big.NewInt(int64(params.BeaconConfig().DepositChainID)))
}

func (*testETHRPC) Version(_ context.Context) string {
	return fmt.Sprintf("%d", params.BeaconConfig().DepositNetworkID)
}
