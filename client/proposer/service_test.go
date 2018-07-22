package proposer

import (
	"crypto/rand"
	"math/big"
	"testing"

	"github.com/prysmaticlabs/prysm/client/syncer"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	gethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/prysmaticlabs/prysm/client/database"
	"github.com/prysmaticlabs/prysm/client/internal"
	"github.com/prysmaticlabs/prysm/client/mainchain"
	"github.com/prysmaticlabs/prysm/client/p2p"
	"github.com/prysmaticlabs/prysm/client/params"
	"github.com/prysmaticlabs/prysm/client/txpool"
	"github.com/prysmaticlabs/prysm/client/types"
	pb "github.com/prysmaticlabs/prysm/proto/sharding/v1"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func settingUpProposer(t *testing.T) (*Proposer, *internal.MockClient) {
	backend, smc := internal.SetupMockClient(t)
	node := &internal.MockClient{SMC: smc, T: t, Backend: backend}

	server, err := p2p.NewServer()
	if err != nil {
		t.Fatalf("Failed to start server %v", err)
	}
	server.Start()

	pool, err := txpool.NewTXPool(server)
	if err != nil {
		t.Fatalf("Failed to start txpool %v", err)
	}
	pool.Start()

	config := &database.ShardDBConfig{DataDir: "", Name: "", InMemory: true}

	db, err := database.NewShardDB(config)
	if err != nil {
		t.Fatalf("Failed create shardDB %v", err)
	}

	db.Start()

	fakeSyncer, err := syncer.NewSyncer(params.DefaultConfig, &mainchain.SMCClient{}, server, db, 1)
	if err != nil {
		t.Fatalf("Failed to start syncer %v", err)
	}

	fakeProposer, err := NewProposer(params.DefaultConfig, node, server, pool, db, 1, fakeSyncer)
	if err != nil {
		t.Fatalf("Failed to create proposer %v", err)
	}

	return fakeProposer, node

}

func TestProposerRoundTrip(t *testing.T) {
	hook := logTest.NewGlobal()
	fakeProposer, node := settingUpProposer(t)

	input := make([]byte, 0, 2000)
	for int64(len(input)) < (types.CollationSizeLimit)/4 {
		input = append(input, []byte{'t', 'e', 's', 't', 'i', 'n', 'g'}...)
	}
	tx := pb.Transaction{Input: input}

	fakeProposer.Start()
	for i := 0; i < 5; i++ {
		node.CommitWithBlock()
	}

	for i := 0; i < 4; i++ {
		fakeProposer.p2p.Broadcast(&tx)
		<-fakeProposer.msgChan
	}

	want := "Collation created"
	length := len(hook.AllEntries())
	for length < 9 {
		length = len(hook.AllEntries())
	}

	msg := hook.LastEntry()

	if msg.Message != want {
		t.Errorf("Incorrect log, wanted %v but got %v", want, msg.Message)
	}

	fakeProposer.cancel()

}

func TestIncompleteCollation(t *testing.T) {
	hook := logTest.NewGlobal()
	fakeProposer, node := settingUpProposer(t)

	input := make([]byte, 0, 2000)
	for int64(len(input)) < (types.CollationSizeLimit)/4 {
		input = append(input, []byte{'t', 'e', 's', 't', 'i', 'n', 'g'}...)
	}
	tx := pb.Transaction{Input: input}

	fakeProposer.Start()
	for i := 0; i < 5; i++ {
		node.CommitWithBlock()
	}

	for i := 0; i < 3; i++ {
		fakeProposer.p2p.Broadcast(&tx)
		<-fakeProposer.msgChan
	}

	want := "Starting proposer service"

	msg := hook.LastEntry()
	if msg.Message != want {
		t.Errorf("Incorrect log, wanted %v but got %v", want, msg.Message)
	}

	length := len(hook.AllEntries())

	if length != 4 {
		t.Errorf("Number of logs was supposed to be 4 but is %v", length)
	}

	fakeProposer.cancel()
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
