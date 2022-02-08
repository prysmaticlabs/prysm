package blockchain

import (
	"github.com/holiman/uint256"
	"github.com/prysmaticlabs/prysm/config/params"
)

// validates terminal pow block by comparing own total difficulty with parent's total difficulty.
//
// def is_valid_terminal_pow_block(block: PowBlock, parent: PowBlock) -> bool:
//    is_total_difficulty_reached = block.total_difficulty >= TERMINAL_TOTAL_DIFFICULTY
//    is_parent_total_difficulty_valid = parent.total_difficulty < TERMINAL_TOTAL_DIFFICULTY
//    return is_total_difficulty_reached and is_parent_total_difficulty_valid
func validTerminalPowBlock(currentDifficulty *uint256.Int, parentDifficulty *uint256.Int) bool {
	ttd := params.BeaconConfig().TerminalTotalDifficulty
	totalDifficultyReached := currentDifficulty.Cmp(ttd) >= 0
	parentTotalDifficultyValid := ttd.Cmp(parentDifficulty) > 0
	return totalDifficultyReached && parentTotalDifficultyValid
}
