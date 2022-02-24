package blockchain

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"
	enginev1 "github.com/prysmaticlabs/prysm/proto/engine/v1"
)

type mockEngineService struct {
	blks map[[32]byte]*enginev1.ExecutionBlock
}

func (*mockEngineService) NewPayload(context.Context, *enginev1.ExecutionPayload) ([]byte, error) {
	return nil, nil
}

func (*mockEngineService) ForkchoiceUpdated(context.Context, *enginev1.ForkchoiceState, *enginev1.PayloadAttributes) (*enginev1.PayloadIDBytes, []byte, error) {
	return nil, nil, nil
}

func (*mockEngineService) GetPayloadV1(
	_ context.Context, _ enginev1.PayloadIDBytes,
) *enginev1.ExecutionPayload {
	return nil
}

func (*mockEngineService) GetPayload(context.Context, [8]byte) (*enginev1.ExecutionPayload, error) {
	return nil, nil
}

func (*mockEngineService) ExchangeTransitionConfiguration(context.Context, *enginev1.TransitionConfiguration) error {
	return nil
}

func (*mockEngineService) LatestExecutionBlock(context.Context) (*enginev1.ExecutionBlock, error) {
	return nil, nil
}

func (m *mockEngineService) ExecutionBlockByHash(_ context.Context, hash common.Hash) (*enginev1.ExecutionBlock, error) {
	blk, ok := m.blks[common.BytesToHash(hash.Bytes())]
	if !ok {
		return nil, errors.New("block not found")
	}
	return blk, nil
}
