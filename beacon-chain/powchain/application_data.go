package powchain

import (
	"context"

	"github.com/ethereum/go-ethereum/eth/catalyst"
)

func (s *Service) AssembleExecutionPayload(
	ctx context.Context, params catalyst.AssembleBlockParams,
) (*catalyst.ExecutableData, error) {
	return s.applicationExecutor.AssembleBlock(ctx, params)
}

func (s *Service) InsertExecutionPayload(
	ctx context.Context, data catalyst.ExecutableData,
) (*catalyst.NewBlockResponse, error) {
	return s.applicationExecutor.NewBlock(ctx, data)
}
