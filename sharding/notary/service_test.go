package notary

import (
	"math/big"
	"testing"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
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

// Verifies that Notary implements the Actor interface.
var _ = sharding.Actor(&Notary{})

// Mock client for testing notary. Should this go into sharding/client/testing?
type smcClient struct {
	smc         *contracts.SMC
	depositFlag bool
	t           *testing.T
}

func (s *smcClient) Account() *accounts.Account {
	return &accounts.Account{Address: addr}
}

func (s *smcClient) SMCCaller() *contracts.SMCCaller {
	return &s.smc.SMCCaller
}

func (s *smcClient) ChainReader() ethereum.ChainReader {
	return nil
}

func (s *smcClient) Context() *cli.Context {
	return nil
}

func (s *smcClient) SMCTransactor() *contracts.SMCTransactor {
	return &s.smc.SMCTransactor
}

func (s *smcClient) SMCFilterer() *contracts.SMCFilterer {
	return &s.smc.SMCFilterer
}

func (s *smcClient) TransactionReceipt(hash common.Hash) (*types.Receipt, error) {
	return nil, nil
}

func (s *smcClient) CreateTXOpts(value *big.Int) (*bind.TransactOpts, error) {
	txOpts := transactOpts()
	txOpts.Value = value
	return txOpts, nil
}

func (s *smcClient) DepositFlag() bool {
	return s.depositFlag
}

func (s *smcClient) SetDepositFlag(deposit bool) {
	s.depositFlag = deposit
}

func (m *smcClient) Sign(hash common.Hash) ([]byte, error) {
	return nil, nil
}

// Unused mockClient methods.
func (m *smcClient) Start() error {
	m.t.Fatal("Start called")
	return nil
}

func (m *smcClient) Close() {
	m.t.Fatal("Close called")
}

func (s *smcClient) DataDirPath() string {
	return "/tmp/datadir"
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
	client := &smcClient{smc: smc, t: t}

	// address should not be in pool initially.
	b, err := isAccountInNotaryPool(client)
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
	client := &smcClient{smc: smc, depositFlag: false, t: t}
	// There should be no notary initially.
	numNotaries, err := smc.NotaryPoolLength(&bind.CallOpts{})
	if err != nil {
		t.Fatal(err)
	}
	if big.NewInt(0).Cmp(numNotaries) != 0 {
		t.Fatalf("Unexpected number of notaries. Got %d, wanted 0.", numNotaries)
	}

	client.SetDepositFlag(false)
	err = joinNotaryPool(client)
	if err == nil {
		t.Error("Joined notary pool while --deposit was not present")
	}

	client.SetDepositFlag(true)
	err = joinNotaryPool(client)
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

	// Trying to join while deposited should do nothing
	err = joinNotaryPool(client)
	if err != nil {
		t.Error(err)
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
