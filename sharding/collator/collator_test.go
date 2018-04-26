package notary

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

// Mock client for testing notary. Should this go into sharding/client/testing?
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
// TODO: consider moving these to common sharding testing package as the notary and smc tests
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

func TestIsAccountInNotaryPool(t *testing.T) {
	backend, smc := setup()
	client := &mockClient{smc: smc, t: t}

	// address should not be in pool initially
	b, err := isAccountInNotaryPool(client)
	if err != nil {
		t.Fatal(err)
	}
	if b {
		t.Fatal("Account unexpectedly in notary pool")
	}

	txOpts := transactOpts()
	// deposit in notary pool, then it should return true
	txOpts.Value = sharding.NotaryDeposit
	if _, err := smc.Deposit(txOpts); err != nil {
		t.Fatalf("Failed to deposit: %v", err)
	}
	backend.Commit()
	b, err = isAccountInNotaryPool(client)
	if err != nil {
		t.Fatal(err)
	}
	if !b {
		t.Fatal("Account not in notary pool when expected to be")
	}
}

func TestJoinNotaryPool(t *testing.T) {
	backend, smc := setup()
	client := &mockClient{smc, t}

	// There should be no notarys initially
	numNotaries, err := smc.NumNotaries(&bind.CallOpts{})
	if err != nil {
		t.Fatal(err)
	}
	if big.NewInt(0).Cmp(numNotaries) != 0 {
		t.Fatalf("Unexpected number of notarys. Got %d, wanted 0.", numNotaries)
	}

	err = joinNotaryPool(client)
	if err != nil {
		t.Fatal(err)
	}
	backend.Commit()

	// Now there should be one notary
	numNotaries, err = smc.NumNotaries(&bind.CallOpts{})
	if err != nil {
		t.Fatal(err)
	}
	if big.NewInt(1).Cmp(numNotaries) != 0 {
		t.Fatalf("Unexpected number of notarys. Got %d, wanted 1.", numNotaries)
	}
}
