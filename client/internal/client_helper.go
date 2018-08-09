// Package internal provides a MockClient for testing.
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
	gethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/prysmaticlabs/prysm/client/contracts"
	shardparams "github.com/prysmaticlabs/prysm/client/params"
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
	depositFlag bool
	Backend     *backends.SimulatedBackend
	BlockNumber int64
}

// Account returns a mock account.
func (m *MockClient) Account() *accounts.Account {
	return &accounts.Account{Address: addr}
}

// SMCCaller returns a mock SMCCaller.
func (m *MockClient) SMCCaller() *contracts.SMCCaller {
	return &m.SMC.SMCCaller
}

// ChainReader returns a mock chain reader.
func (m *MockClient) ChainReader() ethereum.ChainReader {
	return nil
}

// SMCTransactor returns a mock SMCTransactor.
func (m *MockClient) SMCTransactor() *contracts.SMCTransactor {
	return &m.SMC.SMCTransactor
}

// SMCFilterer returns a mock SMCFilterer.
func (m *MockClient) SMCFilterer() *contracts.SMCFilterer {
	return &m.SMC.SMCFilterer
}

// WaitForTransaction waits for a transaction.
func (m *MockClient) WaitForTransaction(ctx context.Context, hash common.Hash, durationInSeconds time.Duration) error {
	m.CommitWithBlock()
	m.FastForward(1)
	return nil
}

// TransactionReceipt returns the transaction receipt from the mock backend.
func (m *MockClient) TransactionReceipt(hash common.Hash) (*gethTypes.Receipt, error) {
	return m.Backend.TransactionReceipt(context.Background(), hash)
}

// CreateTXOpts returns transaction opts with the value assigned.
func (m *MockClient) CreateTXOpts(value *big.Int) (*bind.TransactOpts, error) {
	txOpts := TransactOpts()
	txOpts.Value = value
	return txOpts, nil
}

// SetDepositFlag sets the deposit flag to the boolean value.
func (m *MockClient) SetDepositFlag(value bool) {
	m.depositFlag = value
}

// DepositFlag value.
func (m *MockClient) DepositFlag() bool {
	return m.depositFlag
}

// Sign does nothing?
func (m *MockClient) Sign(hash common.Hash) ([]byte, error) {
	return nil, nil
}

// GetShardCount returns constant shard count.
func (m *MockClient) GetShardCount() (int64, error) {
	return 100, nil
}

// CommitWithBlock commits a block to the backend.
func (m *MockClient) CommitWithBlock() {
	m.Backend.Commit()
	m.BlockNumber = m.BlockNumber + 1
}

// FastForward by iterating the mock backend p times.
func (m *MockClient) FastForward(p int) {
	for i := 0; i < p*int(shardparams.DefaultPeriodLength); i++ {
		m.CommitWithBlock()
	}
}

// SubscribeNewHead does nothing.
func (m *MockClient) SubscribeNewHead(ctx context.Context, ch chan<- *gethTypes.Header) (ethereum.Subscription, error) {
	return nil, nil
}

// BlockByNumber creates a block with a given number.
func (m *MockClient) BlockByNumber(ctx context.Context, number *big.Int) (*gethTypes.Block, error) {
	return gethTypes.NewBlockWithHeader(&gethTypes.Header{Number: big.NewInt(m.BlockNumber)}), nil
}

// TransactOpts Creates a new transaction options.
func TransactOpts() *bind.TransactOpts {
	return bind.NewKeyedTransactor(key)
}

// SetupMockClient sets up the mock client.
func SetupMockClient(t *testing.T) (*backends.SimulatedBackend, *contracts.SMC) {
	backend := backends.NewSimulatedBackend(core.GenesisAlloc{addr: {Balance: accountBalance}})
	_, _, SMC, err := contracts.DeploySMC(TransactOpts(), backend)
	if err != nil {
		t.Fatalf("Failed to deploy SMC contract: %v", err)
	}
	backend.Commit()
	return backend, SMC
}
