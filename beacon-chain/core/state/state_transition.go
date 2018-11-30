package state

import (
	"github.com/prysmaticlabs/prysm/beacon-chain/core/incentives"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/types"
	v "github.com/prysmaticlabs/prysm/beacon-chain/core/validators"
	"github.com/prysmaticlabs/prysm/beacon-chain/utils"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// NewStateTransition computes the new beacon state, given the previous beacon state
// and a beacon block. This method is called during a cycle transition.
// We also check for validator set change transition and compute for new
// committees if necessary during this transition.
func NewStateTransition(
	st *types.BeaconState,
	block *types.Block,
	blockVoteCache utils.BlockVoteCache,
) (*types.BeaconState, error) {
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
			v.TotalActiveValidatorDeposit(st.Validators()),
		)
	}

	crossLinks, err := crossLinkCalculations(
		newState,
		st.PendingAttestations(),
		block.SlotNumber(),
	)
	if err != nil {
		return nil, err
	}

	newState.SetCrossLinks(crossLinks)

	newState.SetLastJustifiedSlot(justifiedSlot)
	newState.SetLastFinalizedSlot(finalizedSlot)
	newState.SetJustifiedStreak(justifiedStreak)
	newState.SetLastStateRecalculationSlot(newState.LastStateRecalculationSlot() + params.BeaconConfig().CycleLength)

	// Exit the validators when their balance fall below min online deposit size.
	newState.SetValidators(v.CheckValidatorMinDeposit(newState.Validators(), block.SlotNumber()))

	// Entering new validator set change transition.
	if newState.IsValidatorSetChange(block.SlotNumber()) {
		newState.SetValidatorSetChangeSlot(newState.LastStateRecalculationSlot())
		shardAndCommitteesForSlots, err := validatorSetRecalculations(
			newState.ShardAndCommitteesForSlots(),
			newState.Validators(),
			block.ParentHash(),
		)
		if err != nil {
			return nil, err
		}
		newState.SetShardAndCommitteesForSlots(shardAndCommitteesForSlots)

		period := block.SlotNumber() / params.BeaconConfig().MinWithdrawalPeriod
		totalPenalties := newState.PenalizedETH(period)
		newState.SetValidators(v.ChangeValidators(block.SlotNumber(), totalPenalties, newState.Validators()))
	}

	return newState, nil
}

// crossLinkCalculations checks if the proposed shard block has recevied
// 2/3 of the votes. If yes, we update crosslink record to point to
// the proposed shard block with latest beacon chain slot numbers.
func crossLinkCalculations(
	st *types.BeaconState,
	pendingAttestations []*pb.AggregatedAttestation,
	currentSlot uint64,
) ([]*pb.CrosslinkRecord, error) {
	slot := st.LastStateRecalculationSlot() + params.BeaconConfig().CycleLength
	crossLinkRecords := st.Crosslinks()
	for _, attestation := range pendingAttestations {
		shardCommittees, err := v.GetShardAndCommitteesForSlot(
			st.ShardAndCommitteesForSlots(),
			st.LastStateRecalculationSlot(),
			attestation.GetSlot(),
		)
		if err != nil {
			return nil, err
		}

		indices, err := v.AttesterIndices(shardCommittees, attestation)
		if err != nil {
			return nil, err
		}

		totalBalance, voteBalance, err := v.VotedBalanceInAttestation(st.Validators(), indices, attestation)
		if err != nil {
			return nil, err
		}

		newValidatorSet, err := incentives.ApplyCrosslinkRewardsAndPenalties(
			crossLinkRecords,
			currentSlot,
			indices,
			attestation,
			st.Validators(),
			v.TotalActiveValidatorDeposit(st.Validators()),
			totalBalance,
			voteBalance,
		)
		if err != nil {
			return nil, err
		}
		st.SetValidators(newValidatorSet)
		crossLinkRecords = UpdateCrosslinks(slot, voteBalance, totalBalance, attestation, crossLinkRecords)
	}
	return crossLinkRecords, nil
}

// validatorSetRecalculation recomputes the validator set.
func validatorSetRecalculations(
	shardAndCommittesForSlots []*pb.ShardAndCommitteeArray,
	validators []*pb.ValidatorRecord,
	seed [32]byte,
) ([]*pb.ShardAndCommitteeArray, error) {
	lastSlot := len(shardAndCommittesForSlots) - 1
	lastCommitteeFromLastSlot := len(shardAndCommittesForSlots[lastSlot].ArrayShardAndCommittee) - 1
	crosslinkLastShard := shardAndCommittesForSlots[lastSlot].ArrayShardAndCommittee[lastCommitteeFromLastSlot].Shard
	crosslinkNextShard := (crosslinkLastShard + 1) % params.BeaconConfig().ShardCount

	newShardCommitteeArray, err := v.ShuffleValidatorsToCommittees(
		seed,
		validators,
		crosslinkNextShard,
	)
	if err != nil {
		return nil, err
	}

	return append(shardAndCommittesForSlots[params.BeaconConfig().CycleLength:], newShardCommitteeArray...), nil
}
