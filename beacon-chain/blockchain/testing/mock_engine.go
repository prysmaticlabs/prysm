package testing

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"
	enginev1 "github.com/prysmaticlabs/prysm/proto/engine/v1"
)

// MockEngineService is a mock implementation of the Caller interface.
type MockEngineService struct {
	newPayloadError error
	forkchoiceError error
	Blks            map[[32]byte]*enginev1.ExecutionBlock
}

// NewPayload --
func (m *MockEngineService) NewPayload(context.Context, *enginev1.ExecutionPayload) ([]byte, error) {
	return nil, m.newPayloadError
}

// ForkchoiceUpdated --
func (m *MockEngineService) ForkchoiceUpdated(context.Context, *enginev1.ForkchoiceState, *enginev1.PayloadAttributes) (*enginev1.PayloadIDBytes, []byte, error) {
	return nil, nil, m.forkchoiceError
}

// GetPayloadV1 --
func (*MockEngineService) GetPayloadV1(
	_ context.Context, _ enginev1.PayloadIDBytes,
) *enginev1.ExecutionPayload {
	return nil
}

// GetPayload --
func (*MockEngineService) GetPayload(context.Context, [8]byte) (*enginev1.ExecutionPayload, error) {
	return nil, nil
}

// ExchangeTransitionConfiguration --
func (*MockEngineService) ExchangeTransitionConfiguration(context.Context, *enginev1.TransitionConfiguration) error {
	return nil
}

// LatestExecutionBlock --
func (m *MockEngineService) LatestExecutionBlock(context.Context) (*enginev1.ExecutionBlock, error) {
	return m.Blks[[32]byte{}], nil
}

// ExecutionBlockByHash --
func (m *MockEngineService) ExecutionBlockByHash(_ context.Context, hash common.Hash) (*enginev1.ExecutionBlock, error) {
	blk, ok := m.Blks[common.BytesToHash(hash.Bytes())]
	if !ok {
		return nil, errors.New("block not found")
	}
	return blk, nil
}
