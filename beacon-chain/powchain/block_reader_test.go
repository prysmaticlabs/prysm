package powchain

import (
	"bytes"
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/prysmaticlabs/prysm/beacon-chain/db"

	"github.com/ethereum/go-ethereum/common"
	gethTypes "github.com/ethereum/go-ethereum/core/types"
)

var endpoint = "ws://127.0.0.1"

func TestLatestMainchainInfo_OK(t *testing.T) {
	testAcc, err := setup()
	if err != nil {
		t.Fatalf("Unable to set up simulated backend %v", err)
	}

	beaconDB, err := db.SetupDB()
	if err != nil {
		t.Fatalf("unable to set up simulated db instance: %v", err)
	}
	web3Service, err := NewWeb3Service(context.Background(), &Web3ServiceConfig{
		Endpoint:        endpoint,
		DepositContract: testAcc.contractAddr,
		Reader:          &goodReader{},
		Logger:          &goodLogger{},
		HTTPLogger:      &goodLogger{},
		BlockFetcher:    &goodFetcher{},
		ContractBackend: testAcc.backend,
		BeaconDB:        beaconDB,
	})
	if err != nil {
		t.Fatalf("unable to setup web3 ETH1.0 chain service: %v", err)
	}
	testAcc.backend.Commit()

	exitRoutine := make(chan bool)

	go func() {
		web3Service.run(web3Service.ctx.Done())
		<-exitRoutine
	}()

	header := &gethTypes.Header{
		Number: big.NewInt(42),
		Time:   308534400,
	}

	web3Service.headerChan <- header
	web3Service.cancel()
	exitRoutine <- true

	if web3Service.blockHeight.Cmp(header.Number) != 0 {
		t.Errorf("block number not set, expected %v, got %v", header.Number, web3Service.blockHeight)
	}

	if web3Service.blockHash.Hex() != header.Hash().Hex() {
		t.Errorf("block hash not set, expected %v, got %v", header.Hash().Hex(), web3Service.blockHash.Hex())
	}

	if web3Service.blockTime != time.Unix(int64(header.Time), 0) {
		t.Errorf("block time not set, expected %v, got %v", time.Unix(int64(header.Time), 0), web3Service.blockTime)
	}

	blockInfoExistsInCache, info, err := web3Service.blockCache.BlockInfoByHash(web3Service.blockHash)
	if err != nil {
		t.Fatal(err)
	}
	if !blockInfoExistsInCache {
		t.Error("Expected block info to exist in cache")
	}
	if info.Hash != web3Service.blockHash {
		t.Errorf(
			"Expected block info hash to be %v, got %v",
			web3Service.blockHash,
			info.Hash,
		)
	}
}

func TestBlockHashByHeight_ReturnsHash(t *testing.T) {
	web3Service, err := NewWeb3Service(context.Background(), &Web3ServiceConfig{
		Endpoint:     endpoint,
		BlockFetcher: &goodFetcher{},
	})
	if err != nil {
		t.Fatalf("unable to setup web3 ETH1.0 chain service: %v", err)
	}
	ctx := context.Background()

	block := gethTypes.NewBlock(
		&gethTypes.Header{
			Number: big.NewInt(0),
		},
		[]*gethTypes.Transaction{},
		[]*gethTypes.Header{},
		[]*gethTypes.Receipt{},
	)
	wanted := block.Hash()

	hash, err := web3Service.BlockHashByHeight(ctx, big.NewInt(0))
	if err != nil {
		t.Fatalf("Could not get block hash with given height %v", err)
	}

	if !bytes.Equal(hash.Bytes(), wanted.Bytes()) {
		t.Fatalf("Block hash did not equal expected hash, expected: %v, got: %v", wanted, hash)
	}

	exists, _, err := web3Service.blockCache.BlockInfoByHash(wanted)
	if err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Error("Expected block info to be cached")
	}
}

func TestBlockExists_ValidHash(t *testing.T) {
	web3Service, err := NewWeb3Service(context.Background(), &Web3ServiceConfig{
		Endpoint:     endpoint,
		BlockFetcher: &goodFetcher{},
	})
	if err != nil {
		t.Fatalf("unable to setup web3 ETH1.0 chain service: %v", err)
	}

	block := gethTypes.NewBlock(
		&gethTypes.Header{
			Number: big.NewInt(0),
		},
		[]*gethTypes.Transaction{},
		[]*gethTypes.Header{},
		[]*gethTypes.Receipt{},
	)

	exists, height, err := web3Service.BlockExists(context.Background(), block.Hash())
	if err != nil {
		t.Fatalf("Could not get block hash with given height %v", err)
	}

	if !exists {
		t.Fatal("Expected BlockExists to return true.")
	}
	if height.Cmp(block.Number()) != 0 {
		t.Fatalf("Block height did not equal expected height, expected: %v, got: %v", big.NewInt(42), height)
	}

	exists, _, err = web3Service.blockCache.BlockInfoByHeight(height)
	if err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Error("Expected block to be cached")
	}
}

func TestBlockExists_InvalidHash(t *testing.T) {
	web3Service, err := NewWeb3Service(context.Background(), &Web3ServiceConfig{
		Endpoint:     endpoint,
		BlockFetcher: &goodFetcher{},
	})
	if err != nil {
		t.Fatalf("unable to setup web3 ETH1.0 chain service: %v", err)
	}

	_, _, err = web3Service.BlockExists(context.Background(), common.BytesToHash([]byte{0}))
	if err == nil {
		t.Fatal("Expected BlockExists to error with invalid hash")
	}
}

func TestBlockExists_UsesCachedBlockInfo(t *testing.T) {
	web3Service, err := NewWeb3Service(context.Background(), &Web3ServiceConfig{
		Endpoint:     endpoint,
		BlockFetcher: nil, // nil blockFetcher would panic if cached value not used
	})
	if err != nil {
		t.Fatalf("unable to setup web3 ETH1.0 chain service: %v", err)
	}

	block := gethTypes.NewBlock(
		&gethTypes.Header{
			Number: big.NewInt(0),
		},
		[]*gethTypes.Transaction{},
		[]*gethTypes.Header{},
		[]*gethTypes.Receipt{},
	)

	if err := web3Service.blockCache.AddBlock(block); err != nil {
		t.Fatal(err)
	}

	exists, height, err := web3Service.BlockExists(context.Background(), block.Hash())
	if err != nil {
		t.Fatalf("Could not get block hash with given height %v", err)
	}

	if !exists {
		t.Fatal("Expected BlockExists to return true.")
	}
	if height.Cmp(block.Number()) != 0 {
		t.Fatalf("Block height did not equal expected height, expected: %v, got: %v", big.NewInt(42), height)
	}
}
