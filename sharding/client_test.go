package sharding

import (
	"context"
	"flag"
	"fmt"
	"math/big"
	"math/rand"
	"os"
	"sync"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/rpc"
	cli "gopkg.in/urfave/cli.v1"
)

// FakeEthService based on implementation of internal/ethapi
type FakeEthService struct {
	mu sync.Mutex

	getCodeResp hexutil.Bytes
	getCodeErr  error
}

// eth_getCode
func (s *FakeEthService) GetCode(ctx context.Context, address common.Address, blockNr rpc.BlockNumber) (hexutil.Bytes, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.getCodeResp, s.getCodeErr
}

// Set return values for eth_getCode
func (s *FakeEthService) SetGetCode(resp hexutil.Bytes, err error) {
	s.mu.Lock()
	s.getCodeResp = resp
	s.getCodeErr = err
	s.mu.Unlock()
}

func (s *FakeEthService) GasPrice(ctx context.Context) (*big.Int, error) {
	return big.NewInt(10000), nil
}

func (s *FakeEthService) GetTransactionCount(ctx context.Context, address common.Address, blockNr rpc.BlockNumber) (*hexutil.Uint64, error) {
	return nil, nil
}

func (s *FakeEthService) SendRawTransaction(ctx context.Context, encodedTx hexutil.Bytes) (common.Hash, error) {
	return common.Hash{}, nil
}

func (s *FakeEthService) GetTransactionReceipt(hash common.Hash) (*types.Receipt, error) {
	return &types.Receipt{
		ContractAddress: common.StringToAddress("0x1"),
		Logs:            []*types.Log{},
	}, nil
}

func (s *FakeEthService) GetTransactionByHash(hash common.Hash) (tx *types.Transaction, isPending bool, err error) {
	return nil, false, nil
}

func newTestServer(endpoint string) (*rpc.Server, error) {
	// Create datadir.
	if err := os.Mkdir(endpoint, 0777); err != nil {
		return nil, err
	}

	// Create a default account without password.
	scryptN, scryptP, keydir, err := (&node.Config{DataDir: endpoint}).AccountConfig()
	if err != nil {
		return nil, err
	}
	if _, err := keystore.StoreKey(keydir, "" /*password*/, scryptN, scryptP); err != nil {
		return nil, err
	}

	// Create server and register eth service with FakeEthService
	server := rpc.NewServer()
	if err := server.RegisterName("eth", new(FakeEthService)); err != nil {
		return nil, err
	}
	l, err := rpc.CreateIPCListener(endpoint + "/geth.ipc")
	if err != nil {
		return nil, err
	}
	go server.ServeListener(l)

	return server, nil
}

func createContext() *cli.Context {
	set := flag.NewFlagSet("test", 0)
	set.String(utils.DataDirFlag.Name, "", "")
	return cli.NewContext(nil, set, nil)
}

func TestShardingClient(t *testing.T) {
	endpoint := fmt.Sprintf("%s/go-ethereum-test-ipc-%d-%d", os.TempDir(), os.Getpid(), rand.Int63())
	server, err := newTestServer(endpoint)
	if err != nil {
		t.Fatalf("Failed to create a test server: %v", err)
	}
	defer server.Stop()

	ctx := createContext()
	if err := ctx.GlobalSet(utils.DataDirFlag.Name, endpoint); err != nil {
		t.Fatalf("Failed to set global variable for flag %s. Error: %v", utils.DataDirFlag.Name, err)
	}

	c := MakeShardingClient(ctx)

	if err := c.Start(); err != nil {
		t.Errorf("Failed to start server: %v", err)
	}
}
