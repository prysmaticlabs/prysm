package v1

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/pkg/errors"
	fieldparams "github.com/prysmaticlabs/prysm/config/fieldparams"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	pb "github.com/prysmaticlabs/prysm/proto/engine/v1"
	"github.com/prysmaticlabs/prysm/testing/require"
	"google.golang.org/protobuf/proto"
)

var _ = Caller(&Client{})

func TestClient_IPC(t *testing.T) {
	server := newTestIPCServer(t)
	defer server.Stop()
	rpcClient := rpc.DialInProc(server)
	defer rpcClient.Close()
	client := &Client{}
	client.rpc = rpcClient
	ctx := context.Background()
	fix := fixtures()

	t.Run(GetPayloadMethod, func(t *testing.T) {
		want, ok := fix["ExecutionPayload"].(*pb.ExecutionPayload)
		require.Equal(t, true, ok)
		payloadId := [8]byte{1}
		resp, err := client.GetPayload(ctx, payloadId)
		require.NoError(t, err)
		require.DeepEqual(t, want, resp)
	})
	t.Run(ForkchoiceUpdatedMethod, func(t *testing.T) {
		want, ok := fix["ForkchoiceUpdatedResponse"].(*ForkchoiceUpdatedResponse)
		require.Equal(t, true, ok)
		payloadID, validHash, err := client.ForkchoiceUpdated(ctx, &pb.ForkchoiceState{}, &pb.PayloadAttributes{})
		require.NoError(t, err)
		require.DeepEqual(t, want.Status.LatestValidHash, validHash)
		require.DeepEqual(t, want.PayloadId, payloadID)
	})
	t.Run(NewPayloadMethod, func(t *testing.T) {
		want, ok := fix["ValidPayloadStatus"].(*pb.PayloadStatus)
		require.Equal(t, true, ok)
		req, ok := fix["ExecutionPayload"].(*pb.ExecutionPayload)
		require.Equal(t, true, ok)
		latestValidHash, err := client.NewPayload(ctx, req)
		require.NoError(t, err)
		require.DeepEqual(t, bytesutil.ToBytes32(want.LatestValidHash), bytesutil.ToBytes32(latestValidHash))
	})
	t.Run(ExchangeTransitionConfigurationMethod, func(t *testing.T) {
		want, ok := fix["TransitionConfiguration"].(*pb.TransitionConfiguration)
		require.Equal(t, true, ok)
		err := client.ExchangeTransitionConfiguration(ctx, want)
		require.NoError(t, err)
	})
	t.Run(ExecutionBlockByNumberMethod, func(t *testing.T) {
		want, ok := fix["ExecutionBlock"].(*pb.ExecutionBlock)
		require.Equal(t, true, ok)
		resp, err := client.LatestExecutionBlock(ctx)
		require.NoError(t, err)
		require.DeepEqual(t, want, resp)
	})
	t.Run(ExecutionBlockByHashMethod, func(t *testing.T) {
		want, ok := fix["ExecutionBlock"].(*pb.ExecutionBlock)
		require.Equal(t, true, ok)
		arg := common.BytesToHash([]byte("foo"))
		resp, err := client.ExecutionBlockByHash(ctx, arg)
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
			enc, err := ioutil.ReadAll(r.Body)
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

		client := &Client{}
		client.rpc = rpcClient

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
			Random:                []byte("random"),
			SuggestedFeeRecipient: []byte("suggestedFeeRecipient"),
		}
		want, ok := fix["ForkchoiceUpdatedResponse"].(*ForkchoiceUpdatedResponse)
		require.Equal(t, true, ok)
		client := forkchoiceUpdateSetup(t, forkChoiceState, payloadAttributes, want)

		// We call the RPC method via HTTP and expect a proper result.
		payloadID, validHash, err := client.ForkchoiceUpdated(ctx, forkChoiceState, payloadAttributes)
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
			Random:                []byte("random"),
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
			Random:                []byte("random"),
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
			Random:                []byte("random"),
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
	t.Run(ForkchoiceUpdatedMethod+" INVALID_TERMINAL_BLOCK status", func(t *testing.T) {
		forkChoiceState := &pb.ForkchoiceState{
			HeadBlockHash:      []byte("head"),
			SafeBlockHash:      []byte("safe"),
			FinalizedBlockHash: []byte("finalized"),
		}
		payloadAttributes := &pb.PayloadAttributes{
			Timestamp:             1,
			Random:                []byte("random"),
			SuggestedFeeRecipient: []byte("suggestedFeeRecipient"),
		}
		want, ok := fix["ForkchoiceUpdatedInvalidTerminalBlockResponse"].(*ForkchoiceUpdatedResponse)
		require.Equal(t, true, ok)
		client := forkchoiceUpdateSetup(t, forkChoiceState, payloadAttributes, want)

		// We call the RPC method via HTTP and expect a proper result.
		payloadID, validHash, err := client.ForkchoiceUpdated(ctx, forkChoiceState, payloadAttributes)
		require.ErrorContains(t, "could not satisfy terminal block condition", err)
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
		resp, err := client.NewPayload(ctx, execPayload)
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
		resp, err := client.NewPayload(ctx, execPayload)
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
		resp, err := client.NewPayload(ctx, execPayload)
		require.ErrorContains(t, "could not validate block hash", err)
		require.DeepEqual(t, []uint8(nil), resp)
	})
	t.Run(NewPayloadMethod+" INVALID_TERMINAL_BLOCK status", func(t *testing.T) {
		execPayload, ok := fix["ExecutionPayload"].(*pb.ExecutionPayload)
		require.Equal(t, true, ok)
		want, ok := fix["InvalidTerminalBlockStatus"].(*pb.PayloadStatus)
		require.Equal(t, true, ok)
		client := newPayloadSetup(t, want, execPayload)

		// We call the RPC method via HTTP and expect a proper result.
		resp, err := client.NewPayload(ctx, execPayload)
		require.ErrorContains(t, "could not satisfy terminal block condition", err)
		require.DeepEqual(t, []uint8(nil), resp)
	})
	t.Run(NewPayloadMethod+" INVALID status", func(t *testing.T) {
		execPayload, ok := fix["ExecutionPayload"].(*pb.ExecutionPayload)
		require.Equal(t, true, ok)
		want, ok := fix["InvalidStatus"].(*pb.PayloadStatus)
		require.Equal(t, true, ok)
		client := newPayloadSetup(t, want, execPayload)

		// We call the RPC method via HTTP and expect a proper result.
		resp, err := client.NewPayload(ctx, execPayload)
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
		resp, err := client.NewPayload(ctx, execPayload)
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

		client := &Client{}
		client.rpc = rpcClient

		// We call the RPC method via HTTP and expect a proper result.
		resp, err := client.LatestExecutionBlock(ctx)
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
			enc, err := ioutil.ReadAll(r.Body)
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

		client := &Client{}
		client.rpc = rpcClient

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
			enc, err := ioutil.ReadAll(r.Body)
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

		client := &Client{}
		client.rpc = rpcClient

		// We call the RPC method via HTTP and expect a proper result.
		resp, err := client.ExecutionBlockByHash(ctx, arg)
		require.NoError(t, err)
		require.DeepEqual(t, want, resp)
	})
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

		client := &Client{}
		client.rpc = rpcClient

		err = client.ExchangeTransitionConfiguration(ctx, request)
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
			resp.TerminalTotalDifficulty = "bar"
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

		client := &Client{}
		client.rpc = rpcClient

		err = client.ExchangeTransitionConfiguration(ctx, request)
		require.Equal(t, true, errors.Is(err, ErrConfigMismatch))
	})
}

type customError struct {
	code int
}

func (c *customError) ErrorCode() int {
	return c.code
}

func (*customError) Error() string {
	return "something went wrong"
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
			given:            &customError{code: -32001},
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
	baseFeePerGas := big.NewInt(6)
	executionPayloadFixture := &pb.ExecutionPayload{
		ParentHash:    foo[:],
		FeeRecipient:  bar,
		StateRoot:     foo[:],
		ReceiptsRoot:  foo[:],
		LogsBloom:     baz,
		Random:        foo[:],
		BlockNumber:   1,
		GasLimit:      1,
		GasUsed:       1,
		Timestamp:     1,
		ExtraData:     foo[:],
		BaseFeePerGas: bytesutil.PadTo(baseFeePerGas.Bytes(), fieldparams.RootLength),
		BlockHash:     foo[:],
		Transactions:  [][]byte{foo[:]},
	}
	number := bytesutil.PadTo([]byte("100"), fieldparams.RootLength)
	hash := bytesutil.PadTo([]byte("hash"), fieldparams.RootLength)
	parent := bytesutil.PadTo([]byte("parentHash"), fieldparams.RootLength)
	sha3Uncles := bytesutil.PadTo([]byte("sha3Uncles"), fieldparams.RootLength)
	miner := bytesutil.PadTo([]byte("miner"), fieldparams.FeeRecipientLength)
	stateRoot := bytesutil.PadTo([]byte("stateRoot"), fieldparams.RootLength)
	transactionsRoot := bytesutil.PadTo([]byte("transactionsRoot"), fieldparams.RootLength)
	receiptsRoot := bytesutil.PadTo([]byte("receiptsRoot"), fieldparams.RootLength)
	logsBloom := bytesutil.PadTo([]byte("logs"), fieldparams.LogsBloomLength)
	executionBlock := &pb.ExecutionBlock{
		Number:           number,
		Hash:             hash,
		ParentHash:       parent,
		Sha3Uncles:       sha3Uncles,
		Miner:            miner,
		StateRoot:        stateRoot,
		TransactionsRoot: transactionsRoot,
		ReceiptsRoot:     receiptsRoot,
		LogsBloom:        logsBloom,
		Difficulty:       bytesutil.PadTo([]byte("1"), fieldparams.RootLength),
		TotalDifficulty:  "2",
		GasLimit:         3,
		GasUsed:          4,
		Timestamp:        5,
		Size:             bytesutil.PadTo([]byte("6"), fieldparams.RootLength),
		ExtraData:        bytesutil.PadTo([]byte("extraData"), fieldparams.RootLength),
		BaseFeePerGas:    bytesutil.PadTo([]byte("baseFeePerGas"), fieldparams.RootLength),
		Transactions:     [][]byte{foo[:]},
		Uncles:           [][]byte{foo[:]},
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
	forkChoiceInvalidTerminalBlockResp := &ForkchoiceUpdatedResponse{
		Status: &pb.PayloadStatus{
			Status:          pb.PayloadStatus_INVALID_TERMINAL_BLOCK,
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
			LatestValidHash: []byte("latestValidHash"),
		},
		PayloadId: &id,
	}
	transitionCfg := &pb.TransitionConfiguration{
		TerminalBlockHash:       params.BeaconConfig().TerminalBlockHash[:],
		TerminalTotalDifficulty: params.BeaconConfig().TerminalTotalDifficulty,
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
	inValidTerminalBlockStatus := &pb.PayloadStatus{
		Status:          pb.PayloadStatus_INVALID_TERMINAL_BLOCK,
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
		"ExecutionBlock":                                executionBlock,
		"ExecutionPayload":                              executionPayloadFixture,
		"ValidPayloadStatus":                            validStatus,
		"InvalidBlockHashStatus":                        inValidBlockHashStatus,
		"InvalidTerminalBlockStatus":                    inValidTerminalBlockStatus,
		"AcceptedStatus":                                acceptedStatus,
		"SyncingStatus":                                 syncingStatus,
		"InvalidStatus":                                 invalidStatus,
		"UnknownStatus":                                 unknownStatus,
		"ForkchoiceUpdatedResponse":                     forkChoiceResp,
		"ForkchoiceUpdatedSyncingResponse":              forkChoiceSyncingResp,
		"ForkchoiceUpdatedInvalidTerminalBlockResponse": forkChoiceInvalidTerminalBlockResp,
		"ForkchoiceUpdatedAcceptedResponse":             forkChoiceAcceptedResp,
		"ForkchoiceUpdatedInvalidResponse":              forkChoiceInvalidResp,
		"TransitionConfiguration":                       transitionCfg,
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

func forkchoiceUpdateSetup(t *testing.T, fcs *pb.ForkchoiceState, att *pb.PayloadAttributes, res *ForkchoiceUpdatedResponse) *Client {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		defer func() {
			require.NoError(t, r.Body.Close())
		}()
		enc, err := ioutil.ReadAll(r.Body)
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

	client := &Client{}
	client.rpc = rpcClient

	return client
}

func newPayloadSetup(t *testing.T, status *pb.PayloadStatus, payload *pb.ExecutionPayload) *Client {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		defer func() {
			require.NoError(t, r.Body.Close())
		}()
		enc, err := ioutil.ReadAll(r.Body)
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

	client := &Client{}
	client.rpc = rpcClient
	return client
}
