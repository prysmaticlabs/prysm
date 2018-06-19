package proposer

import (
	"context"
	"crypto/rand"
	"math/big"
	"testing"

	"fmt"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/sharding/contracts"
	internal "github.com/ethereum/go-ethereum/sharding/internal"
	"github.com/ethereum/go-ethereum/sharding/mainchain"
	"github.com/ethereum/go-ethereum/sharding/params"
	"github.com/ethereum/go-ethereum/sharding/txpool"
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
	depositFlag bool
	backend     *backends.SimulatedBackend
}

func (m *mockNode) Account() *accounts.Account {
	return &accounts.Account{Address: addr}
}

func (m *mockNode) SMCCaller() *contracts.SMCCaller {
	return &m.smc.SMCCaller
}

func (m *mockNode) ChainReader() ethereum.ChainReader {
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

func (m *mockNode) Sign(hash common.Hash) ([]byte, error) {
	return nil, nil
}

func (m *mockNode) GetShardCount() (int64, error) {
	return 100, nil
}

type mockReader struct{}

func (mockReader) BlockByNumber(ctx context.Context, number *big.Int) (*types.Block, error) {
	header := types.Header{Number: big.NewInt(1)}
	return types.NewBlockWithHeader(&header), nil
}

func (mockReader) SubscribeNewHead(ctx context.Context, ch chan<- *types.Header) (ethereum.Subscription, error) {
	return nil, nil
}

type mockContractCaller struct {
	smcCaller *contracts.SMCCaller
}

func (m mockContractCaller) SMCCaller() *contracts.SMCCaller {
	return m.smcCaller
}

func (mockContractCaller) GetShardCount() (int64, error) {
	return 1, nil
}

type mockSigner struct{}

func (mockSigner) Sign(hash common.Hash) ([]byte, error) {
	return make([]byte, 0), nil
}

type mockTransactor struct {
	transactor *contracts.SMCTransactor
}

func (m mockTransactor) SMCTransactor() *contracts.SMCTransactor {
	return m.transactor
}

func (mockTransactor) CreateTXOpts(value *big.Int) (*bind.TransactOpts, error) {
	txOpts := transactOpts()
	txOpts.Value = value
	txOpts.GasPrice = big.NewInt(100000)
	return txOpts, nil
}

func transactOpts() *bind.TransactOpts {
	return bind.NewKeyedTransactor(key)
}

func setup(t *testing.T) (*backends.SimulatedBackend, *contracts.SMC) {
	backend := backends.NewSimulatedBackend(core.GenesisAlloc{addr: {Balance: accountBalance}})
	_, _, smc, err := contracts.DeploySMC(transactOpts(), backend)
	if err != nil {
		t.Fatalf("Failed to deploy SMC contract: %v", err)
	}
	backend.Commit()
	return backend, smc
}

func TestProposerServiceStartStop(t *testing.T) {
	txpool, err := txpool.NewTXPool(nil)
	if err != nil {
		t.Fatalf("Failed to initialize txfeed: %v", err)
	}

	shardID := 1
	//client := &mainchain.SMCClient{}
	client, err := mainchain.NewSMCClient("", "", true, "")
	if err != nil {
		t.Fatalf("Failed to instantiate client: %v", err)
	}

	proposer, err := NewProposer(&params.Config{}, client, nil, txpool, nil, shardID)
	if err != nil {
		t.Fatalf("Failed to initialize proposer: %v", err)
	}

	h := internal.NewLogHandler(t)
	log.Root().SetHandler(h)

	proposer.Start()

	if proposer.Stop() != nil {
		t.Fatalf("Failed to stop proposer: %v", err)
	}

	if proposer.ctx.Err() == nil {
		t.Fatal("Failed to close context")
	}

	h.VerifyLogMsg(fmt.Sprintf("Starting proposer service in shard %d", shardID))
	h.VerifyLogMsg(fmt.Sprintf("Stopping proposer service in shard %d", shardID))
}

func TestProposeCollationsDone(t *testing.T) {
	h := internal.NewLogHandler(t)
	log.Root().SetHandler(h)

	pool, err := txpool.NewTXPool(nil)
	if err != nil {
		t.Fatalf("Failed to initialize txfeed: %v", err)
	}

	shardID := 1
	client := &mainchain.SMCClient{}
	p, err := NewProposer(&params.Config{}, client, nil, pool, nil, shardID)

	done := make(chan struct{})
	subErr := make(chan error)
	requests := make(chan *types.Transaction)

	go p.proposeCollations(done, subErr, requests, &accounts.Account{}, mockReader{}, mockContractCaller{}, mockSigner{}, mockTransactor{})

	done <- struct{}{}

	h.VerifyLogMsg("Proposer context closed, exiting goroutine")
}

func TestProposeCollationsSubErr(t *testing.T) {
	h := internal.NewLogHandler(t)
	log.Root().SetHandler(h)

	pool, err := txpool.NewTXPool(nil)
	if err != nil {
		t.Fatalf("Failed to initialize txfeed: %v", err)
	}

	shardID := 1
	client := &mainchain.SMCClient{}
	p, err := NewProposer(&params.Config{}, client, nil, pool, nil, shardID)

	done := make(chan struct{})
	subErr := make(chan error)
	requests := make(chan *types.Transaction)

	go p.proposeCollations(done, subErr, requests, &accounts.Account{}, mockReader{}, mockContractCaller{}, mockSigner{}, mockTransactor{})

	subErr <- nil

	h.VerifyLogMsg("Transaction feed subscriber closed")
}

func TestProposeCollationsRequests(t *testing.T) {
	h := internal.NewLogHandler(t)
	log.Root().SetHandler(h)

	backend, smc := setup(t)
	node := &mockNode{smc: smc, t: t, backend: backend}

	pool, err := txpool.NewTXPool(nil)
	if err != nil {
		t.Fatalf("Failed to initialize txfeed: %v", err)
	}

	shardID := 1
	client := &mainchain.SMCClient{}
	p, err := NewProposer(&params.Config{PeriodLength: 1}, client, nil, pool, nil, shardID)

	done := make(chan struct{})
	subErr := make(chan error)
	requests := make(chan *types.Transaction)

	go p.proposeCollations(done, subErr, requests, &accounts.Account{}, mockReader{}, mockContractCaller{smcCaller: &smc.SMCCaller}, mockSigner{}, node)

	data := make([]byte, 1024)
	rand.Read(data)
	tx := types.NewTransaction(0, common.HexToAddress("0x0"), nil, 0, nil, data)

	requests <- tx
	done <- struct{}{}

	// TODO: Verify that header was added
}

func TestCreateCollation(t *testing.T) {
	backend, smc := setup(t)
	node := &mockNode{smc: smc, t: t, backend: backend}
	var txs []*types.Transaction
	for i := 0; i < 10; i++ {
		data := make([]byte, 1024)
		rand.Read(data)
		txs = append(txs, types.NewTransaction(0, common.HexToAddress("0x0"),
			nil, 0, nil, data))
	}

	collation, err := createCollation(node, node.Account(), node, big.NewInt(0), big.NewInt(1), txs)
	if err != nil {
		t.Fatalf("Create collation failed: %v", err)
	}

	// fast forward to 2nd period.
	for i := 0; i < 2*int(params.DefaultConfig.PeriodLength); i++ {
		backend.Commit()
	}

	// negative test case #1: create collation with shard > shardCount.
	collation, err = createCollation(node, node.Account(), node, big.NewInt(101), big.NewInt(2), txs)
	if err == nil {
		t.Errorf("Create collation should have failed with invalid shard number")
	}
	// negative test case #2, create collation with blob size > collationBodySizeLimit.
	var badTxs []*types.Transaction
	for i := 0; i <= 1024; i++ {
		data := make([]byte, 1024)
		rand.Read(data)
		badTxs = append(badTxs, types.NewTransaction(0, common.HexToAddress("0x0"),
			nil, 0, nil, data))
	}
	collation, err = createCollation(node, node.Account(), node, big.NewInt(0), big.NewInt(2), badTxs)
	if err == nil {
		t.Errorf("Create collation should have failed with Txs longer than collation body limit")
	}

	// normal test case #1 create collation with correct parameters.
	collation, err = createCollation(node, node.Account(), node, big.NewInt(5), big.NewInt(5), txs)
	if err != nil {
		t.Errorf("Create collation failed: %v", err)
	}
	if collation.Header().Period().Cmp(big.NewInt(5)) != 0 {
		t.Errorf("Incorrect collation period, want 5, got %v ", collation.Header().Period())
	}
	if collation.Header().ShardID().Cmp(big.NewInt(5)) != 0 {
		t.Errorf("Incorrect shard id, want 5, got %v ", collation.Header().ShardID())
	}
	if *collation.ProposerAddress() != node.Account().Address {
		t.Errorf("Incorrect proposer address, got %v", *collation.ProposerAddress())
	}
	if collation.Header().Sig() != nil {
		t.Errorf("Proposer signaure can not be empty")
	}
}

func TestAddCollation(t *testing.T) {
	backend, smc := setup(t)
	node := &mockNode{smc: smc, t: t, backend: backend}
	var txs []*types.Transaction
	for i := 0; i < 10; i++ {
		data := make([]byte, 1024)
		rand.Read(data)
		txs = append(txs, types.NewTransaction(0, common.HexToAddress("0x0"),
			nil, 0, nil, data))
	}

	collation, err := createCollation(node, node.Account(), node, big.NewInt(0), big.NewInt(1), txs)
	if err != nil {
		t.Errorf("Create collation failed: %v", err)
	}

	// fast forward to next period.
	for i := 0; i < int(params.DefaultConfig.PeriodLength); i++ {
		backend.Commit()
	}

	// normal test case #1 create collation with normal parameters.
	err = addHeader(node, collation)
	if err != nil {
		t.Errorf("%v", err)
	}
	backend.Commit()

	// verify collation was correctly added from SMC.
	collationFromSMC, err := smc.CollationRecords(&bind.CallOpts{}, big.NewInt(0), big.NewInt(1))
	if err != nil {
		t.Errorf("Failed to get collation record")
	}
	if collationFromSMC.Proposer != node.Account().Address {
		t.Errorf("Incorrect proposer address, got %v", *collation.ProposerAddress())
	}
	if common.BytesToHash(collationFromSMC.ChunkRoot[:]) != *collation.Header().ChunkRoot() {
		t.Errorf("Incorrect chunk root, got %v", collationFromSMC.ChunkRoot)
	}

	// negative test case #1 create the same collation that just got added to SMC.
	collation, err = createCollation(node, node.Account(), node, big.NewInt(0), big.NewInt(1), txs)
	if err == nil {
		t.Errorf("Create collation should fail due to same collation in SMC")
	}

}

func TestCheckCollation(t *testing.T) {
	backend, smc := setup(t)
	node := &mockNode{smc: smc, t: t, backend: backend}
	var txs []*types.Transaction
	for i := 0; i < 10; i++ {
		data := make([]byte, 1024)
		rand.Read(data)
		txs = append(txs, types.NewTransaction(0, common.HexToAddress("0x0"),
			nil, 0, nil, data))
	}

	collation, err := createCollation(node, node.Account(), node, big.NewInt(0), big.NewInt(1), txs)
	if err != nil {
		t.Errorf("Create collation failed: %v", err)
	}

	for i := 0; i < int(params.DefaultConfig.PeriodLength); i++ {
		backend.Commit()
	}

	err = addHeader(node, collation)
	if err != nil {
		t.Errorf("%v", err)
	}
	backend.Commit()

	// normal test case 1: check if we can still add header for period 1, should return false.
	a, err := checkHeaderAdded(node.SMCCaller(), big.NewInt(0), big.NewInt(1))
	if err != nil {
		t.Errorf("Can not check header submitted: %v", err)
	}
	if a {
		t.Errorf("Check header submitted shouldn't return: %v", a)
	}
	// normal test case 2: check if we can add header for period 2, should return true.
	a, err = checkHeaderAdded(node.SMCCaller(), big.NewInt(0), big.NewInt(2))
	if !a {
		t.Errorf("Check header submitted shouldn't return: %v", a)
	}
}
