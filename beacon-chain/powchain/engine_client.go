package powchain

import (
	"bytes"
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/config/params"
	pb "github.com/prysmaticlabs/prysm/proto/engine/v1"
	"go.opencensus.io/trace"
)

const (
	// NewPayloadMethod v1 request string for JSON-RPC.
	NewPayloadMethod = "engine_newPayloadV1"
	// ForkchoiceUpdatedMethod v1 request string for JSON-RPC.
	ForkchoiceUpdatedMethod = "engine_forkchoiceUpdatedV1"
	// GetPayloadMethod v1 request string for JSON-RPC.
	GetPayloadMethod = "engine_getPayloadV1"
	// ExchangeTransitionConfigurationMethod v1 request string for JSON-RPC.
	ExchangeTransitionConfigurationMethod = "engine_exchangeTransitionConfigurationV1"
	// ExecutionBlockByHashMethod request string for JSON-RPC.
	ExecutionBlockByHashMethod = "eth_getBlockByHash"
	// ExecutionBlockByNumberMethod request string for JSON-RPC.
	ExecutionBlockByNumberMethod = "eth_getBlockByNumber"
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
	NewPayload(ctx context.Context, payload *pb.ExecutionPayload) ([]byte, error)
	ForkchoiceUpdated(
		ctx context.Context, state *pb.ForkchoiceState, attrs *pb.PayloadAttributes,
	) (*pb.PayloadIDBytes, []byte, error)
	GetPayload(ctx context.Context, payloadId [8]byte) (*pb.ExecutionPayload, error)
	ExchangeTransitionConfiguration(
		ctx context.Context, cfg *pb.TransitionConfiguration,
	) error
	LatestExecutionBlock(ctx context.Context) (*pb.ExecutionBlock, error)
	ExecutionBlockByHash(ctx context.Context, hash common.Hash) (*pb.ExecutionBlock, error)
}

// NewPayload calls the engine_newPayloadV1 method via JSON-RPC.
func (s *Service) NewPayload(ctx context.Context, payload *pb.ExecutionPayload) ([]byte, error) {
	ctx, span := trace.StartSpan(ctx, "powchain.engine-api-client.NewPayload")
	defer span.End()
	start := time.Now()
	defer func() {
		newPayloadLatency.Observe(float64(time.Since(start).Milliseconds()))
	}()

	result := &pb.PayloadStatus{}
	err := s.rpcClient.CallContext(ctx, result, NewPayloadMethod, payload)
	if err != nil {
		return nil, handleRPCError(err)
	}

	switch result.Status {
	case pb.PayloadStatus_INVALID_BLOCK_HASH:
		return nil, fmt.Errorf("could not validate block hash: %v", result.ValidationError)
	case pb.PayloadStatus_INVALID_TERMINAL_BLOCK:
		return nil, fmt.Errorf("could not satisfy terminal block condition: %v", result.ValidationError)
	case pb.PayloadStatus_ACCEPTED, pb.PayloadStatus_SYNCING:
		return nil, ErrAcceptedSyncingPayloadStatus
	case pb.PayloadStatus_INVALID:
		return result.LatestValidHash, ErrInvalidPayloadStatus
	case pb.PayloadStatus_VALID:
		return result.LatestValidHash, nil
	default:
		return nil, ErrUnknownPayloadStatus
	}
}

// ForkchoiceUpdated calls the engine_forkchoiceUpdatedV1 method via JSON-RPC.
func (s *Service) ForkchoiceUpdated(
	ctx context.Context, state *pb.ForkchoiceState, attrs *pb.PayloadAttributes,
) (*pb.PayloadIDBytes, []byte, error) {
	ctx, span := trace.StartSpan(ctx, "powchain.engine-api-client.ForkchoiceUpdated")
	defer span.End()
	start := time.Now()
	defer func() {
		forkchoiceUpdatedLatency.Observe(float64(time.Since(start).Milliseconds()))
	}()

	result := &ForkchoiceUpdatedResponse{}
	err := s.rpcClient.CallContext(ctx, result, ForkchoiceUpdatedMethod, state, attrs)
	if err != nil {
		return nil, nil, handleRPCError(err)
	}

	if result.Status == nil {
		return nil, nil, ErrNilResponse
	}
	resp := result.Status
	switch resp.Status {
	case pb.PayloadStatus_INVALID_TERMINAL_BLOCK:
		return nil, nil, fmt.Errorf("could not satisfy terminal block condition: %v", resp.ValidationError)
	case pb.PayloadStatus_SYNCING:
		return nil, nil, ErrAcceptedSyncingPayloadStatus
	case pb.PayloadStatus_INVALID:
		return nil, resp.LatestValidHash, ErrInvalidPayloadStatus
	case pb.PayloadStatus_VALID:
		return result.PayloadId, resp.LatestValidHash, nil
	default:
		return nil, nil, ErrUnknownPayloadStatus
	}
}

// GetPayload calls the engine_getPayloadV1 method via JSON-RPC.
func (s *Service) GetPayload(ctx context.Context, payloadId [8]byte) (*pb.ExecutionPayload, error) {
	ctx, span := trace.StartSpan(ctx, "powchain.engine-api-client.GetPayload")
	defer span.End()
	start := time.Now()
	defer func() {
		getPayloadLatency.Observe(float64(time.Since(start).Milliseconds()))
	}()

	result := &pb.ExecutionPayload{}
	err := s.rpcClient.CallContext(ctx, result, GetPayloadMethod, pb.PayloadIDBytes(payloadId))
	return result, handleRPCError(err)
}

// ExchangeTransitionConfiguration calls the engine_exchangeTransitionConfigurationV1 method via JSON-RPC.
func (s *Service) ExchangeTransitionConfiguration(
	ctx context.Context, cfg *pb.TransitionConfiguration,
) error {
	ctx, span := trace.StartSpan(ctx, "powchain.engine-api-client.ExchangeTransitionConfiguration")
	defer span.End()

	// We set terminal block number to 0 as the parameter is not set on the consensus layer.
	zeroBigNum := big.NewInt(0)
	cfg.TerminalBlockNumber = zeroBigNum.Bytes()
	result := &pb.TransitionConfiguration{}
	if err := s.rpcClient.CallContext(ctx, result, ExchangeTransitionConfigurationMethod, cfg); err != nil {
		return handleRPCError(err)
	}
	// We surface an error to the user if local configuration settings mismatch
	// according to the response from the execution node.
	cfgTerminalHash := params.BeaconConfig().TerminalBlockHash[:]
	if !bytes.Equal(cfgTerminalHash, result.TerminalBlockHash) {
		return errors.Wrapf(
			ErrConfigMismatch,
			"got %#x from execution node, wanted %#x",
			result.TerminalBlockHash,
			cfgTerminalHash,
		)
	}
	ttdCfg := params.BeaconConfig().TerminalTotalDifficulty
	ttdResult, err := hexutil.DecodeBig(result.TerminalTotalDifficulty)
	if err != nil {
		return errors.Wrap(err, "could not decode received terminal total difficulty")
	}
	if ttdResult.String() != ttdCfg {
		return errors.Wrapf(
			ErrConfigMismatch,
			"got %s from execution node, wanted %s",
			ttdResult.String(),
			ttdCfg,
		)
	}
	return nil
}

// LatestExecutionBlock fetches the latest execution engine block by calling
// eth_blockByNumber via JSON-RPC.
func (s *Service) LatestExecutionBlock(ctx context.Context) (*pb.ExecutionBlock, error) {
	ctx, span := trace.StartSpan(ctx, "powchain.engine-api-client.LatestExecutionBlock")
	defer span.End()

	result := &pb.ExecutionBlock{}
	err := s.rpcClient.CallContext(
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
func (s *Service) ExecutionBlockByHash(ctx context.Context, hash common.Hash) (*pb.ExecutionBlock, error) {
	ctx, span := trace.StartSpan(ctx, "powchain.engine-api-client.ExecutionBlockByHash")
	defer span.End()

	result := &pb.ExecutionBlock{}
	err := s.rpcClient.CallContext(ctx, result, ExecutionBlockByHashMethod, hash, false /* no full transaction objects */)
	return result, handleRPCError(err)
}

// Handles errors received from the RPC server according to the specification.
func handleRPCError(err error) error {
	if err == nil {
		return nil
	}
	if isTimeout(err) {
		return errors.Wrapf(ErrHTTPTimeout, "%s", err)
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

// ErrHTTPTimeout returns true if the error is a http.Client timeout error.
var ErrHTTPTimeout = errors.New("timeout from http.Client")

type httpTimeoutError interface {
	Error() string
	Timeout() bool
}

func isTimeout(e error) bool {
	t, ok := e.(httpTimeoutError)
	return ok && t.Timeout()
}
