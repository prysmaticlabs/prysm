package powchain

import (
	"bytes"
	"context"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/holiman/uint256"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	pb "github.com/prysmaticlabs/prysm/proto/engine/v1"
	"github.com/sirupsen/logrus"
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
	// Defines the seconds to wait before timing out engine endpoints with block execution semantics (newPayload, forkchoiceUpdated).
	payloadAndForkchoiceUpdatedTimeout = 8 * time.Second
	// Defines the seconds before timing out engine endpoints with non-block execution semantics.
	defaultEngineTimeout = time.Second
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
	ExecutionBlockByHash(ctx context.Context, hash common.Hash) (*pb.ExecutionBlock, error)
	GetTerminalBlockHash(ctx context.Context) ([]byte, bool, error)
}

// NewPayload calls the engine_newPayloadV1 method via JSON-RPC.
func (s *Service) NewPayload(ctx context.Context, payload *pb.ExecutionPayload) ([]byte, error) {
	ctx, span := trace.StartSpan(ctx, "powchain.engine-api-client.NewPayload")
	defer span.End()
	start := time.Now()
	defer func() {
		newPayloadLatency.Observe(float64(time.Since(start).Milliseconds()))
	}()
	d := time.Now().Add(payloadAndForkchoiceUpdatedTimeout)
	ctx, cancel := context.WithDeadline(ctx, d)
	defer cancel()
	result := &pb.PayloadStatus{}
	err := s.rpcClient.CallContext(ctx, result, NewPayloadMethod, payload)
	if err != nil {
		return nil, handleRPCError(err)
	}

	switch result.Status {
	case pb.PayloadStatus_INVALID_BLOCK_HASH:
		return nil, fmt.Errorf("could not validate block hash: %v", result.ValidationError)
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

	d := time.Now().Add(payloadAndForkchoiceUpdatedTimeout)
	ctx, cancel := context.WithDeadline(ctx, d)
	defer cancel()
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

	d := time.Now().Add(defaultEngineTimeout)
	ctx, cancel := context.WithDeadline(ctx, d)
	defer cancel()
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
	d := time.Now().Add(defaultEngineTimeout)
	ctx, cancel := context.WithDeadline(ctx, d)
	defer cancel()
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

// GetTerminalBlockHash returns the valid terminal block hash based on total difficulty.
//
// Spec code:
// def get_pow_block_at_terminal_total_difficulty(pow_chain: Dict[Hash32, PowBlock]) -> Optional[PowBlock]:
//    # `pow_chain` abstractly represents all blocks in the PoW chain
//    for block in pow_chain:
//        parent = pow_chain[block.parent_hash]
//        block_reached_ttd = block.total_difficulty >= TERMINAL_TOTAL_DIFFICULTY
//        parent_reached_ttd = parent.total_difficulty >= TERMINAL_TOTAL_DIFFICULTY
//        if block_reached_ttd and not parent_reached_ttd:
//            return block
//
//    return None
func (s *Service) GetTerminalBlockHash(ctx context.Context) ([]byte, bool, error) {
	ttd := new(big.Int)
	ttd.SetString(params.BeaconConfig().TerminalTotalDifficulty, 10)
	terminalTotalDifficulty, overflows := uint256.FromBig(ttd)
	if overflows {
		return nil, false, errors.New("could not convert terminal total difficulty to uint256")
	}
	blk, err := s.LatestExecutionBlock(ctx)
	if err != nil {
		return nil, false, errors.Wrap(err, "could not get latest execution block")
	}
	if blk == nil {
		return nil, false, errors.New("latest execution block is nil")
	}

	for {
		if ctx.Err() != nil {
			return nil, false, ctx.Err()
		}
		currentTotalDifficulty, err := tDStringToUint256(blk.TotalDifficulty)
		if err != nil {
			return nil, false, errors.Wrap(err, "could not convert total difficulty to uint256")
		}
		blockReachedTTD := currentTotalDifficulty.Cmp(terminalTotalDifficulty) >= 0

		parentHash := bytesutil.ToBytes32(blk.ParentHash)
		if len(blk.ParentHash) == 0 || parentHash == params.BeaconConfig().ZeroHash {
			return nil, false, nil
		}
		parentBlk, err := s.ExecutionBlockByHash(ctx, parentHash)
		if err != nil {
			return nil, false, errors.Wrap(err, "could not get parent execution block")
		}
		if parentBlk == nil {
			return nil, false, errors.New("parent execution block is nil")
		}
		if blockReachedTTD {
			parentTotalDifficulty, err := tDStringToUint256(parentBlk.TotalDifficulty)
			if err != nil {
				return nil, false, errors.Wrap(err, "could not convert total difficulty to uint256")
			}
			parentReachedTTD := parentTotalDifficulty.Cmp(terminalTotalDifficulty) >= 0
			if !parentReachedTTD {
				log.WithFields(logrus.Fields{
					"number":   blk.Number,
					"hash":     fmt.Sprintf("%#x", bytesutil.Trunc(blk.Hash)),
					"td":       blk.TotalDifficulty,
					"parentTd": parentBlk.TotalDifficulty,
					"ttd":      terminalTotalDifficulty,
				}).Info("Retrieved terminal block hash")
				return blk.Hash, true, nil
			}
		} else {
			return nil, false, nil
		}
		blk = parentBlk
	}
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
		if strings.Contains(err.Error(), "401 Unauthorized") {
			log.Error("HTTP authentication to your execution client is not working. Please ensure " +
				"you are setting a correct value for the --jwt-secret flag in Prysm, or use an IPC connection if on " +
				"the same machine. Please see our documentation for more information on authenticating connections " +
				"here https://docs.prylabs.network/docs/execution-node/authentication")
			return fmt.Errorf("could not authenticate connection to execution client: %v", err)
		}
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

func tDStringToUint256(td string) (*uint256.Int, error) {
	b, err := hexutil.DecodeBig(td)
	if err != nil {
		return nil, err
	}
	i, overflows := uint256.FromBig(b)
	if overflows {
		return nil, errors.New("total difficulty overflowed")
	}
	return i, nil
}
