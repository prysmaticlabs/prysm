package blockchain

import (
	"fmt"

	"github.com/prysmaticlabs/prysm/beacon-chain/types"
)

type beaconDB interface {
	GetBlock(hash [32]byte) (*types.Block, error)
}

// canonicalDecision is the result of the casper fork choice rule
// being applied by the beacon chain. It contains a block, crystallized, active
// state triple that are determined to be canonical by the protocol.
type canonicalDecision struct {
	canonicalBlock             *types.Block
	canonicalCrystallizedState *types.CrystallizedState
	canonicalActiveState       *types.ActiveState
	cycleDidTransition         bool
}

// ghostForkChoice applies a trimmed-down version of
// Latest-Message-Drive GHOST blockchain fork choice
// rule.
func ghostForkChoice(
	db beaconDB,
	currentActiveState *types.ActiveState,
	currentCrystallizedState *types.CrystallizedState,
	blockHashes [][32]byte,
	enableCrossLinksCheck bool,
	enableRewardCheck bool,
	enableAttestationValidityCheck bool,
) (*canonicalDecision, error) {
	// We keep track of the highest scoring received block and
	// its associated states.
	var highestScoringBlock *types.Block
	var highestScoringCrystallizedState *types.CrystallizedState
	var highestScoringActiveState *types.ActiveState
	var highestScore uint64
	var cycleTransitioned bool

	// We loop over every block in order to determine
	// the highest scoring one.
	for i := 0; i < len(blockHashes); i++ {
		block, err := db.GetBlock(blockHashes[i])
		if err != nil {
			return nil, fmt.Errorf("could not get block: %v", err)
		}

		h, err := block.Hash()
		if err != nil {
			return nil, fmt.Errorf("could not hash incoming block: %v", err)
		}

		parentBlock, err := db.GetBlock(block.ParentHash())
		if err != nil {
			return nil, fmt.Errorf("failed to get parent of block %x", h)
		}

		cState := currentCrystallizedState
		aState := currentActiveState

		for cState.IsCycleTransition(parentBlock.SlotNumber()) {
			cState, aState, err = cState.NewStateRecalculations(
				aState,
				block,
				enableCrossLinksCheck,
				enableRewardCheck,
			)
			if err != nil {
				return nil, fmt.Errorf("initialize new cycle transition failed: %v", err)
			}
			cycleTransitioned = true
		}

		aState, err = aState.CalculateNewActiveState(
			block,
			cState,
			parentBlock.SlotNumber(),
			enableAttestationValidityCheck,
		)
		if err != nil {
			return nil, fmt.Errorf("compute active state failed: %v", err)
		}

		// Initially, we set the highest scoring block to the first value in the
		// processed blocks list.
		if i == 0 {
			highestScoringBlock = block
			highestScoringCrystallizedState = cState
			highestScoringActiveState = aState
			continue
		}
		// Score the block and determine if its score is greater than the previously computed one.
		if block.Score(cState.LastFinalizedSlot(), cState.LastJustifiedSlot()) > highestScore {
			highestScoringBlock = block
			highestScoringCrystallizedState = cState
			highestScoringActiveState = aState
		}
	}
	return &canonicalDecision{
		canonicalBlock:             highestScoringBlock,
		canonicalCrystallizedState: highestScoringCrystallizedState,
		canonicalActiveState:       highestScoringActiveState,
		cycleDidTransition:         cycleTransitioned,
	}, nil
}
