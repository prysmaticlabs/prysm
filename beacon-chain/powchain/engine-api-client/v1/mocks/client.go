package mocks

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"
	pb "github.com/prysmaticlabs/prysm/proto/engine/v1"
)

// EngineClient --
type EngineClient struct {
	NewPayloadResp        []byte
	PayloadIDBytes        *pb.PayloadIDBytes
	ForkChoiceUpdatedResp []byte
	ExecutionPayload      *pb.ExecutionPayload
	ExecutionBlock        *pb.ExecutionBlock
	Err                   error
	ErrLatestExecBlock    error
	ErrExecBlockByHash    error
	ErrExecBlockByNum     error
	ErrForkchoiceUpdated  error
	BlockByHashMap        map[[32]byte]*pb.ExecutionBlock
	BlockByNumMap         map[uint64]*pb.ExecutionBlock
}

// NewPayload --
func (e *EngineClient) NewPayload(_ context.Context, _ *pb.ExecutionPayload) ([]byte, error) {
	return e.NewPayloadResp, nil
}

// ForkchoiceUpdated --
func (e *EngineClient) ForkchoiceUpdated(
	_ context.Context, _ *pb.ForkchoiceState, _ *pb.PayloadAttributes,
) (*pb.PayloadIDBytes, []byte, error) {
	return e.PayloadIDBytes, e.ForkChoiceUpdatedResp, e.ErrForkchoiceUpdated
}

// GetPayload --
func (e *EngineClient) GetPayload(_ context.Context, _ [8]byte) (*pb.ExecutionPayload, error) {
	return e.ExecutionPayload, nil
}

// ExchangeTransitionConfiguration --
func (e *EngineClient) ExchangeTransitionConfiguration(_ context.Context, _ *pb.TransitionConfiguration) error {
	return e.Err
}

// LatestExecutionBlock --
func (e *EngineClient) LatestExecutionBlock(_ context.Context) (*pb.ExecutionBlock, error) {
	return e.ExecutionBlock, e.ErrLatestExecBlock
}

// ExecutionBlockByHash --
func (e *EngineClient) ExecutionBlockByHash(_ context.Context, h common.Hash) (*pb.ExecutionBlock, error) {
	b, ok := e.BlockByHashMap[h]
	if !ok {
		return nil, errors.New("block not found")
	}
	return b, e.ErrExecBlockByHash
}

// ExecutionBlockByNumber --
func (e *EngineClient) ExecutionBlockByNumber(_ context.Context, num *big.Int) (*pb.ExecutionBlock, error) {
	b, ok := e.BlockByNumMap[num.Uint64()]
	if !ok {
		return nil, errors.New("block not found")
	}
	return b, e.ErrExecBlockByNum
}
