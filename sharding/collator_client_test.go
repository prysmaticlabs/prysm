package sharding

import (
	"context"
	"flag"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/sharding/contracts"

	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/rpc"
	cli "gopkg.in/urfave/cli.v1"
)

// FakeEthService based on implementation of internal/ethapi.Client
type FakeEthService struct{}

// eth_getCode
func (s *FakeEthService) GetCode(ctx context.Context, address common.Address, blockNr rpc.BlockNumber) (string, error) {
	return contracts.SMCBin, nil
}

func (s *FakeEthService) GasPrice(ctx context.Context) (hexutil.Big, error) {
	b := big.NewInt(1000)
	return hexutil.Big(*b), nil
}

func (s *FakeEthService) EstimateGas(ctx context.Context, msg interface{}) (hexutil.Uint64, error) {
	h := hexutil.Uint64(uint64(1000000))
	return h, nil
}

func (s *FakeEthService) GetTransactionCount(ctx context.Context, address common.Address, blockNr rpc.BlockNumber) (hexutil.Uint64, error) {
	return hexutil.Uint64(uint64(1)), nil
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

type FakeNetworkService struct{}

func (s *FakeNetworkService) Version() (string, error) {
	return "100", nil
}

type FakeNewHeadsService struct{}

func (s *FakeNewHeadsService) NewHeads() {

}

// TODO: Use a simulated backend rather than starting a fake node.
func newTestServer(endpoint string) (*rpc.Server, error) {
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
	if err := server.RegisterName("net", new(FakeNetworkService)); err != nil {
		return nil, err
	}
	if err := server.RegisterName("newHeads", new(FakeNewHeadsService)); err != nil {
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

// TODO(prestonvanloon): Fix this test.
// func TestShardingClient(t *testing.T) {
// 	endpoint := path.Join(os.TempDir(), fmt.Sprintf("go-ethereum-test-ipc-%d-%d", os.Getpid(), rand.Int63()))
// 	server, err := newTestServer(endpoint)
// 	if err != nil {
// 		t.Fatalf("Failed to create a test server: %v", err)
// 	}
// 	defer server.Stop()

// 	ctx := createContext()
// 	if err := ctx.GlobalSet(utils.DataDirFlag.Name, endpoint); err != nil {
// 		t.Fatalf("Failed to set global variable for flag %s. Error: %v", utils.DataDirFlag.Name, err)
// 	}

// 	c := MakeCollatorClient(ctx)

// 	if err := c.Start(); err != nil {
// 		t.Errorf("Failed to start server: %v", err)
// 	}
// }
