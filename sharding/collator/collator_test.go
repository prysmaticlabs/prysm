package collator

import (
	"math/big"
	"testing"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/sharding"
	"github.com/ethereum/go-ethereum/sharding/contracts"
)

var (
	key, _                   = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
	addr                     = crypto.PubkeyToAddress(key.PublicKey)
	accountBalance1001Eth, _ = new(big.Int).SetString("1000000000000000000001", 10)
)

// Mock client for testing collator. Should this go into sharding/client/testing?
type mockClient struct {
	smc *contracts.SMC
	t   *testing.T
}

func (m *mockClient) Account() *accounts.Account {
	return &accounts.Account{Address: addr}
}

func (m *mockClient) SMCCaller() *contracts.SMCCaller {
	return &m.smc.SMCCaller
}

func (m *mockClient) ChainReader() ethereum.ChainReader {
	m.t.Fatal("ChainReader not implemented")
	return nil
}

func (m *mockClient) SMCTransactor() *contracts.SMCTransactor {
	return &m.smc.SMCTransactor
}

func (m *mockClient) CreateTXOps(value *big.Int) (*bind.TransactOpts, error) {
	m.t.Fatal("CreateTXOps not implemented")
	return nil, nil
}

// Unused mockClient methods
func (m *mockClient) Start() error {
	m.t.Fatal("Start called")
	return nil
}
func (m *mockClient) Close() {
	m.t.Fatal("Close called")
}

func TestIsAccountInCollatorPool(t *testing.T) {
	// Test setup (should this go to sharding/client/testing?)
	backend := backends.NewSimulatedBackend(core.GenesisAlloc{addr: {Balance: accountBalance1001Eth}})
	transactOpts := bind.NewKeyedTransactor(key)
	_, _, smc, _ := contracts.DeploySMC(transactOpts, backend)
	backend.Commit()

	client := &mockClient{smc: smc, t: t}

	// address should not be in pool initially
	b, err := isAccountInCollatorPool(client)
	if err != nil {
		t.Fatal(err)
	}
	if b {
		t.Fatal("Account unexpectedly in collator pool")
	}

	// deposit in collator pool, then it should return true
	transactOpts.Value = sharding.CollatorDeposit
	if _, err := smc.Deposit(transactOpts); err != nil {
		t.Fatalf("Failed to deposit: %v", err)
	}
	backend.Commit()
	b, err = isAccountInCollatorPool(client)
	if err != nil {
		t.Fatal(err)
	}
	if !b {
		t.Fatal("Account not in collator pool when expected to be")
	}
}
