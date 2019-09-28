package powchain

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"
	"go.opencensus.io/trace"
)

// BlockExists returns true if the block exists, it's height and any possible error encountered.
func (s *Service) BlockExists(ctx context.Context, hash common.Hash) (bool, *big.Int, error) {
	ctx, span := trace.StartSpan(ctx, "beacon-chain.web3service.BlockExists")
	defer span.End()

	if exists, blkInfo, err := s.blockCache.BlockInfoByHash(hash); exists || err != nil {
		if err != nil {
			return false, nil, err
		}
		span.AddAttributes(trace.BoolAttribute("blockCacheHit", true))
		return true, blkInfo.Number, nil
	}
	span.AddAttributes(trace.BoolAttribute("blockCacheHit", false))
	block, err := s.blockFetcher.BlockByHash(ctx, hash)
	if err != nil {
		return false, big.NewInt(0), errors.Wrap(err, "could not query block with given hash")
	}

	if err := s.blockCache.AddBlock(block); err != nil {
		return false, big.NewInt(0), err
	}

	return true, block.Number(), nil
}

// BlockHashByHeight returns the block hash of the block at the given height.
func (s *Service) BlockHashByHeight(ctx context.Context, height *big.Int) (common.Hash, error) {
	ctx, span := trace.StartSpan(ctx, "beacon-chain.web3service.BlockHashByHeight")
	defer span.End()

	if exists, blkInfo, err := s.blockCache.BlockInfoByHeight(height); exists || err != nil {
		if err != nil {
			return [32]byte{}, err
		}
		span.AddAttributes(trace.BoolAttribute("blockCacheHit", true))
		return blkInfo.Hash, nil
	}
	span.AddAttributes(trace.BoolAttribute("blockCacheHit", false))
	block, err := s.blockFetcher.BlockByNumber(ctx, height)
	if err != nil {
		return [32]byte{}, errors.Wrap(err, "could not query block with given height")
	}
	if err := s.blockCache.AddBlock(block); err != nil {
		return [32]byte{}, err
	}
	return block.Hash(), nil
}

// BlockTimeByHeight fetches an eth1.0 block timestamp by its height.
func (s *Service) BlockTimeByHeight(ctx context.Context, height *big.Int) (uint64, error) {
	ctx, span := trace.StartSpan(ctx, "beacon-chain.web3service.BlockTimeByHeight")
	defer span.End()
	block, err := s.blockFetcher.BlockByNumber(ctx, height)
	if err != nil {
		return 0, errors.Wrap(err, "could not query block with given height")
	}
	return block.Time(), nil
}

// BlockNumberByTimestamp returns the most recent block number up to a given timestamp.
// This is a naive implementation that will use O(ETH1_FOLLOW_DISTANCE) calls to cache
// or ETH1. This is called for multiple times but only changes every
// SlotsPerEth1VotingPeriod (1024 slots) so the whole method should be cached.
func (s *Service) BlockNumberByTimestamp(ctx context.Context, time uint64) (*big.Int, error) {
	ctx, span := trace.StartSpan(ctx, "beacon-chain.web3service.BlockByTimestamp")
	defer span.End()

	head, err := s.blockFetcher.BlockByNumber(ctx, nil)
	if err != nil {
		return nil, err
	}

	for bn := head.Number(); ; bn = big.NewInt(0).Sub(bn, big.NewInt(1)) {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		exists, info, err := s.blockCache.BlockInfoByHeight(bn)
		if err != nil {
			return nil, err
		}

		if !exists {
			blk, err := s.blockFetcher.BlockByNumber(ctx, bn)
			if err != nil {
				return nil, err
			}
			if err := s.blockCache.AddBlock(blk); err != nil {
				return nil, err
			}
			info = blockToBlockInfo(blk)
		}

		if info.Time <= time {
			return info.Number, nil
		}
	}
}
