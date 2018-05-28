package proposer

import (
	"math/big"
	"testing"

	"crypto/rand"

	"github.com/ethereum/go-ethereum"
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
	key, _            = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
	addr              = crypto.PubkeyToAddress(key.PublicKey)
	accountBalance, _ = new(big.Int).SetString("1001000000000000000000", 10)
)

// Mock client for testing proposer.
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

func (m *mockNode) CreateTXOpts(value *big.Int) (*bind.TransactOpts, error) {
	txOpts := transactOpts()
	txOpts.Value = value
	return txOpts, nil
}

func (m *mockNode) DepositFlagSet() bool {
	return m.DepositFlag
}

func (m *mockNode) DataDirFlag() string {
	return "/tmp/datadir"
}

func (m *mockNode) Sign(hash common.Hash) ([]byte, error) {
	return nil, nil
}

// Unused mockClient methods.
func (m *mockNode) Start() error {
	m.t.Fatal("Start called")
	return nil
}

func (m *mockNode) Close() {
	m.t.Fatal("Close called")
}

func transactOpts() *bind.TransactOpts {
	return bind.NewKeyedTransactor(key)
}

func setup() (*backends.SimulatedBackend, *contracts.SMC) {
	backend := backends.NewSimulatedBackend(core.GenesisAlloc{addr: {Balance: accountBalance}})
	_, _, smc, _ := contracts.DeploySMC(transactOpts(), backend)
	backend.Commit()
	return backend, smc
}

func TestCreateCollation(t *testing.T) {
	backend, smc := setup()
	node := &mockNode{smc: smc, t: t}
	var txs []*types.Transaction
	for i := 0; i < 10; i++ {
		data := make([]byte, 1024)
		rand.Read(data)
		txs = append(txs, types.NewTransaction(0, common.HexToAddress("0x0"),
			nil, 0, nil, data))
	}

	collation, err := createCollation(node, big.NewInt(0), big.NewInt(1), txs)
	if err != nil {
		t.Fatalf("Create collation failed: %v", err)
	}
	t.Log(collation.Header().Period())

	// fast forward to 2nd period
	for i := 0; i < 2*int(sharding.PeriodLength); i++ {
		backend.Commit()
	}

	// negative test case #1: create collation with period < currentPeriod
	collation, err = createCollation(node, big.NewInt(0), big.NewInt(3), txs)
	if err == nil {
		t.Fatalf("Create collation should have failed with invalid period")
	}
	// negative test case #2: create collation with shard > shardCount
	collation, err = createCollation(node, big.NewInt(101), big.NewInt(2), txs)
	if err == nil {
		t.Fatalf("Create collation should have failed with invalid shard number")
	}
	// negative test case #3, create collation with blob size > collationBodySizeLimit
	var badTxs []*types.Transaction
	for i := 0; i <= 1024; i++ {
		data := make([]byte, 1024)
		rand.Read(data)
		badTxs = append(badTxs, types.NewTransaction(0, common.HexToAddress("0x0"),
			nil, 0, nil, data))
	}
	collation, err = createCollation(node, big.NewInt(101), big.NewInt(2), badTxs)
	if err == nil {
		t.Fatalf("Create collation should have failed with Txs longer than collation body limit")
	}
}

func TestAddCollation(t *testing.T) {
}

func TestCheckCollation(t *testing.T) {
}
