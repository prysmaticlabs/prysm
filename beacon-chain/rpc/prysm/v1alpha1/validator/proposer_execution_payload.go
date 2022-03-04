package validator

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/holiman/uint256"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	"github.com/sirupsen/logrus"
)

// This returns the valid terminal block hash with an existence bool value.
//
// Spec code:
// def get_terminal_pow_block(pow_chain: Dict[Hash32, PowBlock]) -> Optional[PowBlock]:
//    if TERMINAL_BLOCK_HASH != Hash32():
//        # Terminal block hash override takes precedence over terminal total difficulty
//        if TERMINAL_BLOCK_HASH in pow_chain:
//            return pow_chain[TERMINAL_BLOCK_HASH]
//        else:
//            return None
//
//    return get_pow_block_at_terminal_total_difficulty(pow_chain)
func (vs *Server) getTerminalBlockHashIfExists(ctx context.Context) ([]byte, bool, error) {
	terminalBlockHash := params.BeaconConfig().TerminalBlockHash
	// Terminal block hash override takes precedence over terminal total difficulty.
	if params.BeaconConfig().TerminalBlockHash != params.BeaconConfig().ZeroHash {
		exists, _, err := vs.Eth1BlockFetcher.BlockExists(ctx, terminalBlockHash)
		if err != nil {
			return nil, false, err
		}
		if !exists {
			return nil, false, nil
		}

		return terminalBlockHash.Bytes(), true, nil
	}

	return vs.getPowBlockHashAtTerminalTotalDifficulty(ctx)
}

// This returns the valid terminal block hash based on total difficulty.
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
func (vs *Server) getPowBlockHashAtTerminalTotalDifficulty(ctx context.Context) ([]byte, bool, error) {
	ttd := new(big.Int)
	ttd.SetString(params.BeaconConfig().TerminalTotalDifficulty, 10)
	terminalTotalDifficulty, overflows := uint256.FromBig(ttd)
	if overflows {
		return nil, false, errors.New("could not convert terminal total difficulty to uint256")
	}
	blk, err := vs.ExecutionEngineCaller.LatestExecutionBlock(ctx)
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
		parentBlk, err := vs.ExecutionEngineCaller.ExecutionBlockByHash(ctx, parentHash)
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
		}
		blk = parentBlk
	}
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
