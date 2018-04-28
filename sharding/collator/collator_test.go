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
	accountBalance1001Eth, _ = new(big.Int).SetString("1001000000000000000000", 10)
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

func (m *mockClient) CreateTXOpts(value *big.Int) (*bind.TransactOpts, error) {
	txOpts := transactOpts()
	txOpts.Value = value
	return txOpts, nil
}

// Unused mockClient methods
func (m *mockClient) Start() error {
	m.t.Fatal("Start called")
	return nil
}
func (m *mockClient) Close() {
	m.t.Fatal("Close called")
}

// Helper/setup methods
// TODO: consider moving these to common sharding testing package as the collator and smc tests
// use them.
func transactOpts() *bind.TransactOpts {
	return bind.NewKeyedTransactor(key)
}
func setup() (*backends.SimulatedBackend, *contracts.SMC) {
	backend := backends.NewSimulatedBackend(core.GenesisAlloc{addr: {Balance: accountBalance1001Eth}})
	_, _, smc, _ := contracts.DeploySMC(transactOpts(), backend)
	backend.Commit()
	return backend, smc
}

func TestIsAccountInCollatorPool(t *testing.T) {
	backend, smc := setup()
	client := &mockClient{smc: smc, t: t}

	// address should not be in pool initially
	b, err := isAccountInCollatorPool(client)
	if err != nil {
		t.Fatal(err)
	}
	if b {
		t.Fatal("Account unexpectedly in collator pool")
	}

	txOpts := transactOpts()
	// deposit in collator pool, then it should return true
	txOpts.Value = sharding.NotaryDeposit
	if _, err := smc.RegisterNotary(txOpts); err != nil {
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

func TestJoinCollatorPool(t *testing.T) {
	backend, smc := setup()
	client := &mockClient{smc, t}

	// There should be no collators initially
	numCollators, err := smc.NotaryPoolLength(&bind.CallOpts{})
	if err != nil {
		t.Fatal(err)
	}
	if big.NewInt(0).Cmp(numCollators) != 0 {
		t.Fatalf("Unexpected number of collators. Got %d, wanted 0.", numCollators)
	}

	err = joinCollatorPool(client)
	if err != nil {
		t.Fatal(err)
	}
	backend.Commit()

	// Now there should be one collator
	numCollators, err = smc.NotaryPoolLength(&bind.CallOpts{})
	if err != nil {
		t.Fatal(err)
	}
	if big.NewInt(1).Cmp(numCollators) != 0 {
		t.Fatalf("Unexpected number of collators. Got %d, wanted 1.", numCollators)
	}
}
