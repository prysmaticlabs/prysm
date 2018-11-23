package state

import (
	v "github.com/prysmaticlabs/prysm/beacon-chain/core/validators"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/types"
	"github.com/prysmaticlabs/prysm/beacon-chain/utils"
)

// NewStateTransition computes the new beacon state, given the previous beacon state
// and a beacon block. This method is called during a cycle transition.
// We also check for validator set change transition and compute for new 
// committees if necessary during this transition.
func NewStateTransition(st *types.BeaconState, block *types.Block, blockVoteCache *utils.BlockVoteCache) (*types.BeaconState, error) {
	var lastStateRecalculationSlotCycleBack uint64
	var err error

	newState := st.CopyState()
	justifiedStreak := st.JustifiedStreak()
	justifiedSlot := st.LastJustifiedSlot()
	finalizedSlot := st.LastFinalizedSlot()
	timeSinceFinality := block.SlotNumber() - newState.LastFinalizedSlot()
	recentBlockHashes := st.RecentBlockHashes()
	newState.data.Validators = v.CopyValidators(newState.Validators())

	if c.LastStateRecalculationSlot() < params.BeaconConfig().CycleLength {
		lastStateRecalculationSlotCycleBack = 0
	} else {
		lastStateRecalculationSlotCycleBack = c.LastStateRecalculationSlot() - params.BeaconConfig().CycleLength
	}

	// walk through all the slots from LastStateRecalculationSlot - cycleLength to 
	// LastStateRecalculationSlot - 1.
	for i := uint64(0); i < params.BeaconConfig().CycleLength; i++ {
		var blockVoteBalance uint64

		slot := lastStateRecalculationSlotCycleBack + i
		blockHash := recentBlockHashes[i]

		blockVoteBalance, newState.data.Validators = incentives.TallyVoteBalances(
			blockHash,
			blockVoteCache,
			newState.data.Validators,
			v.ActiveValidatorIndices(newState.data.Validators),
			v.TotalActiveValidatorDeposit(newState.data.Validators),
			timeSinceFinality,
		)

		justifiedSlot, finalizedSlot, justifiedStreak = state.FinalizeAndJustifySlots(
			slot, 
			justifiedSlot, 
			finalizedSlot,
			justifiedStreak, 
			blockVoteBalance, 
			c.TotalDeposits(),
		)
	}

	newState.data.Crosslinks, err = newState.processCrosslinks(
		st.PendingAttestations(), 
		newState.Validators(), 
		block.SlotNumber(),
	)
	if err != nil {
		return nil, err
	}

	newState.data.LastJustifiedSlot = justifiedSlot
	newState.data.LastFinalizedSlot = finalizedSlot
	newState.data.JustifiedStreak = justifiedStreak
	newState.data.LastStateRecalculationSlot = newState.LastStateRecalculationSlot() + params.BeaconConfig().CycleLength

	// Process the pending special records gathered from last cycle.
	newState.data.Validators, err = state.ProcessSpecialRecords(
		block.SlotNumber(), 
		newState.Validators(),
		aState.PendingSpecials(),
	)
	if err != nil {
		return nil, err
	}

	// Exit the validators when their balance fall below min online deposit size.
	newState.data.Validators = v.CheckValidatorMinDeposit(newState.Validators(), block.SlotNumber())

	newState.data.LastFinalizedSlot = finalizedSlot
	// Entering new validator set change transition.
	if newState.isValidatorSetChange(block.SlotNumber()) {
		newState.data.ValidatorSetChangeSlot = newState.LastStateRecalculationSlot()
		newState.data.ShardAndCommitteesForSlots, err = newState.newValidatorSetRecalculations(block.ParentHash())
		if err != nil {
			return nil, err
		}

		period := uint32(block.SlotNumber() / params.BeaconConfig().MinWithdrawalPeriod)
		totalPenalties := newState.PenalizedETH(period)
		newState.data.Validators = v.ChangeValidators(block.SlotNumber(), totalPenalties, newState.Validators())
	}

	return newState, nil
}