package casper

import (
	"math"
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/params"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

func NewValidators() []*pb.ValidatorRecord {
	var validators []*pb.ValidatorRecord

	for i := 0; i < 10; i++ {
		validator := &pb.ValidatorRecord{Balance: 1e18, StartDynasty: 1, EndDynasty: 10}
		validators = append(validators, validator)
	}
	return validators
}

func TestComputeValidatorRewardsAndPenalties(t *testing.T) {
	validators := NewValidators()
	defaultBalance := uint64(1e18)

	rewQuotient := RewardQuotient(1, validators)
	participatedDeposit := 4 * defaultBalance
	totalDeposit := 10 * defaultBalance
	depositFactor := (2*participatedDeposit - totalDeposit) / totalDeposit
	penaltyQuotient := QuadraticPenaltyQuotient()
	timeSinceFinality := uint64(5)

	data := &pb.CrystallizedState{
		Validators:        validators,
		CurrentDynasty:    1,
		TotalDeposits:     totalDeposit,
		LastJustifiedSlot: 4,
		LastFinalizedSlot: 3,
	}

	rewardedValidators, err := CalculateRewards(
		5,
		[]uint32{2, 3, 6, 9},
		data.Validators,
		data.CurrentDynasty,
		participatedDeposit,
		timeSinceFinality)

	if err != nil {
		t.Fatalf("could not compute validator rewards and penalties: %v", err)
	}

	expectedBalance := defaultBalance - defaultBalance/uint64(rewQuotient)

	if rewardedValidators[0].Balance != expectedBalance {
		t.Fatalf("validator balance not updated correctly: %d, %d", rewardedValidators[0].Balance, expectedBalance)
	}

	expectedBalance = uint64(defaultBalance + (defaultBalance/rewQuotient)*depositFactor)

	if rewardedValidators[6].Balance != expectedBalance {
		t.Fatalf("validator balance not updated correctly: %d, %d", rewardedValidators[6].Balance, expectedBalance)
	}

	if rewardedValidators[9].Balance != expectedBalance {
		t.Fatalf("validator balance not updated correctly: %d, %d", rewardedValidators[9].Balance, expectedBalance)
	}

	validators = NewValidators()
	timeSinceFinality = 168

	rewardedValidators, err = CalculateRewards(
		5,
		[]uint32{1, 2, 7, 8},
		validators,
		data.CurrentDynasty,
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
		{Balance: 1e18,
			StartDynasty: 0,
			EndDynasty:   2},
	}
	rewQuotient := RewardQuotient(0, validators)

	if rewQuotient != params.BaseRewardQuotient {
		t.Errorf("incorrect reward quotient: %d", rewQuotient)
	}
}

func TestSlotMaxInterestRate(t *testing.T) {
	validators := []*pb.ValidatorRecord{
		{Balance: 1e18,
			StartDynasty: 0,
			EndDynasty:   2},
	}

	interestRate := SlotMaxInterestRate(0, validators)

	if interestRate != 1/float64(params.BaseRewardQuotient) {
		t.Errorf("incorrect interest rate generated %f", interestRate)
	}

}

func TestQuadraticPenaltyQuotient(t *testing.T) {
	penaltyQuotient := QuadraticPenaltyQuotient()

	if penaltyQuotient != uint64(math.Pow(math.Pow(2, 17), 0.5)) {
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
