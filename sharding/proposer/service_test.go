package proposer

import (
	"crypto/rand"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/sharding/internal"
	"github.com/ethereum/go-ethereum/sharding/params"
)

var (
	key, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
	addr   = crypto.PubkeyToAddress(key.PublicKey)
)

func TestCreateCollation(t *testing.T) {
	backend, smc := internal.SetupMockClient(t)
	node := &internal.MockClient{SMC: smc, T: t, Backend: backend}
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
	backend, smc := internal.SetupMockClient(t)
	node := &internal.MockClient{SMC: smc, T: t, Backend: backend}
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
	err = AddHeader(node, collation)
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
	backend, smc := internal.SetupMockClient(t)
	node := &internal.MockClient{SMC: smc, T: t, Backend: backend}
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

	err = AddHeader(node, collation)
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
	if !a {
		t.Errorf("Check header submitted shouldn't return: %v", a)
	}
}
