package powchain

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/traceutil"
	"go.opencensus.io/trace"
)

// BlockExists returns true if the block exists, it's height and any possible error encountered.
func (s *Service) BlockExists(ctx context.Context, hash common.Hash) (bool, *big.Int, error) {
	ctx, span := trace.StartSpan(ctx, "beacon-chain.web3service.BlockExists")
	defer span.End()

	if exists, hdrInfo, err := s.headerCache.HeaderInfoByHash(hash); exists || err != nil {
		if err != nil {
			return false, nil, err
		}
		span.AddAttributes(trace.BoolAttribute("blockCacheHit", true))
		return true, hdrInfo.Number, nil
	}
	span.AddAttributes(trace.BoolAttribute("blockCacheHit", false))
	header, err := s.eth1DataFetcher.HeaderByHash(ctx, hash)
	if err != nil {
		return false, big.NewInt(0), errors.Wrap(err, "could not query block with given hash")
	}

	if err := s.headerCache.AddHeader(header); err != nil {
		return false, big.NewInt(0), err
	}

	return true, new(big.Int).Set(header.Number), nil
}

// BlockHashByHeight returns the block hash of the block at the given height.
func (s *Service) BlockHashByHeight(ctx context.Context, height *big.Int) (common.Hash, error) {
	ctx, span := trace.StartSpan(ctx, "beacon-chain.web3service.BlockHashByHeight")
	defer span.End()

	if exists, hInfo, err := s.headerCache.HeaderInfoByHeight(height); exists || err != nil {
		if err != nil {
			return [32]byte{}, err
		}
		span.AddAttributes(trace.BoolAttribute("headerCacheHit", true))
		return hInfo.Hash, nil
	}
	span.AddAttributes(trace.BoolAttribute("headerCacheHit", false))

	if s.eth1DataFetcher == nil {
		err := errors.New("nil eth1DataFetcher")
		traceutil.AnnotateError(span, err)
		return [32]byte{}, err
	}

	header, err := s.eth1DataFetcher.HeaderByNumber(ctx, height)
	if err != nil {
		return [32]byte{}, errors.Wrap(err, fmt.Sprintf("could not query header with height %d", height.Uint64()))
	}
	if err := s.headerCache.AddHeader(header); err != nil {
		return [32]byte{}, err
	}
	return header.Hash(), nil
}

// BlockTimeByHeight fetches an eth1.0 block timestamp by its height.
func (s *Service) BlockTimeByHeight(ctx context.Context, height *big.Int) (uint64, error) {
	ctx, span := trace.StartSpan(ctx, "beacon-chain.web3service.BlockTimeByHeight")
	defer span.End()
	if s.eth1DataFetcher == nil {
		err := errors.New("nil eth1DataFetcher")
		traceutil.AnnotateError(span, err)
		return 0, err
	}

	header, err := s.eth1DataFetcher.HeaderByNumber(ctx, height)
	if err != nil {
		return 0, errors.Wrap(err, fmt.Sprintf("could not query block with height %d", height.Uint64()))
	}
	return header.Time, nil
}

// BlockNumberByTimestamp returns the most recent block number up to a given timestamp.
// This is a naive implementation that will use O(ETH1_FOLLOW_DISTANCE) calls to cache
// or ETH1. This is called for multiple times but only changes every
// SlotsPerEth1VotingPeriod (1024 slots) so the whole method should be cached.
func (s *Service) BlockNumberByTimestamp(ctx context.Context, time uint64) (*big.Int, error) {
	ctx, span := trace.StartSpan(ctx, "beacon-chain.web3service.BlockByTimestamp")
	defer span.End()

	if time > s.latestEth1Data.BlockTime {
		return nil, errors.New("provided time is later than the current eth1 head")
	}
	headNumber := big.NewInt(int64(s.latestEth1Data.BlockHeight))
	headTime := s.latestEth1Data.BlockTime
	numOfBlocks := uint64(0)
	estimatedBlk := headNumber.Uint64()
	for {
		if time > headTime+10*params.BeaconConfig().SecondsPerETH1Block {
			numOfBlocks = (time - headTime) / params.BeaconConfig().SecondsPerETH1Block
			estimatedBlk = headNumber.Uint64() + numOfBlocks
		} else if time+10*params.BeaconConfig().SecondsPerETH1Block < headTime {
			numOfBlocks = (headTime - time) / params.BeaconConfig().SecondsPerETH1Block
			estimatedBlk = headNumber.Uint64() - numOfBlocks
		} else {
			break
		}
		hinfo, err := s.retrieveHeaderInfo(ctx, estimatedBlk)
		if err != nil {
			return nil, err
		}
		headNumber = hinfo.Number
		headTime = hinfo.Time
	}

	log.Errorf("num of blocks: %d, estimated %d. Retrieved info time %d wanted time %d and expected head %d "+
		"and head time %d", numOfBlocks, estimatedBlk, headTime, time, headNumber.Uint64(), s.latestEth1Data.BlockTime)
	if headTime >= time {
		return s.findLessTargetEth1Block(ctx, big.NewInt(int64(estimatedBlk)), time)
	}
	return s.findMoreTargetEth1Block(ctx, big.NewInt(int64(estimatedBlk)), time)
}

// Performs a search to find a target eth1 block which is less than or equal to the
// target time. This method is used when head.time >= targetTime
func (s *Service) findLessTargetEth1Block(ctx context.Context, head *big.Int, targetTime uint64) (*big.Int, error) {
	defer func() {
		log.Error("finished finding blocks")
	}()
	for bn := head; ; bn = big.NewInt(0).Sub(bn, big.NewInt(1)) {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		info, err := s.retrieveHeaderInfo(ctx, bn.Uint64())
		if err != nil {
			return nil, err
		}

		if info.Time <= targetTime {
			return info.Number, nil
		}
	}
}

// Performs a search to find a target eth1 block which is the the block which
// is just less than to the target time. This method is used when head.time < targetTime
func (s *Service) findMoreTargetEth1Block(ctx context.Context, head *big.Int, targetTime uint64) (*big.Int, error) {
	defer func() {
		log.Error("finished finding blocks")
	}()
	for bn := head; ; bn = big.NewInt(0).Add(bn, big.NewInt(1)) {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		info, err := s.retrieveHeaderInfo(ctx, bn.Uint64())
		if err != nil {
			return nil, err
		}

		// Return the last block before we hit the threshold
		// time.
		if info.Time > targetTime {
			return big.NewInt(info.Number.Int64() - 1), nil
		}
		// If time is equal, this is our target block.
		if info.Time == targetTime {
			return info.Number, nil
		}
	}
}

func (s *Service) retrieveHeaderInfo(ctx context.Context, bNum uint64) (*headerInfo, error) {
	bn := big.NewInt(int64(bNum))
	log.Errorf("retrieving: %d", bNum)
	exists, info, err := s.headerCache.HeaderInfoByHeight(bn)
	if err != nil {
		return nil, err
	}

	if !exists {
		blk, err := s.eth1DataFetcher.HeaderByNumber(ctx, bn)
		if err != nil {
			return nil, err
		}
		if err := s.headerCache.AddHeader(blk); err != nil {
			return nil, err
		}
		info = headerToHeaderInfo(blk)
	}
	return info, nil
}
