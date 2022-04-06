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
	contracts "github.com/prysmaticlabs/prysm/contracts/deposit"
	"github.com/prysmaticlabs/prysm/contracts/deposit/mock"
	"github.com/prysmaticlabs/prysm/testing/assert"
	"github.com/prysmaticlabs/prysm/testing/require"
)

func setDefaultMocks(service *Service) *Service {
	service.eth1DataFetcher = &goodFetcher{}
	service.httpLogger = &goodLogger{}
	service.cfg.stateNotifier = &goodNotifier{}
	return service
}

func TestLatestMainchainInfo_OK(t *testing.T) {
	testAcc, err := mock.Setup()
	require.NoError(t, err, "Unable to set up simulated backend")

	beaconDB := dbutil.SetupDB(t)
	server, endpoint, err := mockPOW.SetupRPCServer()
	require.NoError(t, err)
	t.Cleanup(func() {
		server.Stop()
	})
	web3Service, err := NewService(context.Background(),
		WithHttpEndpoints([]string{endpoint}),
		WithDepositContractAddress(testAcc.ContractAddr),
		WithDatabase(beaconDB),
	)
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
}

func TestBlockHashByHeight_ReturnsHash(t *testing.T) {
	beaconDB := dbutil.SetupDB(t)
	server, endpoint, err := mockPOW.SetupRPCServer()
	require.NoError(t, err)
	t.Cleanup(func() {
		server.Stop()
	})
	web3Service, err := NewService(context.Background(),
		WithHttpEndpoints([]string{endpoint}),
		WithDatabase(beaconDB),
	)
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
	beaconDB := dbutil.SetupDB(t)
	server, endpoint, err := mockPOW.SetupRPCServer()
	require.NoError(t, err)
	t.Cleanup(func() {
		server.Stop()
	})
	web3Service, err := NewService(context.Background(),
		WithHttpEndpoints([]string{endpoint}),
		WithDatabase(beaconDB),
	)
	require.NoError(t, err, "unable to setup web3 ETH1.0 chain service")

	web3Service = setDefaultMocks(web3Service)
	web3Service.eth1DataFetcher = nil
	ctx := context.Background()

	_, err = web3Service.BlockHashByHeight(ctx, big.NewInt(0))
	require.ErrorContains(t, "nil eth1DataFetcher", err)
}

func TestBlockExists_ValidHash(t *testing.T) {
	beaconDB := dbutil.SetupDB(t)
	server, endpoint, err := mockPOW.SetupRPCServer()
	require.NoError(t, err)
	t.Cleanup(func() {
		server.Stop()
	})
	web3Service, err := NewService(context.Background(),
		WithHttpEndpoints([]string{endpoint}),
		WithDatabase(beaconDB),
	)
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
	beaconDB := dbutil.SetupDB(t)
	server, endpoint, err := mockPOW.SetupRPCServer()
	require.NoError(t, err)
	t.Cleanup(func() {
		server.Stop()
	})
	web3Service, err := NewService(context.Background(),
		WithHttpEndpoints([]string{endpoint}),
		WithDatabase(beaconDB),
	)
	require.NoError(t, err, "unable to setup web3 ETH1.0 chain service")

	web3Service = setDefaultMocks(web3Service)

	_, _, err = web3Service.BlockExists(context.Background(), common.BytesToHash([]byte{0}))
	require.NotNil(t, err, "Expected BlockExists to error with invalid hash")
}

func TestBlockExists_UsesCachedBlockInfo(t *testing.T) {
	beaconDB := dbutil.SetupDB(t)
	server, endpoint, err := mockPOW.SetupRPCServer()
	require.NoError(t, err)
	t.Cleanup(func() {
		server.Stop()
	})
	web3Service, err := NewService(context.Background(),
		WithHttpEndpoints([]string{endpoint}),
		WithDatabase(beaconDB),
	)
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

func TestBlockExistsWithCache_UsesCachedHeaderInfo(t *testing.T) {
	beaconDB := dbutil.SetupDB(t)
	server, endpoint, err := mockPOW.SetupRPCServer()
	require.NoError(t, err)
	t.Cleanup(func() {
		server.Stop()
	})
	web3Service, err := NewService(context.Background(),
		WithHttpEndpoints([]string{endpoint}),
		WithDatabase(beaconDB),
	)
	require.NoError(t, err, "unable to setup web3 ETH1.0 chain service")

	header := &gethTypes.Header{
		Number: big.NewInt(0),
	}

	err = web3Service.headerCache.AddHeader(header)
	require.NoError(t, err)

	exists, height, err := web3Service.BlockExistsWithCache(context.Background(), header.Hash())
	require.NoError(t, err, "Could not get block hash with given height")
	require.Equal(t, true, exists)
	require.Equal(t, 0, height.Cmp(header.Number))
}

func TestBlockExistsWithCache_HeaderNotCached(t *testing.T) {
	beaconDB := dbutil.SetupDB(t)
	server, endpoint, err := mockPOW.SetupRPCServer()
	require.NoError(t, err)
	t.Cleanup(func() {
		server.Stop()
	})
	web3Service, err := NewService(context.Background(),
		WithHttpEndpoints([]string{endpoint}),
		WithDatabase(beaconDB),
	)
	require.NoError(t, err, "unable to setup web3 ETH1.0 chain service")

	exists, height, err := web3Service.BlockExistsWithCache(context.Background(), common.BytesToHash([]byte("hash")))
	require.NoError(t, err, "Could not get block hash with given height")
	require.Equal(t, false, exists)
	require.Equal(t, (*big.Int)(nil), height)
}

func TestService_BlockNumberByTimestamp(t *testing.T) {
	beaconDB := dbutil.SetupDB(t)
	testAcc, err := mock.Setup()
	require.NoError(t, err, "Unable to set up simulated backend")
	server, endpoint, err := mockPOW.SetupRPCServer()
	require.NoError(t, err)
	t.Cleanup(func() {
		server.Stop()
	})
	web3Service, err := NewService(context.Background(),
		WithHttpEndpoints([]string{endpoint}),
		WithDatabase(beaconDB),
	)
	require.NoError(t, err)
	web3Service = setDefaultMocks(web3Service)
	web3Service.eth1DataFetcher = &goodFetcher{backend: testAcc.Backend}

	for i := 0; i < 200; i++ {
		testAcc.Backend.Commit()
	}
	ctx := context.Background()
	hd, err := testAcc.Backend.HeaderByNumber(ctx, nil)
	require.NoError(t, err)
	web3Service.latestEth1Data.BlockTime = hd.Time
	web3Service.latestEth1Data.BlockHeight = hd.Number.Uint64()
	blk, err := web3Service.BlockByTimestamp(ctx, 1000 /* time */)
	require.NoError(t, err)
	if blk.Number.Cmp(big.NewInt(0)) == 0 {
		t.Error("Returned a block with zero number, expected to be non zero")
	}
}

func TestService_BlockNumberByTimestampLessTargetTime(t *testing.T) {
	beaconDB := dbutil.SetupDB(t)
	testAcc, err := mock.Setup()
	require.NoError(t, err, "Unable to set up simulated backend")
	server, endpoint, err := mockPOW.SetupRPCServer()
	require.NoError(t, err)
	t.Cleanup(func() {
		server.Stop()
	})
	web3Service, err := NewService(context.Background(),
		WithHttpEndpoints([]string{endpoint}),
		WithDatabase(beaconDB),
	)
	require.NoError(t, err)
	web3Service = setDefaultMocks(web3Service)
	web3Service.eth1DataFetcher = &goodFetcher{backend: testAcc.Backend}

	for i := 0; i < 200; i++ {
		testAcc.Backend.Commit()
	}
	ctx := context.Background()
	hd, err := testAcc.Backend.HeaderByNumber(ctx, nil)
	require.NoError(t, err)
	web3Service.latestEth1Data.BlockTime = hd.Time
	// Use extremely small deadline to illustrate that context deadlines are respected.
	ctx, cancel := context.WithTimeout(ctx, 100*time.Nanosecond)
	defer cancel()

	// Provide an unattainable target time
	_, err = web3Service.findLessTargetEth1Block(ctx, hd.Number, hd.Time/2)
	require.ErrorContains(t, context.DeadlineExceeded.Error(), err)

	// Provide an attainable target time
	blk, err := web3Service.findLessTargetEth1Block(context.Background(), hd.Number, hd.Time-5)
	require.NoError(t, err)
	require.NotEqual(t, hd.Number.Uint64(), blk.Number.Uint64(), "retrieved block is not less than the head")
}

func TestService_BlockNumberByTimestampMoreTargetTime(t *testing.T) {
	beaconDB := dbutil.SetupDB(t)
	testAcc, err := mock.Setup()
	require.NoError(t, err, "Unable to set up simulated backend")
	server, endpoint, err := mockPOW.SetupRPCServer()
	require.NoError(t, err)
	t.Cleanup(func() {
		server.Stop()
	})
	web3Service, err := NewService(context.Background(),
		WithHttpEndpoints([]string{endpoint}),
		WithDatabase(beaconDB),
	)
	require.NoError(t, err)
	web3Service = setDefaultMocks(web3Service)
	web3Service.eth1DataFetcher = &goodFetcher{backend: testAcc.Backend}

	for i := 0; i < 200; i++ {
		testAcc.Backend.Commit()
	}
	ctx := context.Background()
	hd, err := testAcc.Backend.HeaderByNumber(ctx, nil)
	require.NoError(t, err)
	web3Service.latestEth1Data.BlockTime = hd.Time
	// Use extremely small deadline to illustrate that context deadlines are respected.
	ctx, cancel := context.WithTimeout(ctx, 100*time.Nanosecond)
	defer cancel()

	// Provide an unattainable target time with respect to head
	_, err = web3Service.findMoreTargetEth1Block(ctx, big.NewInt(0).Div(hd.Number, big.NewInt(2)), hd.Time)
	require.ErrorContains(t, context.DeadlineExceeded.Error(), err)

	// Provide an attainable target time with respect to head
	blk, err := web3Service.findMoreTargetEth1Block(context.Background(), big.NewInt(0).Sub(hd.Number, big.NewInt(5)), hd.Time)
	require.NoError(t, err)
	require.Equal(t, hd.Number.Uint64(), blk.Number.Uint64(), "retrieved block is not equal to the head")
}

func TestService_BlockTimeByHeight_ReturnsError_WhenNoEth1Client(t *testing.T) {
	beaconDB := dbutil.SetupDB(t)
	server, endpoint, err := mockPOW.SetupRPCServer()
	require.NoError(t, err)
	t.Cleanup(func() {
		server.Stop()
	})
	web3Service, err := NewService(context.Background(),
		WithHttpEndpoints([]string{endpoint}),
		WithDatabase(beaconDB),
	)
	require.NoError(t, err, "unable to setup web3 ETH1.0 chain service")

	web3Service = setDefaultMocks(web3Service)
	web3Service.eth1DataFetcher = nil
	ctx := context.Background()

	_, err = web3Service.BlockTimeByHeight(ctx, big.NewInt(0))
	require.ErrorContains(t, "nil eth1DataFetcher", err)
}
