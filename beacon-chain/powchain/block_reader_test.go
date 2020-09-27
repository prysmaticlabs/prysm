package powchain

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	gethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/trie"
	dbutil "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	mockPOW "github.com/prysmaticlabs/prysm/beacon-chain/powchain/testing"
	contracts "github.com/prysmaticlabs/prysm/contracts/deposit-contract"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
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
	require.NoError(t, err, "Unable to set up simulated backend")

	beaconDB, _ := dbutil.SetupDB(t)
	web3Service, err := NewService(context.Background(), &Web3ServiceConfig{
		HTTPEndPoint:    endpoint,
		DepositContract: testAcc.ContractAddr,
		BeaconDB:        beaconDB,
	})
	require.NoError(t, err, "Unable to setup web3 ETH1.0 chain service")

	web3Service = setDefaultMocks(web3Service)
	web3Service.rpcClient = &mockPOW.RPCClient{Backend: testAcc.Backend}
	web3Service.eth1DataFetcher = &goodFetcher{backend: testAcc.Backend}

	web3Service.depositContractCaller, err = contracts.NewDepositContractCaller(testAcc.ContractAddr, testAcc.Backend)
	require.NoError(t, err)
	testAcc.Backend.Commit()

	exitRoutine := make(chan bool)

	go func() {
		web3Service.run(web3Service.ctx.Done())
		<-exitRoutine
	}()

	header, err := web3Service.eth1DataFetcher.HeaderByNumber(web3Service.ctx, nil)
	require.NoError(t, err)

	tickerChan := make(chan time.Time)
	web3Service.headTicker = &time.Ticker{C: tickerChan}
	tickerChan <- time.Now()
	web3Service.cancel()
	exitRoutine <- true

	assert.Equal(t, web3Service.latestEth1Data.BlockHeight, header.Number.Uint64())
	assert.Equal(t, hexutil.Encode(web3Service.latestEth1Data.BlockHash), header.Hash().Hex())
	assert.Equal(t, web3Service.latestEth1Data.BlockTime, header.Time)

	headerInfoExistsInCache, info, err := web3Service.headerCache.HeaderInfoByHash(bytesutil.ToBytes32(web3Service.latestEth1Data.BlockHash))
	require.NoError(t, err)
	assert.Equal(t, true, headerInfoExistsInCache, "Expected block info to exist in cache")
	assert.Equal(t, bytesutil.ToBytes32(info.Hash[:]), bytesutil.ToBytes32(web3Service.latestEth1Data.BlockHash))

}

func TestBlockHashByHeight_ReturnsHash(t *testing.T) {
	beaconDB, _ := dbutil.SetupDB(t)
	web3Service, err := NewService(context.Background(), &Web3ServiceConfig{
		HTTPEndPoint: endpoint,
		BeaconDB:     beaconDB,
	})
	require.NoError(t, err, "unable to setup web3 ETH1.0 chain service")

	web3Service = setDefaultMocks(web3Service)
	ctx := context.Background()

	header := &gethTypes.Header{
		Number: big.NewInt(15),
		Time:   150,
	}

	wanted := header.Hash()

	hash, err := web3Service.BlockHashByHeight(ctx, big.NewInt(0))
	require.NoError(t, err, "Could not get block hash with given height")
	require.DeepEqual(t, wanted.Bytes(), hash.Bytes(), "Block hash did not equal expected hash")

	exists, _, err := web3Service.headerCache.HeaderInfoByHash(wanted)
	require.NoError(t, err)
	require.Equal(t, true, exists, "Expected block info to be cached")
}

func TestBlockHashByHeight_ReturnsError_WhenNoEth1Client(t *testing.T) {
	beaconDB, _ := dbutil.SetupDB(t)
	web3Service, err := NewService(context.Background(), &Web3ServiceConfig{
		HTTPEndPoint: endpoint,
		BeaconDB:     beaconDB,
	})
	require.NoError(t, err, "unable to setup web3 ETH1.0 chain service")

	web3Service = setDefaultMocks(web3Service)
	web3Service.eth1DataFetcher = nil
	ctx := context.Background()

	_, err = web3Service.BlockHashByHeight(ctx, big.NewInt(0))
	require.ErrorContains(t, "nil eth1DataFetcher", err)
}

func TestBlockExists_ValidHash(t *testing.T) {
	beaconDB, _ := dbutil.SetupDB(t)
	web3Service, err := NewService(context.Background(), &Web3ServiceConfig{
		HTTPEndPoint: endpoint,
		BeaconDB:     beaconDB,
	})
	require.NoError(t, err, "unable to setup web3 ETH1.0 chain service")

	web3Service = setDefaultMocks(web3Service)

	block := gethTypes.NewBlock(
		&gethTypes.Header{
			Number: big.NewInt(0),
		},
		[]*gethTypes.Transaction{},
		[]*gethTypes.Header{},
		[]*gethTypes.Receipt{},
		new(trie.Trie),
	)

	exists, height, err := web3Service.BlockExists(context.Background(), block.Hash())
	require.NoError(t, err, "Could not get block hash with given height")
	require.Equal(t, true, exists)
	require.Equal(t, 0, height.Cmp(block.Number()))

	exists, _, err = web3Service.headerCache.HeaderInfoByHeight(height)
	require.NoError(t, err)
	require.Equal(t, true, exists, "Expected block to be cached")

}

func TestBlockExists_InvalidHash(t *testing.T) {
	beaconDB, _ := dbutil.SetupDB(t)
	web3Service, err := NewService(context.Background(), &Web3ServiceConfig{
		HTTPEndPoint: endpoint,
		BeaconDB:     beaconDB,
	})
	require.NoError(t, err, "unable to setup web3 ETH1.0 chain service")

	web3Service = setDefaultMocks(web3Service)

	_, _, err = web3Service.BlockExists(context.Background(), common.BytesToHash([]byte{0}))
	require.NotNil(t, err, "Expected BlockExists to error with invalid hash")
}

func TestBlockExists_UsesCachedBlockInfo(t *testing.T) {
	beaconDB, _ := dbutil.SetupDB(t)
	web3Service, err := NewService(context.Background(), &Web3ServiceConfig{
		HTTPEndPoint: endpoint,
		BeaconDB:     beaconDB,
	})
	require.NoError(t, err, "unable to setup web3 ETH1.0 chain service")
	// nil eth1DataFetcher would panic if cached value not used
	web3Service.eth1DataFetcher = nil

	header := &gethTypes.Header{
		Number: big.NewInt(0),
	}

	err = web3Service.headerCache.AddHeader(header)
	require.NoError(t, err)

	exists, height, err := web3Service.BlockExists(context.Background(), header.Hash())
	require.NoError(t, err, "Could not get block hash with given height")
	require.Equal(t, true, exists)
	require.Equal(t, 0, height.Cmp(header.Number))
}

func TestBlockNumberByTimestamp(t *testing.T) {
	beaconDB, _ := dbutil.SetupDB(t)
	web3Service, err := NewService(context.Background(), &Web3ServiceConfig{
		HTTPEndPoint: endpoint,
		BeaconDB:     beaconDB,
	})
	require.NoError(t, err)
	web3Service = setDefaultMocks(web3Service)

	ctx := context.Background()
	bn, err := web3Service.BlockNumberByTimestamp(ctx, 150000 /* time */)
	require.NoError(t, err)
	if bn.Cmp(big.NewInt(0)) == 0 {
		t.Error("Returned a block with zero number, expected to be non zero")
	}
}

func TestService_BlockTimeByHeight_ReturnsError_WhenNoEth1Client(t *testing.T) {
	beaconDB, _ := dbutil.SetupDB(t)
	web3Service, err := NewService(context.Background(), &Web3ServiceConfig{
		HTTPEndPoint: endpoint,
		BeaconDB:     beaconDB,
	})
	require.NoError(t, err, "unable to setup web3 ETH1.0 chain service")

	web3Service = setDefaultMocks(web3Service)
	web3Service.eth1DataFetcher = nil
	ctx := context.Background()

	_, err = web3Service.BlockTimeByHeight(ctx, big.NewInt(0))
	require.ErrorContains(t, "nil eth1DataFetcher", err)
}
