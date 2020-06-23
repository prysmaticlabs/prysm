package powchain

import (
	"bytes"
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	gethTypes "github.com/ethereum/go-ethereum/core/types"
	dbutil "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	mockPOW "github.com/prysmaticlabs/prysm/beacon-chain/powchain/testing"
	contracts "github.com/prysmaticlabs/prysm/contracts/deposit-contract"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
)

var endpoint = "http://127.0.0.1"

func setDefaultMocks(service *Service) *Service {
	service.eth1DataFetcher = &goodFetcher{}
	service.httpLogger = &goodLogger{}
	service.stateNotifier = &goodNotifier{}
	return service
}

func TestLatestMainchainInfo_OK(t *testing.T) {
	testAcc, err := contracts.Setup()
	if err != nil {
		t.Fatalf("Unable to set up simulated backend %v", err)
	}
	beaconDB, _ := dbutil.SetupDB(t)
	web3Service, err := NewService(context.Background(), &Web3ServiceConfig{
		HTTPEndPoint:    endpoint,
		DepositContract: testAcc.ContractAddr,
		BeaconDB:        beaconDB,
	})
	if err != nil {
		t.Fatalf("unable to setup web3 ETH1.0 chain service: %v", err)
	}
	web3Service = setDefaultMocks(web3Service)
	web3Service.rpcClient = &mockPOW.RPCClient{Backend: testAcc.Backend}
	web3Service.eth1DataFetcher = &goodFetcher{backend: testAcc.Backend}

	web3Service.depositContractCaller, err = contracts.NewDepositContractCaller(testAcc.ContractAddr, testAcc.Backend)
	if err != nil {
		t.Fatal(err)
	}
	testAcc.Backend.Commit()

	exitRoutine := make(chan bool)

	go func() {
		web3Service.run(web3Service.ctx.Done())
		<-exitRoutine
	}()

	header, err := web3Service.eth1DataFetcher.HeaderByNumber(web3Service.ctx, nil)
	if err != nil {
		t.Fatal(err)
	}

	tickerChan := make(chan time.Time)
	web3Service.headTicker = &time.Ticker{C: tickerChan}
	tickerChan <- time.Now()
	web3Service.cancel()
	exitRoutine <- true

	if web3Service.latestEth1Data.BlockHeight != header.Number.Uint64() {
		t.Errorf("block number not set, expected %v, got %v", header.Number, web3Service.latestEth1Data.BlockHeight)
	}

	if hexutil.Encode(web3Service.latestEth1Data.BlockHash) != header.Hash().Hex() {
		t.Errorf("block hash not set, expected %v, got %#x", header.Hash().Hex(), web3Service.latestEth1Data.BlockHash)
	}

	if web3Service.latestEth1Data.BlockTime != header.Time {
		t.Errorf("block time not set, expected %v, got %v", time.Unix(int64(header.Time), 0), web3Service.latestEth1Data.BlockTime)
	}

	blockInfoExistsInCache, info, err := web3Service.blockCache.BlockInfoByHash(bytesutil.ToBytes32(web3Service.latestEth1Data.BlockHash))
	if err != nil {
		t.Fatal(err)
	}
	if !blockInfoExistsInCache {
		t.Error("Expected block info to exist in cache")
	}
	if info.Hash != bytesutil.ToBytes32(web3Service.latestEth1Data.BlockHash) {
		t.Errorf(
			"Expected block info hash to be %v, got %v",
			bytesutil.ToBytes32(web3Service.latestEth1Data.BlockHash),
			info.Hash,
		)
	}
}

func TestBlockHashByHeight_ReturnsHash(t *testing.T) {
	beaconDB, _ := dbutil.SetupDB(t)
	web3Service, err := NewService(context.Background(), &Web3ServiceConfig{
		HTTPEndPoint: endpoint,
		BeaconDB:     beaconDB,
	})
	if err != nil {
		t.Fatalf("unable to setup web3 ETH1.0 chain service: %v", err)
	}
	web3Service = setDefaultMocks(web3Service)
	ctx := context.Background()

	block := gethTypes.NewBlock(
		&gethTypes.Header{
			Number: big.NewInt(15),
			Time:   150,
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
	beaconDB, _ := dbutil.SetupDB(t)
	web3Service, err := NewService(context.Background(), &Web3ServiceConfig{
		HTTPEndPoint: endpoint,
		BeaconDB:     beaconDB,
	})
	if err != nil {
		t.Fatalf("unable to setup web3 ETH1.0 chain service: %v", err)
	}
	web3Service = setDefaultMocks(web3Service)

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
	beaconDB, _ := dbutil.SetupDB(t)
	web3Service, err := NewService(context.Background(), &Web3ServiceConfig{
		HTTPEndPoint: endpoint,
		BeaconDB:     beaconDB,
	})
	if err != nil {
		t.Fatalf("unable to setup web3 ETH1.0 chain service: %v", err)
	}
	web3Service = setDefaultMocks(web3Service)

	_, _, err = web3Service.BlockExists(context.Background(), common.BytesToHash([]byte{0}))
	if err == nil {
		t.Fatal("Expected BlockExists to error with invalid hash")
	}
}

func TestBlockExists_UsesCachedBlockInfo(t *testing.T) {
	beaconDB, _ := dbutil.SetupDB(t)
	web3Service, err := NewService(context.Background(), &Web3ServiceConfig{
		HTTPEndPoint: endpoint,
		BeaconDB:     beaconDB,
	})
	if err != nil {
		t.Fatalf("unable to setup web3 ETH1.0 chain service: %v", err)
	}
	// nil eth1DataFetcher would panic if cached value not used
	web3Service.eth1DataFetcher = nil

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

func TestBlockNumberByTimestamp(t *testing.T) {
	beaconDB, _ := dbutil.SetupDB(t)
	web3Service, err := NewService(context.Background(), &Web3ServiceConfig{
		HTTPEndPoint: endpoint,
		BeaconDB:     beaconDB,
	})
	if err != nil {
		t.Fatal(err)
	}
	web3Service = setDefaultMocks(web3Service)

	ctx := context.Background()
	bn, err := web3Service.BlockNumberByTimestamp(ctx, 150000 /* time */)
	if err != nil {
		t.Fatal(err)
	}

	if bn.Cmp(big.NewInt(0)) == 0 {
		t.Error("Returned a block with zero number, expected to be non zero")
	}
}
