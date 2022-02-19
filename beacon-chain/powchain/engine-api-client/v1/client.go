// Package v1 defines an API client for the engine API defined in https://github.com/ethereum/execution-apis.
// This client is used for the Prysm consensus node to connect to execution node as part of
// the Ethereum proof-of-stake machinery.
package v1

import (
	"bytes"
	"context"
	"math/big"
	"net/url"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/config/params"
	pb "github.com/prysmaticlabs/prysm/proto/engine/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
)

const (
	// NewPayloadMethod v1 request string for JSON-RPC.
	NewPayloadMethod = "engine_newPayloadV1"
	// ForkchoiceUpdatedMethod v1 request string for JSON-RPC.
	ForkchoiceUpdatedMethod = "engine_forkchoiceUpdatedV1"
	// GetPayloadMethod v1 request string for JSON-RPC.
	GetPayloadMethod = "engine_getPayloadV1"
	// GetBlobsMethod v1 request string for JSON-RPC.
	GetBlobsMethod = "engine_getBlobsV1"
	// ExchangeTransitionConfigurationMethod v1 request string for JSON-RPC.
	ExchangeTransitionConfigurationMethod = "engine_exchangeTransitionConfigurationV1"
	// ExecutionBlockByHashMethod request string for JSON-RPC.
	ExecutionBlockByHashMethod = "eth_getBlockByHash"
	// ExecutionBlockByNumberMethod request string for JSON-RPC.
	ExecutionBlockByNumberMethod = "eth_getBlockByNumber"
	// DefaultTimeout for HTTP.
	DefaultTimeout = time.Second * 5
)

// ForkchoiceUpdatedResponse is the response kind received by the
// engine_forkchoiceUpdatedV1 endpoint.
type ForkchoiceUpdatedResponse struct {
	Status    *pb.PayloadStatus  `json:"payloadStatus"`
	PayloadId *pb.PayloadIDBytes `json:"payloadId"`
}

// EngineCaller defines a client that can interact with an Ethereum
// execution node's engine service via JSON-RPC.
type EngineCaller interface {
	NewPayload(ctx context.Context, payload *pb.ExecutionPayload) (*pb.PayloadStatus, error)
	ForkchoiceUpdated(
		ctx context.Context, state *pb.ForkchoiceState, attrs *pb.PayloadAttributes,
	) (*ForkchoiceUpdatedResponse, error)
	GetPayload(ctx context.Context, payloadId [8]byte) (*pb.ExecutionPayload, error)
	GetBlobs(ctx context.Context, payloadId [8]byte) (*ethpb.Blob, error)
	ExchangeTransitionConfiguration(
		ctx context.Context, cfg *pb.TransitionConfiguration,
	) (*pb.TransitionConfiguration, error)
	LatestExecutionBlock(ctx context.Context) (*pb.ExecutionBlock, error)
	ExecutionBlockByHash(ctx context.Context, hash common.Hash) (*pb.ExecutionBlock, error)
}

// Client defines a new engine API client for the Prysm consensus node
// to interact with an Ethereum execution node.
type Client struct {
	cfg *config
	rpc *rpc.Client
}

// New returns a ready, engine API client from an endpoint and configuration options.
// Only http(s) and ipc (inter-process communication) URL schemes are supported.
func New(ctx context.Context, endpoint string, opts ...Option) (*Client, error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, err
	}
	c := &Client{
		cfg: defaultConfig(),
	}
	switch u.Scheme {
	case "http", "https":
		c.rpc, err = rpc.DialHTTPWithClient(endpoint, c.cfg.httpClient)
	case "":
		c.rpc, err = rpc.DialIPC(ctx, endpoint)
	default:
		return nil, errors.Wrapf(ErrUnsupportedScheme, "%q", u.Scheme)
	}
	if err != nil {
		return nil, err
	}
	for _, opt := range opts {
		if err := opt(c); err != nil {
			return nil, err
		}
	}
	return c, nil
}

// NewPayload calls the engine_newPayloadV1 method via JSON-RPC.
func (c *Client) NewPayload(ctx context.Context, payload *pb.ExecutionPayload) (*pb.PayloadStatus, error) {
	result := &pb.PayloadStatus{}
	err := c.rpc.CallContext(ctx, result, NewPayloadMethod, payload)
	return result, handleRPCError(err)
}

// ForkchoiceUpdated calls the engine_forkchoiceUpdatedV1 method via JSON-RPC.
func (c *Client) ForkchoiceUpdated(
	ctx context.Context, state *pb.ForkchoiceState, attrs *pb.PayloadAttributes,
) (*ForkchoiceUpdatedResponse, error) {
	result := &ForkchoiceUpdatedResponse{}
	err := c.rpc.CallContext(ctx, result, ForkchoiceUpdatedMethod, state, attrs)
	return result, handleRPCError(err)
}

// GetPayload calls the engine_getPayloadV1 method via JSON-RPC.
func (c *Client) GetPayload(ctx context.Context, payloadId [8]byte) (*pb.ExecutionPayload, error) {
	result := &pb.ExecutionPayload{}
	err := c.rpc.CallContext(ctx, result, GetPayloadMethod, pb.PayloadIDBytes(payloadId))
	if err != nil {
		return nil, err
	}
	// TODO: Use real transactions here.
	mockTxs, err := mockBlobTransactions(256)
	if err != nil {
		return nil, err
	}
	result.Transactions = mockTxs
	return result, handleRPCError(err)
}

// GetBlobs calls the engine_getBlobsV1 method via JSON-RPC.
func (c *Client) GetBlobs(ctx context.Context, payloadId [8]byte) (*ethpb.Blob, error) {
	result := &ethpb.Blob{
		Blob: make([][]byte, 0), // TODO: Mock blobs.
	}
	return result, nil
}

// ExchangeTransitionConfiguration calls the engine_exchangeTransitionConfigurationV1 method via JSON-RPC.
func (c *Client) ExchangeTransitionConfiguration(
	ctx context.Context, cfg *pb.TransitionConfiguration,
) (*pb.TransitionConfiguration, error) {
	// Terminal block number should be set to 0
	zeroBigNum := big.NewInt(0)
	cfg.TerminalBlockNumber = zeroBigNum.Bytes()
	result := &pb.TransitionConfiguration{}
	if err := c.rpc.CallContext(ctx, result, ExchangeTransitionConfigurationMethod, cfg); err != nil {
		return nil, handleRPCError(err)
	}
	// We surface an error to the user if local configuration settings mismatch
	// according to the response from the execution node.
	cfgTerminalHash := params.BeaconConfig().TerminalBlockHash[:]
	if !bytes.Equal(cfgTerminalHash, result.TerminalBlockHash) {
		return nil, errors.Wrapf(
			ErrMismatchTerminalBlockHash,
			"got %#x from execution node, wanted %#x",
			result.TerminalBlockHash,
			cfgTerminalHash,
		)
	}
	ttdCfg := params.BeaconConfig().TerminalTotalDifficulty
	if ttdCfg != result.TerminalTotalDifficulty {
		return nil, errors.Wrapf(
			ErrMismatchTerminalTotalDiff,
			"got %s from execution node, wanted %s",
			result.TerminalTotalDifficulty,
			ttdCfg,
		)
	}
	return result, nil
}

// LatestExecutionBlock fetches the latest execution engine block by calling
// eth_blockByNumber via JSON-RPC.
func (c *Client) LatestExecutionBlock(ctx context.Context) (*pb.ExecutionBlock, error) {
	result := &pb.ExecutionBlock{}
	err := c.rpc.CallContext(
		ctx,
		result,
		ExecutionBlockByNumberMethod,
		"latest",
		false, /* no full transaction objects */
	)
	return result, handleRPCError(err)
}

// ExecutionBlockByHash fetches an execution engine block by hash by calling
// eth_blockByHash via JSON-RPC.
func (c *Client) ExecutionBlockByHash(ctx context.Context, hash common.Hash) (*pb.ExecutionBlock, error) {
	result := &pb.ExecutionBlock{}
	err := c.rpc.CallContext(ctx, result, ExecutionBlockByHashMethod, hash, false /* no full transaction objects */)
	return result, handleRPCError(err)
}

// Returns a list of SSZ-encoded,
func mockBlobTransactions(numItems uint64) ([][]byte, error) {
	blobTxs := make([]*pb.SignedBlobTransaction, numItems)
	foo := [32]byte{}
	addr := [20]byte{}
	for i := uint64(0); i < numItems; i++ {
		blobTxs[i] = &pb.SignedBlobTransaction{
			Header: &pb.BlobTransaction{
				Nonce:               i,
				Gas:                 1,
				MaxBasefee:          foo[:],
				PriorityFee:         foo[:],
				Address:             addr[:],
				Value:               foo[:],
				Data:                []byte("foo"),
				BlobVersionedHashes: nil,
			},
			Signatures: &pb.ECDSASignature{
				V: []byte{1},
				R: make([]byte, 32),
				S: make([]byte, 32),
			},
		}
	}
	txs := make([][]byte, 0) // TODO: Add some mock txs.
	return txs, nil
}

// Handles errors received from the RPC server according to the specification.
func handleRPCError(err error) error {
	if err == nil {
		return nil
	}
	e, ok := err.(rpc.Error)
	if !ok {
		return errors.Wrap(err, "got an unexpected error")
	}
	switch e.ErrorCode() {
	case -32700:
		return ErrParse
	case -32600:
		return ErrInvalidRequest
	case -32601:
		return ErrMethodNotFound
	case -32602:
		return ErrInvalidParams
	case -32603:
		return ErrInternal
	case -32001:
		return ErrUnknownPayload
	case -32000:
		// Only -32000 status codes are data errors in the RPC specification.
		errWithData, ok := err.(rpc.DataError)
		if !ok {
			return errors.Wrap(err, "got an unexpected error")
		}
		return errors.Wrapf(ErrServer, "%v", errWithData.ErrorData())
	default:
		return err
	}
}
