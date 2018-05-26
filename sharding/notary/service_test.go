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
	cli "gopkg.in/urfave/cli.v1"
)

var (
	key, _                   = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
	addr                     = crypto.PubkeyToAddress(key.PublicKey)
	accountBalance1001Eth, _ = new(big.Int).SetString("1001000000000000000000", 10)
)

// Mock client for testing notary. Should this go into sharding/client/testing?
type mockNode struct {
	smc         *contracts.SMC
	t           *testing.T
	DepositFlag bool
}

func (m *mockNode) Account() *accounts.Account {
	return &accounts.Account{Address: addr}
}

func (m *mockNode) SMCCaller() *contracts.SMCCaller {
	return &m.smc.SMCCaller
}

func (m *mockNode) ChainReader() ethereum.ChainReader {
	m.t.Fatal("ChainReader not implemented")
	return nil
}

func (m *mockNode) Context() *cli.Context {
	return nil
}

func (m *mockNode) Register(s sharding.ServiceConstructor) error {
	return nil
}

func (m *mockNode) SMCTransactor() *contracts.SMCTransactor {
	return &m.smc.SMCTransactor
}

func (m *mockNode) SMCFilterer() *contracts.SMCFilterer {
	return &m.smc.SMCFilterer
}

func (m *mockNode) CreateTXOpts(value *big.Int) (*bind.TransactOpts, error) {
	txOpts := transactOpts()
	txOpts.Value = value
	return txOpts, nil
}

func (m *mockNode) DepositFlagSet() bool {
	return m.DepositFlag
}

// Unused mockClient methods.
func (m *mockNode) Start() error {
	m.t.Fatal("Start called")
	return nil
}

func (m *mockNode) Close() {
	m.t.Fatal("Close called")
}

// Helper/setup methods.
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
	node := &mockNode{smc: smc, t: t}

	// address should not be in pool initially.
	b, err := isAccountInNotaryPool(node)
	if err != nil {
		t.Fatal(err)
	}
	if b {
		t.Fatal("Account unexpectedly in notary pool")
	}

	txOpts := transactOpts()
	// deposit in notary pool, then it should return true.
	txOpts.Value = sharding.NotaryDeposit
	if _, err := smc.RegisterNotary(txOpts); err != nil {
		t.Fatalf("Failed to deposit: %v", err)
	}
	backend.Commit()
	b, err = isAccountInNotaryPool(node)
	if err != nil {
		t.Fatal(err)
	}
	if !b {
		t.Fatal("Account not in notary pool when expected to be")
	}
}

func TestJoinNotaryPool(t *testing.T) {
	backend, smc := setup()
	node := &mockNode{smc: smc, t: t, DepositFlag: false}
	// There should be no notary initially.
	numNotaries, err := smc.NotaryPoolLength(&bind.CallOpts{})
	if err != nil {
		t.Fatal(err)
	}
	if big.NewInt(0).Cmp(numNotaries) != 0 {
		t.Fatalf("Unexpected number of notaries. Got %d, wanted 0.", numNotaries)
	}

	node.DepositFlag = false
	err = joinNotaryPool(node)
	if err == nil {
		t.Error("Joined notary pool while --deposit was not present")
	}

	node.DepositFlag = true
	err = joinNotaryPool(node)
	if err != nil {
		t.Fatal(err)
	}
	backend.Commit()

	// Now there should be one notary.
	numNotaries, err = smc.NotaryPoolLength(&bind.CallOpts{})
	if err != nil {
		t.Fatal(err)
	}
	if big.NewInt(1).Cmp(numNotaries) != 0 {
		t.Fatalf("Unexpected number of notaries. Got %d, wanted 1.", numNotaries)
	}

	// Trying to join while deposited should return an error
	err = joinNotaryPool(node)
	if err == nil {
		t.Error("Tried to join Notary Pool while already deposited")
	}
	backend.Commit()

	numNotaries, err = smc.NotaryPoolLength(&bind.CallOpts{})
	if err != nil {
		t.Fatal(err)
	}
	if big.NewInt(1).Cmp(numNotaries) != 0 {
		t.Fatalf("Unexpected number of notaries. Got %d, wanted 1.", numNotaries)
	}

}

func TestLeaveNotaryPool(t *testing.T) {
	backend, smc := setup()
	node := &mockNode{smc: smc, t: t, DepositFlag: true}

	err := joinNotaryPool(node)
	if err != nil {
		t.Fatal(err)
	}

	backend.Commit()

	err = leaveNotaryPool(node)
	backend.Commit()

	filterOps := &bind.FilterOpts{0, nil, nil}
	events, err := node.SMCFilterer().FilterNotaryDeregistered(filterOps)
	yes := events.Next()
	if err == nil {
		t.Errorf("Unable to filter events: %v\n%v\n %v", yes, err, events.Event)
	}

	if err != nil {

		t.Fatal(err)
	}

}
