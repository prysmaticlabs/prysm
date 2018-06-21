package internal

import (
	"context"
	"math/big"
	"testing"
	"time"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/sharding/contracts"
)

var (
	key, _            = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
	addr              = crypto.PubkeyToAddress(key.PublicKey)
	accountBalance, _ = new(big.Int).SetString("1001000000000000000000", 10)
)

// MockClient for testing proposer.
type MockClient struct {
	SMC         *contracts.SMC
	T           *testing.T
	DepositFlag bool
	Backend     *backends.SimulatedBackend
}

func (m *MockClient) Account() *accounts.Account {
	return &accounts.Account{Address: addr}
}

func (m *MockClient) SMCCaller() *contracts.SMCCaller {
	return &m.SMC.SMCCaller
}

func (m *MockClient) ChainReader() ethereum.ChainReader {
	return nil
}

func (m *MockClient) SMCTransactor() *contracts.SMCTransactor {
	return &m.SMC.SMCTransactor
}

func (m *MockClient) SMCFilterer() *contracts.SMCFilterer {
	return &m.SMC.SMCFilterer
}

func (m *MockClient) WaitForTransaction(ctx context.Context, hash common.Hash, durationInSeconds time.Duration) error {
	return nil
}

func (m *MockClient) TransactionReceipt(hash common.Hash) (*types.Receipt, error) {
	return nil, nil
}

func (m *MockClient) CreateTXOpts(value *big.Int) (*bind.TransactOpts, error) {
	txOpts := TransactOpts()
	txOpts.Value = value
	return txOpts, nil
}

func (m *MockClient) Sign(hash common.Hash) ([]byte, error) {
	return nil, nil
}

func (m *MockClient) GetShardCount() (int64, error) {
	return 100, nil
}

func TransactOpts() *bind.TransactOpts {
	return bind.NewKeyedTransactor(key)
}

func SetupMockNode(t *testing.T) (*backends.SimulatedBackend, *contracts.SMC) {
	backend := backends.NewSimulatedBackend(core.GenesisAlloc{addr: {Balance: accountBalance}})
	_, _, SMC, err := contracts.DeploySMC(TransactOpts(), backend)
	if err != nil {
		t.Fatalf("Failed to deploy SMC contract: %v", err)
	}
	backend.Commit()
	return backend, SMC
}
