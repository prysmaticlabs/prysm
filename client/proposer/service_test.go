package proposer

import (
	"crypto/rand"
	"fmt"
	"math"
	"time"
	//"math"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	gethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/prysmaticlabs/prysm/client/database"
	"github.com/prysmaticlabs/prysm/client/internal"
	"github.com/prysmaticlabs/prysm/client/mainchain"
	"github.com/prysmaticlabs/prysm/client/params"
	"github.com/prysmaticlabs/prysm/client/syncer"
	"github.com/prysmaticlabs/prysm/client/txpool"
	pb "github.com/prysmaticlabs/prysm/proto/sharding/v1"
	"github.com/prysmaticlabs/prysm/shared/p2p"
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

	c := params.GetDefaultConfig()
	fakeProposer, err := NewProposer(c, node, server, pool, db, 1, fakeSyncer)
	if err != nil {
		t.Fatalf("Failed to create proposer %v", err)
	}
	fakeProposer.config.CollationSizeLimit = int64(math.Pow(float64(2), float64(10)))

	return fakeProposer, node
}

// waitForLogMsg keeps checking for log messages until "want" is found or a
// timeout expires, returning the index of the message if found.
// The motivation is to be more resilient to connection failure errors that can
// appear in the log:
// Failed to connect to peer: dial attempt failed: <peer.ID 16Uiu2> --> <peer.ID 16Uiu2> dial attempt failed: connection refused
func waitForLogMsg(hook *logTest.Hook, want string) (int, error) {
	timeout := time.After(60 * time.Second)
	tick := time.Tick(500 * time.Millisecond)
	for {
		select {
		case <-timeout:
			return 0, fmt.Errorf("didn't get %q log entry on time", want)
		case <-tick:
			for index, msg := range hook.AllEntries() {
				if msg.Message == want {
					return index, nil
				}
			}
		}
	}
}

func TestProposerRoundTrip(t *testing.T) {
	hook := logTest.NewGlobal()
	fakeProposer, node := settingUpProposer(t)
	defer func() {
		fakeProposer.dbService.Stop()
		fakeProposer.txpool.Stop()
		fakeProposer.p2p.Stop()
	}()

	input := make([]byte, 0, 2000)
	for len(input) < int(fakeProposer.config.CollationSizeLimit/4) {
		input = append(input, []byte{'t', 'e', 's', 't', 'i', 'n', 'g'}...)
	}
	tx := pb.Transaction{Input: input}

	for i := 0; i < 5; i++ {
		node.CommitWithBlock()
	}
	fakeProposer.Start()
	defer fakeProposer.Stop()

	for i := 0; i < 4; i++ {
		fakeProposer.p2p.Broadcast(&tx)
	}

	if _, err := waitForLogMsg(hook, "Collation created"); err != nil {
		t.Fatal(err.Error())
	}
}

func TestIncompleteCollation(t *testing.T) {
	hook := logTest.NewGlobal()
	fakeProposer, node := settingUpProposer(t)
	defer func() {
		fakeProposer.dbService.Stop()
		fakeProposer.txpool.Stop()
		fakeProposer.p2p.Stop()
	}()

	input := make([]byte, 0, 2000)
	for int64(len(input)) < (fakeProposer.config.CollationSizeLimit)/4 {
		input = append(input, []byte{'t', 'e', 's', 't', 'i', 'n', 'g'}...)
	}
	tx := pb.Transaction{Input: input}

	for i := 0; i < 5; i++ {
		node.CommitWithBlock()
	}
	fakeProposer.Start()

	for i := 0; i < 3; i++ {
		fakeProposer.p2p.Broadcast(&tx)
	}
	fakeProposer.Stop()

	index, err := waitForLogMsg(hook, "Stopping proposer service")
	if err != nil {
		t.Fatal(err.Error())
	}
	for i := index; i >= 0; i-- {
		if hook.AllEntries()[i].Message == "Collation created" {
			t.Fatal("Collation created before stopping proposer")
		}
	}
}

func TestCollationWitInDiffPeriod(t *testing.T) {
	hook := logTest.NewGlobal()
	fakeProposer, node := settingUpProposer(t)
	defer func() {
		fakeProposer.dbService.Stop()
		fakeProposer.txpool.Stop()
		fakeProposer.p2p.Stop()
	}()

	input := make([]byte, 0, 2000)
	for int64(len(input)) < (fakeProposer.config.CollationSizeLimit)/4 {
		input = append(input, []byte{'t', 'e', 's', 't', 'i', 'n', 'g'}...)
	}
	tx := pb.Transaction{Input: input}

	for i := 0; i < 5; i++ {
		node.CommitWithBlock()
	}
	fakeProposer.Start()
	defer fakeProposer.Stop()

	for i := 0; i < 5; i++ {
		node.CommitWithBlock()
	}

	fakeProposer.p2p.Broadcast(&tx)

	if _, err := waitForLogMsg(hook, "Collation created"); err != nil {
		t.Fatal(err.Error())
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
