package proposer

import (
	"crypto/rand"
	"fmt"
	"github.com/prysmaticlabs/geth-sharding/sharding/syncer"
	"math/big"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	gethTypes "github.com/ethereum/go-ethereum/core/types"
	pb "github.com/prysmaticlabs/geth-sharding/proto/sharding/v1"
	"github.com/prysmaticlabs/geth-sharding/sharding/database"
	"github.com/prysmaticlabs/geth-sharding/sharding/internal"
	"github.com/prysmaticlabs/geth-sharding/sharding/mainchain"
	"github.com/prysmaticlabs/geth-sharding/sharding/p2p"
	"github.com/prysmaticlabs/geth-sharding/sharding/params"
	"github.com/prysmaticlabs/geth-sharding/sharding/txpool"
	"github.com/prysmaticlabs/geth-sharding/sharding/types"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func TestProposeCollation(t *testing.T) {
	hook := logTest.NewGlobal()
	server, err := p2p.NewServer()

	backend, smc := internal.SetupMockClient(t)
	node := &internal.MockClient{SMC: smc, T: t, Backend: backend}
	if err != nil {
		t.Fatalf("Failed to start server %v", err)
	}
	server.Start()

	pool, err := txpool.NewTXPool(server)
	if err != nil {
		t.Fatalf("Failed to start server %v", err)
	}
	pool.Start()

	tmp := fmt.Sprintf("%s/datadir", os.TempDir())
	config := &database.ShardDBConfig{DataDir: tmp, Name: "shardDB", InMemory: false}

	db, err := database.NewShardDB(config)
	db.Start()

	fakeSyncer, err := syncer.NewSyncer(params.DefaultConfig, &mainchain.SMCClient{}, server, db, 1)
	if err != nil {
		t.Fatalf("Failed to start server %v", err)
	}

	fakeProposer, err := NewProposer(params.DefaultConfig, &mainchain.SMCClient{}, server, pool, db, 1, fakeSyncer)
	input := make([]byte, 0, 2000)
	for int64(len(input)) < (types.CollationSizelimit)/4 {
		input = append(input, []byte{'t', 'e', 's', 't', 'i', 'n', 'g'}...)
	}
	tx := pb.Transaction{Input: input}

	fakeProposer.Start()
	for i := 0; i < 50; i++ {
		fakeProposer.p2p.Broadcast(&tx)
	}

	msg := hook.LastEntry()
	want := "Collation created"
	if msg == nil || msg.Message != want {
		t.Errorf("incorrect log. wanted: %s. got: %v", want, msg)
	}

}
func TestCreateCollation(t *testing.T) {
	backend, smc := internal.SetupMockClient(t)
	node := &internal.MockClient{SMC: smc, T: t, Backend: backend}
	var txs []*gethTypes.Transaction
	for i := 0; i < 10; i++ {
		data := make([]byte, 1024)
		rand.Read(data)
		txs = append(txs, gethTypes.NewTransaction(0, common.HexToAddress("0x0"),
			nil, 0, nil, data))
	}

	_, err := createCollation(node, node.Account(), node, big.NewInt(0), big.NewInt(1), txs)
	if err != nil {
		t.Fatalf("Create collation failed: %v", err)
	}

	// fast forward to 2nd period.
	for i := 0; i < 2*int(params.DefaultConfig.PeriodLength); i++ {
		backend.Commit()
	}

	// negative test case #1: create collation with shard > shardCount.
	_, err = createCollation(node, node.Account(), node, big.NewInt(101), big.NewInt(2), txs)
	if err == nil {
		t.Errorf("Create collation should have failed with invalid shard number")
	}
	// negative test case #2, create collation with blob size > collationBodySizeLimit.
	var badTxs []*gethTypes.Transaction
	for i := 0; i <= 1024; i++ {
		data := make([]byte, 1024)
		rand.Read(data)
		badTxs = append(badTxs, gethTypes.NewTransaction(0, common.HexToAddress("0x0"),
			nil, 0, nil, data))
	}
	_, err = createCollation(node, node.Account(), node, big.NewInt(0), big.NewInt(2), badTxs)
	if err == nil {
		t.Errorf("Create collation should have failed with Txs longer than collation body limit")
	}

	// normal test case #1 create collation with correct parameters.
	collation, err := createCollation(node, node.Account(), node, big.NewInt(5), big.NewInt(5), txs)
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
	if collation.Header().Sig() != [32]byte{} {
		t.Errorf("Proposer signaure can not be empty")
	}
}

func TestAddCollation(t *testing.T) {
	backend, smc := internal.SetupMockClient(t)
	node := &internal.MockClient{SMC: smc, T: t, Backend: backend}
	var txs []*gethTypes.Transaction
	for i := 0; i < 10; i++ {
		data := make([]byte, 1024)
		rand.Read(data)
		txs = append(txs, gethTypes.NewTransaction(0, common.HexToAddress("0x0"),
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
	err = AddHeader(node, node, collation)
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
	_, err = createCollation(node, node.Account(), node, big.NewInt(0), big.NewInt(1), txs)
	if err == nil {
		t.Errorf("Create collation should fail due to same collation in SMC")
	}
}

func TestCheckCollation(t *testing.T) {
	backend, smc := internal.SetupMockClient(t)
	node := &internal.MockClient{SMC: smc, T: t, Backend: backend}
	var txs []*gethTypes.Transaction
	for i := 0; i < 10; i++ {
		data := make([]byte, 1024)
		rand.Read(data)
		txs = append(txs, gethTypes.NewTransaction(0, common.HexToAddress("0x0"),
			nil, 0, nil, data))
	}

	collation, err := createCollation(node, node.Account(), node, big.NewInt(0), big.NewInt(1), txs)
	if err != nil {
		t.Errorf("Create collation failed: %v", err)
	}

	for i := 0; i < int(params.DefaultConfig.PeriodLength); i++ {
		backend.Commit()
	}

	err = AddHeader(node, node, collation)
	if err != nil {
		t.Errorf("%v", err)
	}
	backend.Commit()

	// normal test case 1: check if we can still add header for period 1, should return false.
	a, err := checkHeaderAdded(node, big.NewInt(0), big.NewInt(1))
	if err != nil {
		t.Errorf("Can not check header submitted: %v", err)
	}
	if a {
		t.Errorf("Check header submitted shouldn't return: %v", a)
	}
	// normal test case 2: check if we can add header for period 2, should return true.
	a, err = checkHeaderAdded(node, big.NewInt(0), big.NewInt(2))
	if err != nil {
		t.Error(err)
	}
	if !a {
		t.Errorf("Check header submitted shouldn't return: %v", a)
	}
}
