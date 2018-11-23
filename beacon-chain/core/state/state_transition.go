package state

import (
	"github.com/prysmaticlabs/prysm/beacon-chain/core/incentives"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/types"
	v "github.com/prysmaticlabs/prysm/beacon-chain/core/validators"
	"github.com/prysmaticlabs/prysm/beacon-chain/utils"
	"github.com/prysmaticlabs/prysm/shared/params"
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
	newState.SetValidators(v.CopyValidators(newState.Validators()))

	if st.LastStateRecalculationSlot() < params.BeaconConfig().CycleLength {
		lastStateRecalculationSlotCycleBack = 0
	} else {
		lastStateRecalculationSlotCycleBack = st.LastStateRecalculationSlot() - params.BeaconConfig().CycleLength
	}

	// walk through all the slots from LastStateRecalculationSlot - cycleLength to
	// LastStateRecalculationSlot - 1.
	for i := uint64(0); i < params.BeaconConfig().CycleLength; i++ {
		var blockVoteBalance uint64

		slot := lastStateRecalculationSlotCycleBack + i
		blockHash := recentBlockHashes[i]

		blockVoteBalance, validators := incentives.TallyVoteBalances(
			blockHash,
			blockVoteCache,
			newState.Validators(),
			v.ActiveValidatorIndices(newState.Validators()),
			v.TotalActiveValidatorDeposit(newState.Validators()),
			timeSinceFinality,
		)

		newState.SetValidators(validators)

		justifiedSlot, finalizedSlot, justifiedStreak = FinalizeAndJustifySlots(
			slot,
			justifiedSlot,
			finalizedSlot,
			justifiedStreak,
			blockVoteBalance,
			v.TotalActiveValidatorDeposit(newState.Validators()),
		)
	}

	crossLinks, err := newState.processCrosslinks(
		st.PendingAttestations(),
		newState.Validators(),
		block.SlotNumber(),
	)
	if err != nil {
		return nil, err
	}

	newState.SetCrossLinks(crossLinks)
	newState.SetLastJustifiedSlot(justifiedSlot)
	newState.SetLastFinalizedSlot(finalizedSlot)
	newState.SetJustifiedStreak(justifiedSlot)
	newState.SetLastStateRecalculationSlot(newState.LastStateRecalculationSlot() + params.BeaconConfig().CycleLength)

	// Exit the validators when their balance fall below min online deposit size.
	newState.SetValidators(v.CheckValidatorMinDeposit(newState.Validators(), block.SlotNumber()))

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
