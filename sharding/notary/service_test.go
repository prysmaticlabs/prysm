package notary

import (
	"context"
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
	"github.com/ethereum/go-ethereum/sharding/params"
	cli "gopkg.in/urfave/cli.v1"
)

var (
	key, _                   = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
	addr                     = crypto.PubkeyToAddress(key.PublicKey)
	accountBalance1001Eth, _ = new(big.Int).SetString("1001000000000000000000", 10)
)

// Verifies that Notary implements the Actor interface.
var _ = sharding.Actor(&Notary{})

type newBackend struct {
	backend     *backends.SimulatedBackend
	blocknumber int64
}

// Mock client for testing notary. Should this go into sharding/client/testing?
type smcClient struct {
	smc         *contracts.SMC
	depositFlag bool
	t           *testing.T
	backend     *newBackend
}

func (s *smcClient) SubscribeNewHead(ctx context.Context, ch chan<- *types.Header) (ethereum.Subscription, error) {
	return nil, nil
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

func (s *smcClient) WaitForTransaction(hash common.Hash, duration int64) error {
	s.backend.CommitWithBlock()
	return nil
}

func (s *smcClient) TransactionReceipt(hash common.Hash) (*types.Receipt, error) {
	return s.backend.backend.TransactionReceipt(context.Background(), hash)
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

func (b *newBackend) CommitWithBlock() {
	b.backend.Commit()
	b.blocknumber++
}

func (s *smcClient) GetShardCount() (int64, error) {
	return 100, nil
}

// Helper/setup methods.
// TODO: consider moving these to common sharding testing package as the notary and smc tests
// use them.
func transactOpts() *bind.TransactOpts {
	return bind.NewKeyedTransactor(key)
}

func setup() (*newBackend, *contracts.SMC) {
	backend := newBackend{backends.NewSimulatedBackend(core.GenesisAlloc{addr: {Balance: accountBalance1001Eth}}), 0}
	_, _, smc, _ := contracts.DeploySMC(transactOpts(), backend.backend)
	backend.CommitWithBlock()
	return &backend, smc
}

func TestHasAccountBeenDeregistered(t *testing.T) {
	backend, smc := setup()
	client := &smcClient{smc: smc, t: t, backend: backend}

	client.SetDepositFlag(true)
	err := joinNotaryPool(client)
	if err != nil {
		t.Error(err)
	}
	backend.CommitWithBlock()

	err = leaveNotaryPool(client)
	backend.CommitWithBlock()

	if err != nil {
		t.Error(err)
	}

	dreg, err := hasAccountBeenDeregistered(client)

	if err != nil {
		t.Error(err)
	}

	if !dreg {
		t.Error("account unable to be deregistered from notary pool")

	}

}

func TestIsLockupOver(t *testing.T) {
	backend, smc := setup()
	client := &smcClient{smc: smc, t: t, backend: backend}

	client.SetDepositFlag(true)
	err := joinNotaryPool(client)
	if err != nil {
		t.Error(err)
	}
	backend.CommitWithBlock()

	err = leaveNotaryPool(client)
	backend.CommitWithBlock()

	if err != nil {
		t.Error(err)
	}

	err = releaseNotary(client, big.NewInt(backend.blocknumber))

	if err != nil {
		t.Error(err)
	}

	lockup, err := isLockUpOver(client, big.NewInt(backend.blocknumber))

	if err != nil {
		t.Error(err)
	}

	if !lockup {
		t.Error("lockup period is not over despite account being relased from registry")
	}

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
		t.Fatal("account unexpectedly in notary pool")
	}

	txOpts := transactOpts()
	// deposit in notary pool, then it should return true.
	txOpts.Value = params.DefaultConfig.NotaryDeposit
	if _, err := smc.RegisterNotary(txOpts); err != nil {
		t.Fatalf("Failed to deposit: %v", err)
	}
	backend.CommitWithBlock()
	b, err = isAccountInNotaryPool(client)
	if err != nil {
		t.Error(err)
	}
	if !b {
		t.Error("account not in notary pool when expected to be")
	}
}

func TestJoinNotaryPool(t *testing.T) {
	backend, smc := setup()
	client := &smcClient{smc: smc, depositFlag: false, t: t, backend: backend}
	// There should be no notary initially.
	numNotaries, err := smc.NotaryPoolLength(&bind.CallOpts{})
	if err != nil {
		t.Error(err)
	}
	if big.NewInt(0).Cmp(numNotaries) != 0 {
		t.Errorf("unexpected number of notaries. Got %d, wanted 0.", numNotaries)
	}

	client.SetDepositFlag(false)
	err = joinNotaryPool(params.DefaultConfig, client)
	if err == nil {
		t.Error("joined notary pool while --deposit was not present")
	}

	client.SetDepositFlag(true)
	err = joinNotaryPool(params.DefaultConfig, client)
	if err != nil {
		t.Error(err)
	}
	backend.CommitWithBlock()

	// Now there should be one notary.
	numNotaries, err = smc.NotaryPoolLength(&bind.CallOpts{})
	if err != nil {
		t.Error(err)
	}
	if big.NewInt(1).Cmp(numNotaries) != 0 {
		t.Errorf("unexpected number of notaries. Got %d, wanted 1", numNotaries)
	}

	// Trying to join while deposited should do nothing
	err = joinNotaryPool(params.DefaultConfig, client)
	if err != nil {
		t.Error(err)
	}
	backend.CommitWithBlock()

	numNotaries, err = smc.NotaryPoolLength(&bind.CallOpts{})
	if err != nil {
		t.Error(err)
	}
	if big.NewInt(1).Cmp(numNotaries) != 0 {
		t.Errorf("unexpected number of notaries. Got %d, wanted 1", numNotaries)
	}

}

func TestLeaveNotaryPool(t *testing.T) {
	backend, smc := setup()
	client := &smcClient{smc: smc, t: t, depositFlag: true, backend: backend}

	// Test Leaving Notary Pool Before Joining it

	err := leaveNotaryPool(client)
	if err == nil {
		t.Error("able to leave notary pool despite having not joined it")
	}

	// Roundtrip Test , Join and leave pool

	err = joinNotaryPool(client)
	if err != nil {
		t.Error(err)
	}
	backend.CommitWithBlock()

	// Now there should be one notary.
	numNotaries, err := smc.NotaryPoolLength(&bind.CallOpts{})
	if err != nil {
		t.Error(err)
	}
	if big.NewInt(1).Cmp(numNotaries) != 0 {
		t.Errorf("unexpected number of notaries. Got %d, wanted 1", numNotaries)
	}

	err = leaveNotaryPool(client)
	if err != nil {
		t.Error(err)
	}
	backend.CommitWithBlock()

	numNotaries, err = smc.NotaryPoolLength(&bind.CallOpts{})
	if err != nil {
		t.Error(err)
	}
	if big.NewInt(0).Cmp(numNotaries) != 0 {
		t.Errorf("unexpected number of notaries. Got %d, wanted 0", numNotaries)
	}

}

func TestReleaseNotary(t *testing.T) {
	backend, smc := setup()
	client := &smcClient{smc: smc, t: t, depositFlag: true, backend: backend}

	// Test Release Notary Before Joining it

	err := releaseNotary(client, big.NewInt(0))
	if err == nil {
		t.Error("released From notary despite never joining pool")
	}

	// Roundtrip Test , Join and leave pool and release Notary

	err = joinNotaryPool(client)
	if err != nil {
		t.Error(err)
	}
	backend.CommitWithBlock()

	err = leaveNotaryPool(client)
	backend.CommitWithBlock()

	if err != nil {

		t.Error(err)
	}

	balance, err := backend.backend.BalanceAt(context.Background(), addr, nil)
	if err != nil {
		t.Error("unable to retrieve balance")
	}

	err = releaseNotary(client, big.NewInt(backend.blocknumber))
	backend.CommitWithBlock()

	if err != nil {

		t.Fatal(err)
	}

	nreg, err := smc.NotaryRegistry(&bind.CallOpts{}, addr)
	if err != nil {
		t.Fatal(err)
	}
	if nreg.Deposited {
		t.Error("Unable to release Notary and deposit money back")
	}

	newbalance, err := backend.backend.BalanceAt(context.Background(), addr, nil)
	if err != nil {
		t.Error("unable to retrieve balance")
	}

	if balance.Cmp(newbalance) != 1 {
		t.Errorf("Deposit was not returned, balance is currently:", newbalance)
	}

}

func TestSubmitVote(t *testing.T) {
	backend, smc := setup()
	client := &smcClient{smc: smc, t: t, depositFlag: true, backend: backend}

	err := joinNotaryPool(client)
	if err != nil {
		t.Error(err)
	}
	backend.CommitWithBlock()

	// TODO: vote Test is unable to continue until chainreader is implemented as
	// finding the current period requires knowing the latest blocknumber on the
	// chain #164
}
