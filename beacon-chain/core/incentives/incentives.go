// Package incentives defines Casper Proof of Stake rewards and penalties for validator
// records based on Vitalik Buterin's Friendly Finality Gadget protocol. Validator balances
// depend on time to finality as well as deposit-weighted functions. This package provides
// pure functions that can then be incorporated into a beacon chain state transition.
package incentives

import (
	v "github.com/prysmaticlabs/prysm/beacon-chain/core/validators"
	"github.com/prysmaticlabs/prysm/beacon-chain/utils"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bitutil"
	"github.com/prysmaticlabs/prysm/shared/mathutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// TallyVoteBalances calculates all the votes behind a block and
// then rewards validators for their participation in voting for that block.
func TallyVoteBalances(
	blockHash [32]byte,
	blockVoteCache utils.BlockVoteCache,
	validators []*pb.ValidatorRecord,
	activeValidatorIndices []uint32,
	totalActiveValidatorDeposit uint64,
	timeSinceFinality uint64,
) (uint64, []*pb.ValidatorRecord) {
	blockVote, ok := blockVoteCache[blockHash]
	if !ok {
		return 0, validators
	}

	blockVoteBalance := blockVote.VoteTotalDeposit
	voterIndices := blockVote.VoterIndices
	newValidatorRegistry := CalculateRewards(
		voterIndices,
		activeValidatorIndices,
		validators,
		totalActiveValidatorDeposit,
		blockVoteBalance,
		timeSinceFinality,
	)

	return blockVoteBalance, newValidatorRegistry
}

// CalculateRewards adjusts validators balances by applying rewards or penalties
// based on FFG incentive structure.
// FFG Rewards scheme rewards validator who have voted on blocks, and penalises those validators
// who are offline. The penalties are more severe the longer they are offline.
func CalculateRewards(
	voterIndices []uint32,
	activeValidatorIndices []uint32,
	validators []*pb.ValidatorRecord,
	totalActiveValidatorDeposit uint64,
	totalParticipatedDeposit uint64,
	timeSinceFinality uint64,
) []*pb.ValidatorRecord {

	newValidatorSet := v.CopyValidatorRegistry(validators)

	// Calculate the reward and penalty quotients for the validator set.
	rewardQuotient := RewardQuotient(totalActiveValidatorDeposit)
	penaltyQuotient := QuadraticPenaltyQuotient()

	if timeSinceFinality <= 3*params.BeaconConfig().CycleLength {
		for _, validatorIndex := range activeValidatorIndices {
			var voted bool

			for _, voterIndex := range voterIndices {
				if voterIndex == validatorIndex {
					voted = true
					balance := validators[validatorIndex].GetBalance()
					newBalance := int64(balance) + int64(balance/rewardQuotient)*(2*int64(totalParticipatedDeposit)-int64(totalActiveValidatorDeposit))/int64(totalActiveValidatorDeposit)
					newValidatorSet[validatorIndex].Balance = uint64(newBalance)
					break
				}
			}

			if !voted {
				newBalance := newValidatorSet[validatorIndex].GetBalance()
				newBalance -= newBalance / rewardQuotient
				newValidatorSet[validatorIndex].Balance = newBalance
			}
		}

	} else {
		for _, validatorIndex := range activeValidatorIndices {
			var voted bool

			for _, voterIndex := range voterIndices {
				if voterIndex == validatorIndex {
					voted = true
					break
				}
			}

			if !voted {
				newBalance := newValidatorSet[validatorIndex].GetBalance()
				newBalance -= newBalance/rewardQuotient + newBalance*timeSinceFinality/penaltyQuotient
				newValidatorSet[validatorIndex].Balance = newBalance
			}
		}

	}

	return newValidatorSet
}

// ApplyCrosslinkRewardsAndPenalties applies the appropriate rewards and
// penalties according to the attestation for a shard.
func ApplyCrosslinkRewardsAndPenalties(
	crosslinkRecords []*pb.CrosslinkRecord,
	slot uint64,
	attesterIndices []uint32,
	attestation *pb.AggregatedAttestation,
	validators []*pb.ValidatorRecord,
	totalActiveValidatorDeposit uint64,
	totalBalance uint64,
	voteBalance uint64,
) ([]*pb.ValidatorRecord, error) {
	newValidatorSet := v.CopyValidatorRegistry(validators)

	rewardQuotient := RewardQuotient(totalActiveValidatorDeposit)
	for _, attesterIndex := range attesterIndices {
		timeSinceLastConfirmation := slot - crosslinkRecords[attestation.Shard].GetSlot()

		checkBit, err := bitutil.CheckBit(attestation.AttesterBitfield, int(attesterIndex))
		if err != nil {
			return nil, err
		}
		if checkBit {
			newValidatorSet[attesterIndex] = RewardValidatorCrosslink(totalBalance, voteBalance, rewardQuotient, newValidatorSet[attesterIndex])
		} else {
			newValidatorSet[attesterIndex] = PenaliseValidatorCrosslink(timeSinceLastConfirmation, rewardQuotient, newValidatorSet[attesterIndex])
		}
	}
	return newValidatorSet, nil
}

// RewardQuotient returns the reward quotient for validators which will be used to
// reward validators for voting on blocks, or penalise them for being offline.
func RewardQuotient(totalActiveValidatorDeposit uint64) uint64 {
	totalDepositETH := totalActiveValidatorDeposit / params.BeaconConfig().Gwei
	return params.BeaconConfig().BaseRewardQuotient * mathutil.IntegerSquareRoot(totalDepositETH)
}

// QuadraticPenaltyQuotient is the quotient that will be used to apply penalties to offline
// validators.
func QuadraticPenaltyQuotient() uint64 {
	dropTimeFactor := params.BeaconConfig().SqrtExpDropTime
	return dropTimeFactor * dropTimeFactor
}

// QuadraticPenalty returns the penalty that will be applied to an offline validator
// based on the number of slots that they are offline.
func QuadraticPenalty(numberOfSlots uint64) uint64 {
	slotFactor := (numberOfSlots * numberOfSlots) / 2
	penaltyQuotient := QuadraticPenaltyQuotient()
	return slotFactor / penaltyQuotient
}

// RewardValidatorCrosslink applies rewards to validators part of a shard committee for voting on a shard.
// TODO(#538): Change this to big.Int as tests using 64 bit integers fail due to integer overflow.
func RewardValidatorCrosslink(
	totalDeposit uint64,
	participatedDeposits uint64,
	rewardQuotient uint64,
	validator *pb.ValidatorRecord,
) *pb.ValidatorRecord {
	currentBalance := int64(validator.Balance)
	currentBalance += int64(currentBalance) / int64(rewardQuotient) * (2*int64(participatedDeposits) - int64(totalDeposit)) / int64(totalDeposit)
	return &pb.ValidatorRecord{
		Pubkey:                 validator.Pubkey,
		RandaoCommitmentHash32: validator.RandaoCommitmentHash32,
		Balance:                uint64(currentBalance),
		Status:                 validator.Status,
		LatestStatusChangeSlot: validator.LatestStatusChangeSlot,
	}
}

// PenaliseValidatorCrosslink applies penalties to validators part of a shard committee for not voting on a shard.
func PenaliseValidatorCrosslink(
	timeSinceLastConfirmation uint64,
	rewardQuotient uint64,
	validator *pb.ValidatorRecord,
) *pb.ValidatorRecord {
	newBalance := validator.Balance
	quadraticQuotient := QuadraticPenaltyQuotient()
	newBalance -= newBalance/rewardQuotient + newBalance*timeSinceLastConfirmation/quadraticQuotient
	return &pb.ValidatorRecord{
		Pubkey:                 validator.Pubkey,
		RandaoCommitmentHash32: validator.RandaoCommitmentHash32,
		Balance:                uint64(newBalance),
		Status:                 validator.Status,
		LatestStatusChangeSlot: validator.LatestStatusChangeSlot,
	}
}
