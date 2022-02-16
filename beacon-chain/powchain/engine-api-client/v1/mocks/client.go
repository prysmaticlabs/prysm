package mocks

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
	v1 "github.com/prysmaticlabs/prysm/beacon-chain/powchain/engine-api-client/v1"
	pb "github.com/prysmaticlabs/prysm/proto/engine/v1"
)

// EngineClient --
type EngineClient struct {
	PayloadStatus             *pb.PayloadStatus
	ForkChoiceUpdatedResponse *v1.ForkchoiceUpdatedResponse
	ExecutionPayload          *pb.ExecutionPayload
	Err                       error
	ExecutionBlock            *pb.ExecutionBlock
}

// NewPayload --
func (e *EngineClient) NewPayload(_ context.Context, _ *pb.ExecutionPayload) (*pb.PayloadStatus, error) {
	return e.PayloadStatus, nil
}

// ForkchoiceUpdated --
func (e *EngineClient) ForkchoiceUpdated(
	_ context.Context, _ *pb.ForkchoiceState, _ *pb.PayloadAttributes,
) (*v1.ForkchoiceUpdatedResponse, error) {
	return e.ForkChoiceUpdatedResponse, nil
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
	return e.ExecutionBlock, nil
}

// ExecutionBlockByHash --
func (e *EngineClient) ExecutionBlockByHash(_ context.Context, _ common.Hash) (*pb.ExecutionBlock, error) {
	return e.ExecutionBlock, nil
}
