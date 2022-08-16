package execution

import (
	"context"
	"encoding/json"
	"errors"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/holiman/uint256"
	mockChain "github.com/prysmaticlabs/prysm/v3/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/feed"
	statefeed "github.com/prysmaticlabs/prysm/v3/beacon-chain/core/feed/state"
	mocks "github.com/prysmaticlabs/prysm/v3/beacon-chain/execution/testing"
	fieldparams "github.com/prysmaticlabs/prysm/v3/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/blocks"
	pb "github.com/prysmaticlabs/prysm/v3/proto/engine/v1"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
	logTest "github.com/sirupsen/logrus/hooks/test"
	"google.golang.org/protobuf/proto"
)

func Test_checkTransitionConfiguration(t *testing.T) {
	params.SetupTestConfigCleanup(t)
	cfg := params.BeaconConfig().Copy()
	cfg.BellatrixForkEpoch = 0
	params.OverrideBeaconConfig(cfg)
	hook := logTest.NewGlobal()

	t.Run("context canceled", func(t *testing.T) {
		ctx := context.Background()
		m := &mocks.EngineClient{}
		m.Err = errors.New("something went wrong")

		srv := setupTransitionConfigTest(t)
		srv.cfg.stateNotifier = &mockChain.MockStateNotifier{}
		checkTransitionPollingInterval = time.Millisecond
		ctx, cancel := context.WithCancel(ctx)
		go srv.checkTransitionConfiguration(ctx, make(chan *feed.Event, 1))
		<-time.After(100 * time.Millisecond)
		cancel()
		require.LogsContain(t, hook, "Could not check configuration values")
	})

	t.Run("block containing execution payload exits routine", func(t *testing.T) {
		ctx := context.Background()
		m := &mocks.EngineClient{}
		m.Err = errors.New("something went wrong")
		srv := setupTransitionConfigTest(t)
		srv.cfg.stateNotifier = &mockChain.MockStateNotifier{}

		checkTransitionPollingInterval = time.Millisecond
		ctx, cancel := context.WithCancel(ctx)
		exit := make(chan bool)
		notification := make(chan *feed.Event)
		go func() {
			srv.checkTransitionConfiguration(ctx, notification)
			exit <- true
		}()
		payload := emptyPayload()
		payload.GasUsed = 21000
		wrappedBlock, err := blocks.NewSignedBeaconBlock(&ethpb.SignedBeaconBlockBellatrix{
			Block: &ethpb.BeaconBlockBellatrix{
				Body: &ethpb.BeaconBlockBodyBellatrix{
					ExecutionPayload: payload,
				},
			}},
		)
		require.NoError(t, err)
		notification <- &feed.Event{
			Data: &statefeed.BlockProcessedData{
				SignedBlock: wrappedBlock,
			},
			Type: statefeed.BlockProcessed,
		}
		<-exit
		cancel()
		require.LogsContain(t, hook, "PoS transition is complete, no longer checking")
	})
}

func TestService_handleExchangeConfigurationError(t *testing.T) {
	hook := logTest.NewGlobal()
	t.Run("clears existing service error", func(t *testing.T) {
		srv := setupTransitionConfigTest(t)
		srv.isRunning = true
		srv.runError = ErrConfigMismatch
		srv.handleExchangeConfigurationError(nil)
		require.Equal(t, true, srv.Status() == nil)
	})
	t.Run("does not clear existing service error if wrong kind", func(t *testing.T) {
		srv := setupTransitionConfigTest(t)
		srv.isRunning = true
		err := errors.New("something else went wrong")
		srv.runError = err
		srv.handleExchangeConfigurationError(nil)
		require.ErrorIs(t, err, srv.Status())
	})
	t.Run("sets service error on config mismatch", func(t *testing.T) {
		srv := setupTransitionConfigTest(t)
		srv.isRunning = true
		srv.handleExchangeConfigurationError(ErrConfigMismatch)
		require.Equal(t, ErrConfigMismatch, srv.Status())
		require.LogsContain(t, hook, configMismatchLog)
	})
	t.Run("does not set service error if unrelated problem", func(t *testing.T) {
		srv := setupTransitionConfigTest(t)
		srv.isRunning = true
		srv.handleExchangeConfigurationError(errors.New("foo"))
		require.Equal(t, true, srv.Status() == nil)
		require.LogsContain(t, hook, "Could not check configuration values")
	})
}

func setupTransitionConfigTest(t testing.TB) *Service {
	fix := fixtures()
	request, ok := fix["TransitionConfiguration"].(*pb.TransitionConfiguration)
	require.Equal(t, true, ok)
	resp, ok := proto.Clone(request).(*pb.TransitionConfiguration)
	require.Equal(t, true, ok)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		defer func() {
			require.NoError(t, r.Body.Close())
		}()

		// Change the terminal block hash.
		h := common.BytesToHash([]byte("foo"))
		resp.TerminalBlockHash = h[:]
		respJSON := map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      1,
			"result":  resp,
		}
		require.NoError(t, json.NewEncoder(w).Encode(respJSON))
	}))
	defer srv.Close()

	rpcClient, err := rpc.DialHTTP(srv.URL)
	require.NoError(t, err)
	defer rpcClient.Close()

	service := &Service{
		cfg: &config{},
	}
	service.rpcClient = rpcClient
	return service
}

func TestService_logTtdStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		defer func() {
			require.NoError(t, r.Body.Close())
		}()

		resp := &pb.ExecutionBlock{
			Header: gethtypes.Header{
				ParentHash:  common.Hash{},
				UncleHash:   common.Hash{},
				Coinbase:    common.Address{},
				Root:        common.Hash{},
				TxHash:      common.Hash{},
				ReceiptHash: common.Hash{},
				Bloom:       gethtypes.Bloom{},
				Difficulty:  big.NewInt(1),
				Number:      big.NewInt(2),
				GasLimit:    3,
				GasUsed:     4,
				Time:        5,
				Extra:       nil,
				MixDigest:   common.Hash{},
				Nonce:       gethtypes.BlockNonce{},
				BaseFee:     big.NewInt(6),
			},
			TotalDifficulty: "0x12345678",
		}
		respJSON := map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      1,
			"result":  resp,
		}
		require.NoError(t, json.NewEncoder(w).Encode(respJSON))
	}))
	defer srv.Close()

	rpcClient, err := rpc.DialHTTP(srv.URL)
	require.NoError(t, err)
	defer rpcClient.Close()

	service := &Service{
		cfg: &config{},
	}
	service.rpcClient = rpcClient

	ttd := new(uint256.Int)
	reached, err := service.logTtdStatus(context.Background(), ttd.SetUint64(24343))
	require.NoError(t, err)
	require.Equal(t, true, reached)

	reached, err = service.logTtdStatus(context.Background(), ttd.SetUint64(323423484))
	require.NoError(t, err)
	require.Equal(t, false, reached)
}

func TestService_logTtdStatus_NotSyncedClient(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		defer func() {
			require.NoError(t, r.Body.Close())
		}()

		resp := (*pb.ExecutionBlock)(nil) // Nil response when a client is not synced
		respJSON := map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      1,
			"result":  resp,
		}
		require.NoError(t, json.NewEncoder(w).Encode(respJSON))
	}))
	defer srv.Close()

	rpcClient, err := rpc.DialHTTP(srv.URL)
	require.NoError(t, err)
	defer rpcClient.Close()

	service := &Service{
		cfg: &config{},
	}
	service.rpcClient = rpcClient

	ttd := new(uint256.Int)
	reached, err := service.logTtdStatus(context.Background(), ttd.SetUint64(24343))
	require.NoError(t, err)
	require.Equal(t, false, reached)
}

func emptyPayload() *pb.ExecutionPayload {
	return &pb.ExecutionPayload{
		ParentHash:    make([]byte, fieldparams.RootLength),
		FeeRecipient:  make([]byte, fieldparams.FeeRecipientLength),
		StateRoot:     make([]byte, fieldparams.RootLength),
		ReceiptsRoot:  make([]byte, fieldparams.RootLength),
		LogsBloom:     make([]byte, fieldparams.LogsBloomLength),
		PrevRandao:    make([]byte, fieldparams.RootLength),
		BaseFeePerGas: make([]byte, fieldparams.RootLength),
		BlockHash:     make([]byte, fieldparams.RootLength),
		Transactions:  make([][]byte, 0),
		ExtraData:     make([]byte, 0),
	}
}
