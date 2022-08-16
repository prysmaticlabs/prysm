package execution

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/holiman/uint256"
	"github.com/pkg/errors"
	mocks "github.com/prysmaticlabs/prysm/v3/beacon-chain/execution/testing"
	fieldparams "github.com/prysmaticlabs/prysm/v3/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/interfaces"
	"github.com/prysmaticlabs/prysm/v3/encoding/bytesutil"
	pb "github.com/prysmaticlabs/prysm/v3/proto/engine/v1"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
	"github.com/prysmaticlabs/prysm/v3/testing/util"
	"google.golang.org/protobuf/proto"
)

var (
	_ = ExecutionPayloadReconstructor(&Service{})
	_ = EngineCaller(&Service{})
	_ = ExecutionPayloadReconstructor(&Service{})
	_ = EngineCaller(&mocks.EngineClient{})
)

func TestClient_IPC(t *testing.T) {
	server := newTestIPCServer(t)
	defer server.Stop()
	rpcClient := rpc.DialInProc(server)
	defer rpcClient.Close()
	srv := &Service{}
	srv.rpcClient = rpcClient
	ctx := context.Background()
	fix := fixtures()

	t.Run(GetPayloadMethod, func(t *testing.T) {
		want, ok := fix["ExecutionPayload"].(*pb.ExecutionPayload)
		require.Equal(t, true, ok)
		payloadId := [8]byte{1}
		resp, err := srv.GetPayload(ctx, payloadId)
		require.NoError(t, err)
		require.DeepEqual(t, want, resp)
	})
	t.Run(ForkchoiceUpdatedMethod, func(t *testing.T) {
		want, ok := fix["ForkchoiceUpdatedResponse"].(*ForkchoiceUpdatedResponse)
		require.Equal(t, true, ok)
		payloadID, validHash, err := srv.ForkchoiceUpdated(ctx, &pb.ForkchoiceState{}, &pb.PayloadAttributes{})
		require.NoError(t, err)
		require.DeepEqual(t, want.Status.LatestValidHash, validHash)
		require.DeepEqual(t, want.PayloadId, payloadID)
	})
	t.Run(NewPayloadMethod, func(t *testing.T) {
		want, ok := fix["ValidPayloadStatus"].(*pb.PayloadStatus)
		require.Equal(t, true, ok)
		req, ok := fix["ExecutionPayload"].(*pb.ExecutionPayload)
		require.Equal(t, true, ok)
		wrappedPayload, err := blocks.WrappedExecutionPayload(req)
		require.NoError(t, err)
		latestValidHash, err := srv.NewPayload(ctx, wrappedPayload)
		require.NoError(t, err)
		require.DeepEqual(t, bytesutil.ToBytes32(want.LatestValidHash), bytesutil.ToBytes32(latestValidHash))
	})
	t.Run(ExchangeTransitionConfigurationMethod, func(t *testing.T) {
		want, ok := fix["TransitionConfiguration"].(*pb.TransitionConfiguration)
		require.Equal(t, true, ok)
		err := srv.ExchangeTransitionConfiguration(ctx, want)
		require.NoError(t, err)
	})
	t.Run(ExecutionBlockByNumberMethod, func(t *testing.T) {
		want, ok := fix["ExecutionBlock"].(*pb.ExecutionBlock)
		require.Equal(t, true, ok)
		resp, err := srv.LatestExecutionBlock(ctx)
		require.NoError(t, err)
		require.DeepEqual(t, want, resp)
	})
	t.Run(ExecutionBlockByHashMethod, func(t *testing.T) {
		want, ok := fix["ExecutionBlock"].(*pb.ExecutionBlock)
		require.Equal(t, true, ok)
		arg := common.BytesToHash([]byte("foo"))
		resp, err := srv.ExecutionBlockByHash(ctx, arg, true /* with txs */)
		require.NoError(t, err)
		require.DeepEqual(t, want, resp)
	})
}

func TestClient_HTTP(t *testing.T) {
	ctx := context.Background()
	fix := fixtures()

	t.Run(GetPayloadMethod, func(t *testing.T) {
		payloadId := [8]byte{1}
		want, ok := fix["ExecutionPayload"].(*pb.ExecutionPayload)
		require.Equal(t, true, ok)
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			defer func() {
				require.NoError(t, r.Body.Close())
			}()
			enc, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			jsonRequestString := string(enc)

			reqArg, err := json.Marshal(pb.PayloadIDBytes(payloadId))
			require.NoError(t, err)

			// We expect the JSON string RPC request contains the right arguments.
			require.Equal(t, true, strings.Contains(
				jsonRequestString, string(reqArg),
			))
			resp := map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      1,
				"result":  want,
			}
			err = json.NewEncoder(w).Encode(resp)
			require.NoError(t, err)
		}))
		defer srv.Close()

		rpcClient, err := rpc.DialHTTP(srv.URL)
		require.NoError(t, err)
		defer rpcClient.Close()

		client := &Service{}
		client.rpcClient = rpcClient

		// We call the RPC method via HTTP and expect a proper result.
		resp, err := client.GetPayload(ctx, payloadId)
		require.NoError(t, err)
		require.DeepEqual(t, want, resp)
	})
	t.Run(ForkchoiceUpdatedMethod+" VALID status", func(t *testing.T) {
		forkChoiceState := &pb.ForkchoiceState{
			HeadBlockHash:      []byte("head"),
			SafeBlockHash:      []byte("safe"),
			FinalizedBlockHash: []byte("finalized"),
		}
		payloadAttributes := &pb.PayloadAttributes{
			Timestamp:             1,
			PrevRandao:            []byte("random"),
			SuggestedFeeRecipient: []byte("suggestedFeeRecipient"),
		}
		want, ok := fix["ForkchoiceUpdatedResponse"].(*ForkchoiceUpdatedResponse)
		require.Equal(t, true, ok)
		srv := forkchoiceUpdateSetup(t, forkChoiceState, payloadAttributes, want)

		// We call the RPC method via HTTP and expect a proper result.
		payloadID, validHash, err := srv.ForkchoiceUpdated(ctx, forkChoiceState, payloadAttributes)
		require.NoError(t, err)
		require.DeepEqual(t, want.Status.LatestValidHash, validHash)
		require.DeepEqual(t, want.PayloadId, payloadID)
	})
	t.Run(ForkchoiceUpdatedMethod+" SYNCING status", func(t *testing.T) {
		forkChoiceState := &pb.ForkchoiceState{
			HeadBlockHash:      []byte("head"),
			SafeBlockHash:      []byte("safe"),
			FinalizedBlockHash: []byte("finalized"),
		}
		payloadAttributes := &pb.PayloadAttributes{
			Timestamp:             1,
			PrevRandao:            []byte("random"),
			SuggestedFeeRecipient: []byte("suggestedFeeRecipient"),
		}
		want, ok := fix["ForkchoiceUpdatedSyncingResponse"].(*ForkchoiceUpdatedResponse)
		require.Equal(t, true, ok)
		client := forkchoiceUpdateSetup(t, forkChoiceState, payloadAttributes, want)

		// We call the RPC method via HTTP and expect a proper result.
		payloadID, validHash, err := client.ForkchoiceUpdated(ctx, forkChoiceState, payloadAttributes)
		require.ErrorIs(t, err, ErrAcceptedSyncingPayloadStatus)
		require.DeepEqual(t, (*pb.PayloadIDBytes)(nil), payloadID)
		require.DeepEqual(t, []byte(nil), validHash)
	})
	t.Run(ForkchoiceUpdatedMethod+" INVALID status", func(t *testing.T) {
		forkChoiceState := &pb.ForkchoiceState{
			HeadBlockHash:      []byte("head"),
			SafeBlockHash:      []byte("safe"),
			FinalizedBlockHash: []byte("finalized"),
		}
		payloadAttributes := &pb.PayloadAttributes{
			Timestamp:             1,
			PrevRandao:            []byte("random"),
			SuggestedFeeRecipient: []byte("suggestedFeeRecipient"),
		}
		want, ok := fix["ForkchoiceUpdatedInvalidResponse"].(*ForkchoiceUpdatedResponse)
		require.Equal(t, true, ok)
		client := forkchoiceUpdateSetup(t, forkChoiceState, payloadAttributes, want)

		// We call the RPC method via HTTP and expect a proper result.
		payloadID, validHash, err := client.ForkchoiceUpdated(ctx, forkChoiceState, payloadAttributes)
		require.ErrorIs(t, err, ErrInvalidPayloadStatus)
		require.DeepEqual(t, (*pb.PayloadIDBytes)(nil), payloadID)
		require.DeepEqual(t, want.Status.LatestValidHash, validHash)
	})
	t.Run(ForkchoiceUpdatedMethod+" UNKNOWN status", func(t *testing.T) {
		forkChoiceState := &pb.ForkchoiceState{
			HeadBlockHash:      []byte("head"),
			SafeBlockHash:      []byte("safe"),
			FinalizedBlockHash: []byte("finalized"),
		}
		payloadAttributes := &pb.PayloadAttributes{
			Timestamp:             1,
			PrevRandao:            []byte("random"),
			SuggestedFeeRecipient: []byte("suggestedFeeRecipient"),
		}
		want, ok := fix["ForkchoiceUpdatedAcceptedResponse"].(*ForkchoiceUpdatedResponse)
		require.Equal(t, true, ok)
		client := forkchoiceUpdateSetup(t, forkChoiceState, payloadAttributes, want)

		// We call the RPC method via HTTP and expect a proper result.
		payloadID, validHash, err := client.ForkchoiceUpdated(ctx, forkChoiceState, payloadAttributes)
		require.ErrorIs(t, err, ErrUnknownPayloadStatus)
		require.DeepEqual(t, (*pb.PayloadIDBytes)(nil), payloadID)
		require.DeepEqual(t, []byte(nil), validHash)
	})
	t.Run(NewPayloadMethod+" VALID status", func(t *testing.T) {
		execPayload, ok := fix["ExecutionPayload"].(*pb.ExecutionPayload)
		require.Equal(t, true, ok)
		want, ok := fix["ValidPayloadStatus"].(*pb.PayloadStatus)
		require.Equal(t, true, ok)
		client := newPayloadSetup(t, want, execPayload)

		// We call the RPC method via HTTP and expect a proper result.
		wrappedPayload, err := blocks.WrappedExecutionPayload(execPayload)
		require.NoError(t, err)
		resp, err := client.NewPayload(ctx, wrappedPayload)
		require.NoError(t, err)
		require.DeepEqual(t, want.LatestValidHash, resp)
	})
	t.Run(NewPayloadMethod+" SYNCING status", func(t *testing.T) {
		execPayload, ok := fix["ExecutionPayload"].(*pb.ExecutionPayload)
		require.Equal(t, true, ok)
		want, ok := fix["SyncingStatus"].(*pb.PayloadStatus)
		require.Equal(t, true, ok)
		client := newPayloadSetup(t, want, execPayload)

		// We call the RPC method via HTTP and expect a proper result.
		wrappedPayload, err := blocks.WrappedExecutionPayload(execPayload)
		require.NoError(t, err)
		resp, err := client.NewPayload(ctx, wrappedPayload)
		require.ErrorIs(t, ErrAcceptedSyncingPayloadStatus, err)
		require.DeepEqual(t, []uint8(nil), resp)
	})
	t.Run(NewPayloadMethod+" INVALID_BLOCK_HASH status", func(t *testing.T) {
		execPayload, ok := fix["ExecutionPayload"].(*pb.ExecutionPayload)
		require.Equal(t, true, ok)
		want, ok := fix["InvalidBlockHashStatus"].(*pb.PayloadStatus)
		require.Equal(t, true, ok)
		client := newPayloadSetup(t, want, execPayload)

		// We call the RPC method via HTTP and expect a proper result.
		wrappedPayload, err := blocks.WrappedExecutionPayload(execPayload)
		require.NoError(t, err)
		resp, err := client.NewPayload(ctx, wrappedPayload)
		require.ErrorIs(t, ErrInvalidBlockHashPayloadStatus, err)
		require.DeepEqual(t, []uint8(nil), resp)
	})
	t.Run(NewPayloadMethod+" INVALID status", func(t *testing.T) {
		execPayload, ok := fix["ExecutionPayload"].(*pb.ExecutionPayload)
		require.Equal(t, true, ok)
		want, ok := fix["InvalidStatus"].(*pb.PayloadStatus)
		require.Equal(t, true, ok)
		client := newPayloadSetup(t, want, execPayload)

		// We call the RPC method via HTTP and expect a proper result.
		wrappedPayload, err := blocks.WrappedExecutionPayload(execPayload)
		require.NoError(t, err)
		resp, err := client.NewPayload(ctx, wrappedPayload)
		require.ErrorIs(t, ErrInvalidPayloadStatus, err)
		require.DeepEqual(t, want.LatestValidHash, resp)
	})
	t.Run(NewPayloadMethod+" UNKNOWN status", func(t *testing.T) {
		execPayload, ok := fix["ExecutionPayload"].(*pb.ExecutionPayload)
		require.Equal(t, true, ok)
		want, ok := fix["UnknownStatus"].(*pb.PayloadStatus)
		require.Equal(t, true, ok)
		client := newPayloadSetup(t, want, execPayload)

		// We call the RPC method via HTTP and expect a proper result.
		wrappedPayload, err := blocks.WrappedExecutionPayload(execPayload)
		require.NoError(t, err)
		resp, err := client.NewPayload(ctx, wrappedPayload)
		require.ErrorIs(t, ErrUnknownPayloadStatus, err)
		require.DeepEqual(t, []uint8(nil), resp)
	})
	t.Run(ExecutionBlockByNumberMethod, func(t *testing.T) {
		want, ok := fix["ExecutionBlock"].(*pb.ExecutionBlock)
		require.Equal(t, true, ok)
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			defer func() {
				require.NoError(t, r.Body.Close())
			}()
			resp := map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      1,
				"result":  want,
			}
			err := json.NewEncoder(w).Encode(resp)
			require.NoError(t, err)
		}))
		defer srv.Close()

		rpcClient, err := rpc.DialHTTP(srv.URL)
		require.NoError(t, err)
		defer rpcClient.Close()

		service := &Service{}
		service.rpcClient = rpcClient

		// We call the RPC method via HTTP and expect a proper result.
		resp, err := service.LatestExecutionBlock(ctx)
		require.NoError(t, err)
		require.DeepEqual(t, want, resp)
	})
	t.Run(ExchangeTransitionConfigurationMethod, func(t *testing.T) {
		want, ok := fix["TransitionConfiguration"].(*pb.TransitionConfiguration)
		require.Equal(t, true, ok)
		encodedReq, err := json.Marshal(want)
		require.NoError(t, err)
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			defer func() {
				require.NoError(t, r.Body.Close())
			}()
			enc, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			jsonRequestString := string(enc)
			// We expect the JSON string RPC request contains the right arguments.
			require.Equal(t, true, strings.Contains(
				jsonRequestString, string(encodedReq),
			))
			resp := map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      1,
				"result":  want,
			}
			err = json.NewEncoder(w).Encode(resp)
			require.NoError(t, err)
		}))
		defer srv.Close()

		rpcClient, err := rpc.DialHTTP(srv.URL)
		require.NoError(t, err)
		defer rpcClient.Close()

		client := &Service{}
		client.rpcClient = rpcClient

		// We call the RPC method via HTTP and expect a proper result.
		err = client.ExchangeTransitionConfiguration(ctx, want)
		require.NoError(t, err)
	})
	t.Run(ExecutionBlockByHashMethod, func(t *testing.T) {
		arg := common.BytesToHash([]byte("foo"))
		want, ok := fix["ExecutionBlock"].(*pb.ExecutionBlock)
		require.Equal(t, true, ok)
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			defer func() {
				require.NoError(t, r.Body.Close())
			}()
			enc, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			jsonRequestString := string(enc)
			// We expect the JSON string RPC request contains the right arguments.
			require.Equal(t, true, strings.Contains(
				jsonRequestString, fmt.Sprintf("%#x", arg),
			))
			resp := map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      1,
				"result":  want,
			}
			err = json.NewEncoder(w).Encode(resp)
			require.NoError(t, err)
		}))
		defer srv.Close()

		rpcClient, err := rpc.DialHTTP(srv.URL)
		require.NoError(t, err)
		defer rpcClient.Close()

		service := &Service{}
		service.rpcClient = rpcClient

		// We call the RPC method via HTTP and expect a proper result.
		resp, err := service.ExecutionBlockByHash(ctx, arg, true /* with txs */)
		require.NoError(t, err)
		require.DeepEqual(t, want, resp)
	})
}

func TestReconstructFullBellatrixBlock(t *testing.T) {
	ctx := context.Background()
	t.Run("nil block", func(t *testing.T) {
		service := &Service{}

		_, err := service.ReconstructFullBellatrixBlock(ctx, nil)
		require.ErrorContains(t, "nil data", err)
	})
	t.Run("only blinded block", func(t *testing.T) {
		want := "can only reconstruct block from blinded block format"
		service := &Service{}
		bellatrixBlock := util.NewBeaconBlockBellatrix()
		wrapped, err := blocks.NewSignedBeaconBlock(bellatrixBlock)
		require.NoError(t, err)
		_, err = service.ReconstructFullBellatrixBlock(ctx, wrapped)
		require.ErrorContains(t, want, err)
	})
	t.Run("pre-merge execution payload", func(t *testing.T) {
		service := &Service{}
		bellatrixBlock := util.NewBlindedBeaconBlockBellatrix()
		wanted := util.NewBeaconBlockBellatrix()
		wanted.Block.Slot = 1
		// Make sure block hash is the zero hash.
		bellatrixBlock.Block.Body.ExecutionPayloadHeader.BlockHash = make([]byte, 32)
		bellatrixBlock.Block.Slot = 1
		wrapped, err := blocks.NewSignedBeaconBlock(bellatrixBlock)
		require.NoError(t, err)
		wantedWrapped, err := blocks.NewSignedBeaconBlock(wanted)
		require.NoError(t, err)
		reconstructed, err := service.ReconstructFullBellatrixBlock(ctx, wrapped)
		require.NoError(t, err)
		require.DeepEqual(t, wantedWrapped, reconstructed)
	})
	t.Run("properly reconstructs block with correct payload", func(t *testing.T) {
		fix := fixtures()
		payload, ok := fix["ExecutionPayload"].(*pb.ExecutionPayload)
		require.Equal(t, true, ok)

		jsonPayload := make(map[string]interface{})
		tx := gethtypes.NewTransaction(
			0,
			common.HexToAddress("095e7baea6a6c7c4c2dfeb977efac326af552d87"),
			big.NewInt(0), 0, big.NewInt(0),
			nil,
		)
		txs := []*gethtypes.Transaction{tx}
		encodedBinaryTxs := make([][]byte, 1)
		var err error
		encodedBinaryTxs[0], err = txs[0].MarshalBinary()
		require.NoError(t, err)
		payload.Transactions = encodedBinaryTxs
		jsonPayload["transactions"] = txs
		num := big.NewInt(1)
		encodedNum := hexutil.EncodeBig(num)
		jsonPayload["hash"] = hexutil.Encode(payload.BlockHash)
		jsonPayload["parentHash"] = common.BytesToHash([]byte("parent"))
		jsonPayload["sha3Uncles"] = common.BytesToHash([]byte("uncles"))
		jsonPayload["miner"] = common.BytesToAddress([]byte("miner"))
		jsonPayload["stateRoot"] = common.BytesToHash([]byte("state"))
		jsonPayload["transactionsRoot"] = common.BytesToHash([]byte("txs"))
		jsonPayload["receiptsRoot"] = common.BytesToHash([]byte("receipts"))
		jsonPayload["logsBloom"] = gethtypes.BytesToBloom([]byte("bloom"))
		jsonPayload["gasLimit"] = hexutil.EncodeUint64(1)
		jsonPayload["gasUsed"] = hexutil.EncodeUint64(2)
		jsonPayload["timestamp"] = hexutil.EncodeUint64(3)
		jsonPayload["number"] = encodedNum
		jsonPayload["extraData"] = common.BytesToHash([]byte("extra"))
		jsonPayload["totalDifficulty"] = "0x123456"
		jsonPayload["difficulty"] = encodedNum
		jsonPayload["size"] = encodedNum
		jsonPayload["baseFeePerGas"] = encodedNum

		wrappedPayload, err := blocks.WrappedExecutionPayload(payload)
		require.NoError(t, err)
		header, err := blocks.PayloadToHeader(wrappedPayload)
		require.NoError(t, err)

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			defer func() {
				require.NoError(t, r.Body.Close())
			}()
			respJSON := map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      1,
				"result":  jsonPayload,
			}
			require.NoError(t, json.NewEncoder(w).Encode(respJSON))
		}))
		defer srv.Close()

		rpcClient, err := rpc.DialHTTP(srv.URL)
		require.NoError(t, err)
		defer rpcClient.Close()

		service := &Service{}
		service.rpcClient = rpcClient
		blindedBlock := util.NewBlindedBeaconBlockBellatrix()

		blindedBlock.Block.Body.ExecutionPayloadHeader = header
		wrapped, err := blocks.NewSignedBeaconBlock(blindedBlock)
		require.NoError(t, err)
		reconstructed, err := service.ReconstructFullBellatrixBlock(ctx, wrapped)
		require.NoError(t, err)

		got, err := reconstructed.Block().Body().Execution()
		require.NoError(t, err)
		require.DeepEqual(t, payload, got.Proto())
	})
}

func TestReconstructFullBellatrixBlockBatch(t *testing.T) {
	ctx := context.Background()
	t.Run("nil block", func(t *testing.T) {
		service := &Service{}

		_, err := service.ReconstructFullBellatrixBlockBatch(ctx, []interfaces.SignedBeaconBlock{nil})
		require.ErrorContains(t, "nil data", err)
	})
	t.Run("only blinded block", func(t *testing.T) {
		want := "can only reconstruct block from blinded block format"
		service := &Service{}
		bellatrixBlock := util.NewBeaconBlockBellatrix()
		wrapped, err := blocks.NewSignedBeaconBlock(bellatrixBlock)
		require.NoError(t, err)
		_, err = service.ReconstructFullBellatrixBlockBatch(ctx, []interfaces.SignedBeaconBlock{wrapped})
		require.ErrorContains(t, want, err)
	})
	t.Run("pre-merge execution payload", func(t *testing.T) {
		service := &Service{}
		bellatrixBlock := util.NewBlindedBeaconBlockBellatrix()
		wanted := util.NewBeaconBlockBellatrix()
		wanted.Block.Slot = 1
		// Make sure block hash is the zero hash.
		bellatrixBlock.Block.Body.ExecutionPayloadHeader.BlockHash = make([]byte, 32)
		bellatrixBlock.Block.Slot = 1
		wrapped, err := blocks.NewSignedBeaconBlock(bellatrixBlock)
		require.NoError(t, err)
		wantedWrapped, err := blocks.NewSignedBeaconBlock(wanted)
		require.NoError(t, err)
		reconstructed, err := service.ReconstructFullBellatrixBlockBatch(ctx, []interfaces.SignedBeaconBlock{wrapped})
		require.NoError(t, err)
		require.DeepEqual(t, []interfaces.SignedBeaconBlock{wantedWrapped}, reconstructed)
	})
	t.Run("properly reconstructs block batch with correct payload", func(t *testing.T) {
		fix := fixtures()
		payload, ok := fix["ExecutionPayload"].(*pb.ExecutionPayload)
		require.Equal(t, true, ok)

		jsonPayload := make(map[string]interface{})
		tx := gethtypes.NewTransaction(
			0,
			common.HexToAddress("095e7baea6a6c7c4c2dfeb977efac326af552d87"),
			big.NewInt(0), 0, big.NewInt(0),
			nil,
		)
		txs := []*gethtypes.Transaction{tx}
		encodedBinaryTxs := make([][]byte, 1)
		var err error
		encodedBinaryTxs[0], err = txs[0].MarshalBinary()
		require.NoError(t, err)
		payload.Transactions = encodedBinaryTxs
		jsonPayload["transactions"] = txs
		num := big.NewInt(1)
		encodedNum := hexutil.EncodeBig(num)
		jsonPayload["hash"] = hexutil.Encode(payload.BlockHash)
		jsonPayload["parentHash"] = common.BytesToHash([]byte("parent"))
		jsonPayload["sha3Uncles"] = common.BytesToHash([]byte("uncles"))
		jsonPayload["miner"] = common.BytesToAddress([]byte("miner"))
		jsonPayload["stateRoot"] = common.BytesToHash([]byte("state"))
		jsonPayload["transactionsRoot"] = common.BytesToHash([]byte("txs"))
		jsonPayload["receiptsRoot"] = common.BytesToHash([]byte("receipts"))
		jsonPayload["logsBloom"] = gethtypes.BytesToBloom([]byte("bloom"))
		jsonPayload["gasLimit"] = hexutil.EncodeUint64(1)
		jsonPayload["gasUsed"] = hexutil.EncodeUint64(2)
		jsonPayload["timestamp"] = hexutil.EncodeUint64(3)
		jsonPayload["number"] = encodedNum
		jsonPayload["extraData"] = common.BytesToHash([]byte("extra"))
		jsonPayload["totalDifficulty"] = "0x123456"
		jsonPayload["difficulty"] = encodedNum
		jsonPayload["size"] = encodedNum
		jsonPayload["baseFeePerGas"] = encodedNum

		wrappedPayload, err := blocks.WrappedExecutionPayload(payload)
		require.NoError(t, err)
		header, err := blocks.PayloadToHeader(wrappedPayload)
		require.NoError(t, err)

		bellatrixBlock := util.NewBlindedBeaconBlockBellatrix()
		wanted := util.NewBeaconBlockBellatrix()
		wanted.Block.Slot = 1
		// Make sure block hash is the zero hash.
		bellatrixBlock.Block.Body.ExecutionPayloadHeader.BlockHash = make([]byte, 32)
		bellatrixBlock.Block.Slot = 1
		wrappedEmpty, err := blocks.NewSignedBeaconBlock(bellatrixBlock)
		require.NoError(t, err)
		wantedWrappedEmpty, err := blocks.NewSignedBeaconBlock(wanted)
		require.NoError(t, err)

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			defer func() {
				require.NoError(t, r.Body.Close())
			}()

			respJSON := []map[string]interface{}{
				{
					"jsonrpc": "2.0",
					"id":      1,
					"result":  jsonPayload,
				},
				{
					"jsonrpc": "2.0",
					"id":      2,
					"result":  jsonPayload,
				},
			}
			require.NoError(t, json.NewEncoder(w).Encode(respJSON))
			require.NoError(t, json.NewEncoder(w).Encode(respJSON))

		}))
		defer srv.Close()

		rpcClient, err := rpc.DialHTTP(srv.URL)
		require.NoError(t, err)
		defer rpcClient.Close()

		service := &Service{}
		service.rpcClient = rpcClient
		blindedBlock := util.NewBlindedBeaconBlockBellatrix()

		blindedBlock.Block.Body.ExecutionPayloadHeader = header
		wrapped, err := blocks.NewSignedBeaconBlock(blindedBlock)
		require.NoError(t, err)
		copiedWrapped, err := wrapped.Copy()
		require.NoError(t, err)

		reconstructed, err := service.ReconstructFullBellatrixBlockBatch(ctx, []interfaces.SignedBeaconBlock{wrappedEmpty, wrapped, copiedWrapped})
		require.NoError(t, err)

		// Make sure empty blocks are handled correctly
		require.DeepEqual(t, wantedWrappedEmpty, reconstructed[0])

		// Handle normal execution blocks correctly
		got, err := reconstructed[1].Block().Body().Execution()
		require.NoError(t, err)
		require.DeepEqual(t, payload, got.Proto())

		got, err = reconstructed[2].Block().Body().Execution()
		require.NoError(t, err)
		require.DeepEqual(t, payload, got.Proto())
	})
}

func TestServer_getPowBlockHashAtTerminalTotalDifficulty(t *testing.T) {
	tests := []struct {
		name                  string
		paramsTd              string
		currentPowBlock       *pb.ExecutionBlock
		parentPowBlock        *pb.ExecutionBlock
		errLatestExecutionBlk error
		wantTerminalBlockHash []byte
		wantExists            bool
		errString             string
	}{
		{
			name:      "config td overflows",
			paramsTd:  "1115792089237316195423570985008687907853269984665640564039457584007913129638912",
			errString: "could not convert terminal total difficulty to uint256",
		},
		{
			name:                  "could not get latest execution block",
			paramsTd:              "1",
			errLatestExecutionBlk: errors.New("blah"),
			errString:             "could not get latest execution block",
		},
		{
			name:      "nil latest execution block",
			paramsTd:  "1",
			errString: "latest execution block is nil",
		},
		{
			name:     "current execution block invalid TD",
			paramsTd: "1",
			currentPowBlock: &pb.ExecutionBlock{
				Hash:            common.BytesToHash([]byte("a")),
				TotalDifficulty: "1115792089237316195423570985008687907853269984665640564039457584007913129638912",
			},
			errString: "could not convert total difficulty to uint256",
		},
		{
			name:     "current execution block has zero hash parent",
			paramsTd: "2",
			currentPowBlock: &pb.ExecutionBlock{
				Hash: common.BytesToHash([]byte("a")),
				Header: gethtypes.Header{
					ParentHash: common.BytesToHash(params.BeaconConfig().ZeroHash[:]),
				},
				TotalDifficulty: "0x3",
			},
		},
		{
			name:     "could not get parent block",
			paramsTd: "2",
			currentPowBlock: &pb.ExecutionBlock{
				Hash: common.BytesToHash([]byte("a")),
				Header: gethtypes.Header{
					ParentHash: common.BytesToHash([]byte("b")),
				},
				TotalDifficulty: "0x3",
			},
			errString: "could not get parent execution block",
		},
		{
			name:     "parent execution block invalid TD",
			paramsTd: "2",
			currentPowBlock: &pb.ExecutionBlock{
				Hash: common.BytesToHash([]byte("a")),
				Header: gethtypes.Header{
					ParentHash: common.BytesToHash([]byte("b")),
				},
				TotalDifficulty: "0x3",
			},
			parentPowBlock: &pb.ExecutionBlock{
				Hash: common.BytesToHash([]byte("b")),
				Header: gethtypes.Header{
					ParentHash: common.BytesToHash([]byte("c")),
				},
				TotalDifficulty: "1",
			},
			errString: "could not convert total difficulty to uint256",
		},
		{
			name:     "happy case",
			paramsTd: "2",
			currentPowBlock: &pb.ExecutionBlock{
				Hash: common.BytesToHash([]byte("a")),
				Header: gethtypes.Header{
					ParentHash: common.BytesToHash([]byte("b")),
				},
				TotalDifficulty: "0x3",
			},
			parentPowBlock: &pb.ExecutionBlock{
				Hash: common.BytesToHash([]byte("b")),
				Header: gethtypes.Header{
					ParentHash: common.BytesToHash([]byte("c")),
				},
				TotalDifficulty: "0x1",
			},
			wantExists:            true,
			wantTerminalBlockHash: common.BytesToHash([]byte("a")).Bytes(),
		},
		{
			name:     "happy case, but invalid timestamp",
			paramsTd: "2",
			currentPowBlock: &pb.ExecutionBlock{
				Hash: common.BytesToHash([]byte("a")),
				Header: gethtypes.Header{
					ParentHash: common.BytesToHash([]byte("b")),
					Time:       1,
				},
				TotalDifficulty: "0x3",
			},
			parentPowBlock: &pb.ExecutionBlock{
				Hash: common.BytesToHash([]byte("b")),
				Header: gethtypes.Header{
					ParentHash: common.BytesToHash([]byte("c")),
				},
				TotalDifficulty: "0x1",
			},
		},
		{
			name:     "ttd not reached",
			paramsTd: "3",
			currentPowBlock: &pb.ExecutionBlock{
				Hash: common.BytesToHash([]byte("a")),
				Header: gethtypes.Header{
					ParentHash: common.BytesToHash([]byte("b")),
				},
				TotalDifficulty: "0x2",
			},
			parentPowBlock: &pb.ExecutionBlock{
				Hash: common.BytesToHash([]byte("b")),
				Header: gethtypes.Header{
					ParentHash: common.BytesToHash([]byte("c")),
				},
				TotalDifficulty: "0x1",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := params.BeaconConfig().Copy()
			cfg.TerminalTotalDifficulty = tt.paramsTd
			params.OverrideBeaconConfig(cfg)
			var m map[[32]byte]*pb.ExecutionBlock
			if tt.parentPowBlock != nil {
				m = map[[32]byte]*pb.ExecutionBlock{
					tt.parentPowBlock.Hash: tt.parentPowBlock,
				}
			}
			client := mocks.EngineClient{
				ErrLatestExecBlock: tt.errLatestExecutionBlk,
				ExecutionBlock:     tt.currentPowBlock,
				BlockByHashMap:     m,
			}
			b, e, err := client.GetTerminalBlockHash(context.Background(), 1)
			if tt.errString != "" {
				require.ErrorContains(t, tt.errString, err)
			} else {
				require.NoError(t, err)
				require.DeepEqual(t, tt.wantExists, e)
				require.DeepEqual(t, tt.wantTerminalBlockHash, b)
			}
		})
	}
}

func Test_tDStringToUint256(t *testing.T) {
	i, err := tDStringToUint256("0x0")
	require.NoError(t, err)
	require.DeepEqual(t, uint256.NewInt(0), i)

	i, err = tDStringToUint256("0x10000")
	require.NoError(t, err)
	require.DeepEqual(t, uint256.NewInt(65536), i)

	_, err = tDStringToUint256("100")
	require.ErrorContains(t, "hex string without 0x prefix", err)

	_, err = tDStringToUint256("0xzzzzzz")
	require.ErrorContains(t, "invalid hex string", err)

	_, err = tDStringToUint256("0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF" +
		"FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF")
	require.ErrorContains(t, "hex number > 256 bits", err)
}

func TestExchangeTransitionConfiguration(t *testing.T) {
	fix := fixtures()
	ctx := context.Background()
	t.Run("wrong terminal block hash", func(t *testing.T) {
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

		service := &Service{}
		service.rpcClient = rpcClient

		err = service.ExchangeTransitionConfiguration(ctx, request)
		require.Equal(t, true, errors.Is(err, ErrConfigMismatch))
	})
	t.Run("wrong terminal total difficulty", func(t *testing.T) {
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
			resp.TerminalTotalDifficulty = "0x1"
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

		service := &Service{}
		service.rpcClient = rpcClient

		err = service.ExchangeTransitionConfiguration(ctx, request)
		require.Equal(t, true, errors.Is(err, ErrConfigMismatch))
	})
}

type customError struct {
	code    int
	timeout bool
}

func (c *customError) ErrorCode() int {
	return c.code
}

func (*customError) Error() string {
	return "something went wrong"
}

func (c *customError) Timeout() bool {
	return c.timeout
}

type dataError struct {
	code int
	data interface{}
}

func (c *dataError) ErrorCode() int {
	return c.code
}

func (*dataError) Error() string {
	return "something went wrong"
}

func (c *dataError) ErrorData() interface{} {
	return c.data
}

func Test_handleRPCError(t *testing.T) {
	got := handleRPCError(nil)
	require.Equal(t, true, got == nil)

	var tests = []struct {
		name             string
		expected         error
		expectedContains string
		given            error
	}{
		{
			name:             "not an rpc error",
			expectedContains: "got an unexpected error",
			given:            errors.New("foo"),
		},
		{
			name:             "HTTP times out",
			expectedContains: ErrHTTPTimeout.Error(),
			given:            &customError{timeout: true},
		},
		{
			name:             "ErrParse",
			expectedContains: ErrParse.Error(),
			given:            &customError{code: -32700},
		},
		{
			name:             "ErrInvalidRequest",
			expectedContains: ErrInvalidRequest.Error(),
			given:            &customError{code: -32600},
		},
		{
			name:             "ErrMethodNotFound",
			expectedContains: ErrMethodNotFound.Error(),
			given:            &customError{code: -32601},
		},
		{
			name:             "ErrInvalidParams",
			expectedContains: ErrInvalidParams.Error(),
			given:            &customError{code: -32602},
		},
		{
			name:             "ErrInternal",
			expectedContains: ErrInternal.Error(),
			given:            &customError{code: -32603},
		},
		{
			name:             "ErrUnknownPayload",
			expectedContains: ErrUnknownPayload.Error(),
			given:            &customError{code: -38001},
		},
		{
			name:             "ErrInvalidForkchoiceState",
			expectedContains: ErrInvalidForkchoiceState.Error(),
			given:            &customError{code: -38002},
		},
		{
			name:             "ErrInvalidPayloadAttributes",
			expectedContains: ErrInvalidPayloadAttributes.Error(),
			given:            &customError{code: -38003},
		},
		{
			name:             "ErrServer unexpected no data",
			expectedContains: "got an unexpected error",
			given:            &customError{code: -32000},
		},
		{
			name:             "ErrServer with data",
			expectedContains: ErrServer.Error(),
			given:            &dataError{code: -32000, data: 5},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := handleRPCError(tt.given)
			require.ErrorContains(t, tt.expectedContains, got)
		})
	}
}

func newTestIPCServer(t *testing.T) *rpc.Server {
	server := rpc.NewServer()
	err := server.RegisterName("engine", new(testEngineService))
	require.NoError(t, err)
	err = server.RegisterName("eth", new(testEngineService))
	require.NoError(t, err)
	return server
}

func fixtures() map[string]interface{} {
	foo := bytesutil.ToBytes32([]byte("foo"))
	bar := bytesutil.PadTo([]byte("bar"), 20)
	baz := bytesutil.PadTo([]byte("baz"), 256)
	baseFeePerGas := big.NewInt(12345)
	executionPayloadFixture := &pb.ExecutionPayload{
		ParentHash:    foo[:],
		FeeRecipient:  bar,
		StateRoot:     foo[:],
		ReceiptsRoot:  foo[:],
		LogsBloom:     baz,
		PrevRandao:    foo[:],
		BlockNumber:   1,
		GasLimit:      1,
		GasUsed:       1,
		Timestamp:     1,
		ExtraData:     foo[:],
		BaseFeePerGas: bytesutil.PadTo(baseFeePerGas.Bytes(), fieldparams.RootLength),
		BlockHash:     foo[:],
		Transactions:  [][]byte{foo[:]},
	}
	parent := bytesutil.PadTo([]byte("parentHash"), fieldparams.RootLength)
	sha3Uncles := bytesutil.PadTo([]byte("sha3Uncles"), fieldparams.RootLength)
	miner := bytesutil.PadTo([]byte("miner"), fieldparams.FeeRecipientLength)
	stateRoot := bytesutil.PadTo([]byte("stateRoot"), fieldparams.RootLength)
	transactionsRoot := bytesutil.PadTo([]byte("transactionsRoot"), fieldparams.RootLength)
	receiptsRoot := bytesutil.PadTo([]byte("receiptsRoot"), fieldparams.RootLength)
	logsBloom := bytesutil.PadTo([]byte("logs"), fieldparams.LogsBloomLength)
	executionBlock := &pb.ExecutionBlock{
		Header: gethtypes.Header{
			ParentHash:  common.BytesToHash(parent),
			UncleHash:   common.BytesToHash(sha3Uncles),
			Coinbase:    common.BytesToAddress(miner),
			Root:        common.BytesToHash(stateRoot),
			TxHash:      common.BytesToHash(transactionsRoot),
			ReceiptHash: common.BytesToHash(receiptsRoot),
			Bloom:       gethtypes.BytesToBloom(logsBloom),
			Difficulty:  big.NewInt(1),
			Number:      big.NewInt(2),
			GasLimit:    3,
			GasUsed:     4,
			Time:        5,
			Extra:       []byte("extra"),
			MixDigest:   common.BytesToHash([]byte("mix")),
			Nonce:       gethtypes.EncodeNonce(6),
			BaseFee:     big.NewInt(7),
		},
	}
	status := &pb.PayloadStatus{
		Status:          pb.PayloadStatus_VALID,
		LatestValidHash: foo[:],
		ValidationError: "",
	}
	id := pb.PayloadIDBytes([8]byte{1, 0, 0, 0, 0, 0, 0, 0})
	forkChoiceResp := &ForkchoiceUpdatedResponse{
		Status:    status,
		PayloadId: &id,
	}
	forkChoiceSyncingResp := &ForkchoiceUpdatedResponse{
		Status: &pb.PayloadStatus{
			Status:          pb.PayloadStatus_SYNCING,
			LatestValidHash: nil,
		},
		PayloadId: &id,
	}
	forkChoiceAcceptedResp := &ForkchoiceUpdatedResponse{
		Status: &pb.PayloadStatus{
			Status:          pb.PayloadStatus_ACCEPTED,
			LatestValidHash: nil,
		},
		PayloadId: &id,
	}
	forkChoiceInvalidResp := &ForkchoiceUpdatedResponse{
		Status: &pb.PayloadStatus{
			Status:          pb.PayloadStatus_INVALID,
			LatestValidHash: bytesutil.PadTo([]byte("latestValidHash"), 32),
		},
		PayloadId: &id,
	}
	b, _ := new(big.Int).SetString(params.BeaconConfig().TerminalTotalDifficulty, 10)
	ttd, _ := uint256.FromBig(b)
	transitionCfg := &pb.TransitionConfiguration{
		TerminalBlockHash:       params.BeaconConfig().TerminalBlockHash[:],
		TerminalTotalDifficulty: ttd.Hex(),
		TerminalBlockNumber:     big.NewInt(0).Bytes(),
	}
	validStatus := &pb.PayloadStatus{
		Status:          pb.PayloadStatus_VALID,
		LatestValidHash: foo[:],
		ValidationError: "",
	}
	inValidBlockHashStatus := &pb.PayloadStatus{
		Status:          pb.PayloadStatus_INVALID_BLOCK_HASH,
		LatestValidHash: nil,
	}
	acceptedStatus := &pb.PayloadStatus{
		Status:          pb.PayloadStatus_ACCEPTED,
		LatestValidHash: nil,
	}
	syncingStatus := &pb.PayloadStatus{
		Status:          pb.PayloadStatus_SYNCING,
		LatestValidHash: nil,
	}
	invalidStatus := &pb.PayloadStatus{
		Status:          pb.PayloadStatus_INVALID,
		LatestValidHash: foo[:],
	}
	unknownStatus := &pb.PayloadStatus{
		Status:          pb.PayloadStatus_UNKNOWN,
		LatestValidHash: foo[:],
	}
	return map[string]interface{}{
		"ExecutionBlock":                    executionBlock,
		"ExecutionPayload":                  executionPayloadFixture,
		"ValidPayloadStatus":                validStatus,
		"InvalidBlockHashStatus":            inValidBlockHashStatus,
		"AcceptedStatus":                    acceptedStatus,
		"SyncingStatus":                     syncingStatus,
		"InvalidStatus":                     invalidStatus,
		"UnknownStatus":                     unknownStatus,
		"ForkchoiceUpdatedResponse":         forkChoiceResp,
		"ForkchoiceUpdatedSyncingResponse":  forkChoiceSyncingResp,
		"ForkchoiceUpdatedAcceptedResponse": forkChoiceAcceptedResp,
		"ForkchoiceUpdatedInvalidResponse":  forkChoiceInvalidResp,
		"TransitionConfiguration":           transitionCfg,
	}
}

func Test_fullPayloadFromExecutionBlock(t *testing.T) {
	type args struct {
		header *pb.ExecutionPayloadHeader
		block  *pb.ExecutionBlock
	}
	wantedHash := common.BytesToHash([]byte("foo"))
	tests := []struct {
		name string
		args args
		want *pb.ExecutionPayload
		err  string
	}{
		{
			name: "block hash field in header and block hash mismatch",
			args: args{
				header: &pb.ExecutionPayloadHeader{
					BlockHash: []byte("foo"),
				},
				block: &pb.ExecutionBlock{
					Hash: common.BytesToHash([]byte("bar")),
				},
			},
			err: "does not match execution block hash",
		},
		{
			name: "ok",
			args: args{
				header: &pb.ExecutionPayloadHeader{
					BlockHash: wantedHash[:],
				},
				block: &pb.ExecutionBlock{
					Hash: wantedHash,
				},
			},
			want: &pb.ExecutionPayload{
				BlockHash: wantedHash[:],
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wrapped, err := blocks.WrappedExecutionPayloadHeader(tt.args.header)
			got, err := fullPayloadFromExecutionBlock(wrapped, tt.args.block)
			if (err != nil) && !strings.Contains(err.Error(), tt.err) {
				t.Fatalf("Wanted err %s got %v", tt.err, err)
			}
			require.DeepEqual(t, tt.want, got)
		})
	}
}

type testEngineService struct{}

func (*testEngineService) NoArgsRets() {}

func (*testEngineService) GetBlockByHash(
	_ context.Context, _ common.Hash, _ bool,
) *pb.ExecutionBlock {
	fix := fixtures()
	item, ok := fix["ExecutionBlock"].(*pb.ExecutionBlock)
	if !ok {
		panic("not found")
	}
	return item
}

func (*testEngineService) GetBlockByNumber(
	_ context.Context, _ string, _ bool,
) *pb.ExecutionBlock {
	fix := fixtures()
	item, ok := fix["ExecutionBlock"].(*pb.ExecutionBlock)
	if !ok {
		panic("not found")
	}
	return item
}

func (*testEngineService) GetPayloadV1(
	_ context.Context, _ pb.PayloadIDBytes,
) *pb.ExecutionPayload {
	fix := fixtures()
	item, ok := fix["ExecutionPayload"].(*pb.ExecutionPayload)
	if !ok {
		panic("not found")
	}
	return item
}

func (*testEngineService) ExchangeTransitionConfigurationV1(
	_ context.Context, _ *pb.TransitionConfiguration,
) *pb.TransitionConfiguration {
	fix := fixtures()
	item, ok := fix["TransitionConfiguration"].(*pb.TransitionConfiguration)
	if !ok {
		panic("not found")
	}
	return item
}

func (*testEngineService) ForkchoiceUpdatedV1(
	_ context.Context, _ *pb.ForkchoiceState, _ *pb.PayloadAttributes,
) *ForkchoiceUpdatedResponse {
	fix := fixtures()
	item, ok := fix["ForkchoiceUpdatedResponse"].(*ForkchoiceUpdatedResponse)
	if !ok {
		panic("not found")
	}
	item.Status.Status = pb.PayloadStatus_VALID
	return item
}

func (*testEngineService) NewPayloadV1(
	_ context.Context, _ *pb.ExecutionPayload,
) *pb.PayloadStatus {
	fix := fixtures()
	item, ok := fix["ValidPayloadStatus"].(*pb.PayloadStatus)
	if !ok {
		panic("not found")
	}
	return item
}

func forkchoiceUpdateSetup(t *testing.T, fcs *pb.ForkchoiceState, att *pb.PayloadAttributes, res *ForkchoiceUpdatedResponse) *Service {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		defer func() {
			require.NoError(t, r.Body.Close())
		}()
		enc, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		jsonRequestString := string(enc)

		forkChoiceStateReq, err := json.Marshal(fcs)
		require.NoError(t, err)
		payloadAttrsReq, err := json.Marshal(att)
		require.NoError(t, err)

		// We expect the JSON string RPC request contains the right arguments.
		require.Equal(t, true, strings.Contains(
			jsonRequestString, string(forkChoiceStateReq),
		))
		require.Equal(t, true, strings.Contains(
			jsonRequestString, string(payloadAttrsReq),
		))
		resp := map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      1,
			"result":  res,
		}
		err = json.NewEncoder(w).Encode(resp)
		require.NoError(t, err)
	}))

	rpcClient, err := rpc.DialHTTP(srv.URL)
	require.NoError(t, err)

	service := &Service{}
	service.rpcClient = rpcClient
	return service
}

func newPayloadSetup(t *testing.T, status *pb.PayloadStatus, payload *pb.ExecutionPayload) *Service {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		defer func() {
			require.NoError(t, r.Body.Close())
		}()
		enc, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		jsonRequestString := string(enc)

		reqArg, err := json.Marshal(payload)
		require.NoError(t, err)

		// We expect the JSON string RPC request contains the right arguments.
		require.Equal(t, true, strings.Contains(
			jsonRequestString, string(reqArg),
		))
		resp := map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      1,
			"result":  status,
		}
		err = json.NewEncoder(w).Encode(resp)
		require.NoError(t, err)
	}))

	rpcClient, err := rpc.DialHTTP(srv.URL)
	require.NoError(t, err)

	service := &Service{}
	service.rpcClient = rpcClient
	return service
}
