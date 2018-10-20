package casper

import (
	"math"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/mathutil"

	"github.com/prysmaticlabs/prysm/beacon-chain/params"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

func NewValidators() []*pb.ValidatorRecord {
	var validators []*pb.ValidatorRecord

	for i := 0; i < 10; i++ {
		validator := &pb.ValidatorRecord{Balance: 32 * 1e9, Status: uint64(params.Active)}
		validators = append(validators, validator)
	}
	return validators
}

func TestComputeValidatorRewardsAndPenalties(t *testing.T) {
	validators := NewValidators()
	defaultBalance := uint64(32 * 1e9)

	rewQuotient := RewardQuotient(validators)
	participatedDeposit := 4 * defaultBalance
	totalDeposit := 10 * defaultBalance
	penaltyQuotient := quadraticPenaltyQuotient()
	timeSinceFinality := uint64(5)

	data := &pb.CrystallizedState{
		Validators:             validators,
		ValidatorSetChangeSlot: 1,
		LastJustifiedSlot:      4,
		LastFinalizedSlot:      3,
	}

	rewardedValidators := CalculateRewards(
		5,
		[]uint32{2, 3, 6, 9},
		data.Validators,
		participatedDeposit,
		timeSinceFinality)

	expectedBalance := defaultBalance - defaultBalance/uint64(rewQuotient)

	if rewardedValidators[0].Balance != expectedBalance {
		t.Fatalf("validator balance not updated correctly: %d, %d", rewardedValidators[0].Balance, expectedBalance)
	}

	expectedBalance = uint64(int64(defaultBalance) + int64(defaultBalance/rewQuotient)*int64(2*uint64(participatedDeposit)-uint64(totalDeposit))/int64(totalDeposit))

	if rewardedValidators[6].Balance != expectedBalance {
		t.Fatalf("validator balance not updated correctly: %d, %d", rewardedValidators[6].Balance, expectedBalance)
	}

	if rewardedValidators[9].Balance != expectedBalance {
		t.Fatalf("validator balance not updated correctly: %d, %d", rewardedValidators[9].Balance, expectedBalance)
	}

	validators = NewValidators()
	timeSinceFinality = 200

	rewardedValidators = CalculateRewards(
		5,
		[]uint32{1, 2, 7, 8},
		validators,
		participatedDeposit,
		timeSinceFinality)

	if rewardedValidators[1].Balance != defaultBalance {
		t.Fatalf("validator balance not updated correctly: %d, %d", rewardedValidators[1].Balance, defaultBalance)
	}

	if rewardedValidators[7].Balance != defaultBalance {
		t.Fatalf("validator balance not updated correctly: %d, %d", rewardedValidators[7].Balance, defaultBalance)
	}

	expectedBalance = defaultBalance - (defaultBalance/rewQuotient + defaultBalance*timeSinceFinality/penaltyQuotient)

	if rewardedValidators[0].Balance != expectedBalance {
		t.Fatalf("validator balance not updated correctly: %d, %d", rewardedValidators[0].Balance, expectedBalance)
	}

	if rewardedValidators[9].Balance != expectedBalance {
		t.Fatalf("validator balance not updated correctly: %d, %d", rewardedValidators[9].Balance, expectedBalance)
	}

}

func TestRewardQuotient(t *testing.T) {
	validators := []*pb.ValidatorRecord{
		{Balance: 1e9, Status: uint64(params.Active)},
	}
	rewQuotient := RewardQuotient(validators)

	if rewQuotient != params.GetConfig().BaseRewardQuotient {
		t.Errorf("incorrect reward quotient: %d", rewQuotient)
	}
}

func TestSlotMaxInterestRate(t *testing.T) {
	validators := []*pb.ValidatorRecord{
		{Balance: 1e9, Status: uint64(params.Active)},
	}

	interestRate := SlotMaxInterestRate(validators)

	if interestRate != 1/float64(params.GetConfig().BaseRewardQuotient) {
		t.Errorf("incorrect interest rate generated %f", interestRate)
	}

}

func TestQuadraticPenaltyQuotient(t *testing.T) {
	penaltyQuotient := quadraticPenaltyQuotient()
	if penaltyQuotient != uint64(math.Pow(math.Pow(2, 13), 2)) {
		t.Errorf("incorrect penalty quotient %d", penaltyQuotient)
	}
}

func TestQuadraticPenalty(t *testing.T) {
	numOfSlots := uint64(4)
	penalty := QuadraticPenalty(numOfSlots)
	penaltyQuotient := uint64(math.Pow(math.Pow(2, 17), 0.5))

	expectedPenalty := (numOfSlots * numOfSlots / 2) / penaltyQuotient

	if expectedPenalty != penalty {
		t.Errorf("quadric penalty is not the expected amount for %d slots %d", numOfSlots, penalty)
	}
}

func TestRewardCrosslink(t *testing.T) {
	totalDeposit := uint64(6e18)
	participatedDeposit := uint64(3e18)
	rewardQuotient := params.GetConfig().BaseRewardQuotient * mathutil.IntegerSquareRoot(totalDeposit)
	validator := &pb.ValidatorRecord{
		Balance: 1e18,
	}

	RewardValidatorCrosslink(totalDeposit, participatedDeposit, rewardQuotient, validator)

	if validator.Balance != 1e18 {
		t.Errorf("validator balances have changed when they were not supposed to %d", validator.Balance)
	}
	participatedDeposit = uint64(4e18)
	RewardValidatorCrosslink(totalDeposit, participatedDeposit, rewardQuotient, validator)

}

func TestPenaltyCrosslink(t *testing.T) {
	totalDeposit := uint64(6e18)
	rewardQuotient := params.GetConfig().BaseRewardQuotient * mathutil.IntegerSquareRoot(totalDeposit)
	validator := &pb.ValidatorRecord{
		Balance: 1e18,
	}
	timeSinceConfirmation := uint64(10)
	quadraticQuotient := quadraticPenaltyQuotient()

	PenaliseValidatorCrosslink(timeSinceConfirmation, rewardQuotient, validator)
	expectedBalance := 1e18 - (1e18/rewardQuotient + 1e18*timeSinceConfirmation/quadraticQuotient)

	if validator.Balance != expectedBalance {
		t.Fatalf("balances not updated correctly %d, %d", validator.Balance, expectedBalance)
	}

}
