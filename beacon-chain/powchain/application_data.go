package powchain

import (
	"context"

	"github.com/ethereum/go-ethereum/eth"
)

func (s *Service) ProduceApplicationData(
	ctx context.Context, params eth.ProduceBlockParams,
) (*eth.ApplicationPayload, error) {
	return s.applicationExecutor.ProduceBlock(ctx, params)
}

func (s *Service) InsertApplicationData(
	ctx context.Context, params eth.InsertBlockParams,
) (bool, error) {
	return s.applicationExecutor.InsertBlock(ctx, params)
}
