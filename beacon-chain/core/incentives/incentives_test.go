package incentives

import (
	"math"
	"testing"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/mathutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func newValidatorRegistry() []*pb.ValidatorRecord {
	var validators []*pb.ValidatorRecord
	for i := 0; i < 10; i++ {
		validator := &pb.ValidatorRecord{Balance: 32 * 1e9, Status: uint64(params.Active)}
		validators = append(validators, validator)
	}
	return validators
}

func TestComputeValidatorRewardsAndPenalties(t *testing.T) {
	validators := newValidatorRegistry()
	defaultBalance := uint64(32 * 1e9)

	participatedDeposit := 4 * defaultBalance
	totalDeposit := 10 * defaultBalance
	rewQuotient := RewardQuotient(totalDeposit)
	penaltyQuotient := QuadraticPenaltyQuotient()
	timeSinceFinality := uint64(5)

	data := &pb.BeaconState{
		ValidatorRegistry:               validators,
		ValidatorRegistryLastChangeSlot: 1,
		JustifiedSlot:                   4,
		FinalizedSlot:                   3,
	}

	activeValidatorIndices := make([]uint32, 0, len(validators))
	for i, v := range validators {
		if v.Status == uint64(params.Active) {
			activeValidatorIndices = append(activeValidatorIndices, uint32(i))
		}
	}

	rewardedValidatorRegistry := CalculateRewards(
		[]uint32{2, 3, 6, 9},
		activeValidatorIndices,
		data.ValidatorRegistry,
		totalDeposit,
		participatedDeposit,
		timeSinceFinality,
	)

	expectedBalance := defaultBalance - defaultBalance/uint64(rewQuotient)

	if rewardedValidatorRegistry[0].Balance != expectedBalance {
		t.Fatalf("validator balance not updated correctly: %d, %d", rewardedValidatorRegistry[0].Balance, expectedBalance)
	}

	expectedBalance = uint64(int64(defaultBalance) + int64(defaultBalance/rewQuotient)*(2*int64(participatedDeposit)-int64(totalDeposit))/int64(totalDeposit))

	if rewardedValidatorRegistry[6].Balance != expectedBalance {
		t.Fatalf("validator balance not updated correctly: %d, %d", rewardedValidatorRegistry[6].Balance, expectedBalance)
	}

	if rewardedValidatorRegistry[9].Balance != expectedBalance {
		t.Fatalf("validator balance not updated correctly: %d, %d", rewardedValidatorRegistry[9].Balance, expectedBalance)
	}

	validators = newValidatorRegistry()
	timeSinceFinality = 200

	activeValidatorIndices = make([]uint32, 0, len(validators))
	for i, v := range validators {
		if v.Status == uint64(params.Active) {
			activeValidatorIndices = append(activeValidatorIndices, uint32(i))
		}
	}

	rewardedValidatorRegistry = CalculateRewards(
		[]uint32{1, 2, 7, 8},
		activeValidatorIndices,
		validators,
		totalDeposit,
		participatedDeposit,
		timeSinceFinality)

	if rewardedValidatorRegistry[1].Balance != defaultBalance {
		t.Fatalf("validator balance not updated correctly: %d, %d", rewardedValidatorRegistry[1].Balance, defaultBalance)
	}

	if rewardedValidatorRegistry[7].Balance != defaultBalance {
		t.Fatalf("validator balance not updated correctly: %d, %d", rewardedValidatorRegistry[7].Balance, defaultBalance)
	}

	expectedBalance = defaultBalance - (defaultBalance/rewQuotient + defaultBalance*timeSinceFinality/penaltyQuotient)

	if rewardedValidatorRegistry[0].Balance != expectedBalance {
		t.Fatalf("validator balance not updated correctly: %d, %d", rewardedValidatorRegistry[0].Balance, expectedBalance)
	}

	if rewardedValidatorRegistry[9].Balance != expectedBalance {
		t.Fatalf("validator balance not updated correctly: %d, %d", rewardedValidatorRegistry[9].Balance, expectedBalance)
	}

}

func TestRewardQuotient(t *testing.T) {
	defaultBalance := uint64(2 * 1e9)
	totalDeposit := defaultBalance
	rewQuotient := RewardQuotient(totalDeposit)

	if rewQuotient != params.BeaconConfig().BaseRewardQuotient {
		t.Errorf("incorrect reward quotient: %d", rewQuotient)
	}
}

func TestQuadraticPenaltyQuotient(t *testing.T) {
	penaltyQuotient := QuadraticPenaltyQuotient()
	if penaltyQuotient != uint64(math.Pow(2, 32)) {
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
	rewardQuotient := params.BeaconConfig().BaseRewardQuotient * mathutil.IntegerSquareRoot(totalDeposit)
	validator := &pb.ValidatorRecord{
		Balance: 1e18,
	}

	validator = RewardValidatorCrosslink(totalDeposit, participatedDeposit, rewardQuotient, validator)

	if validator.Balance != 1e18 {
		t.Errorf("validator balances have changed when they were not supposed to %d", validator.Balance)
	}
}

func TestPenaltyCrosslink(t *testing.T) {
	totalDeposit := uint64(6e18)
	rewardQuotient := params.BeaconConfig().BaseRewardQuotient * mathutil.IntegerSquareRoot(totalDeposit)
	validator := &pb.ValidatorRecord{
		Balance: 1e18,
	}
	timeSinceConfirmation := uint64(10)
	quadraticQuotient := QuadraticPenaltyQuotient()

	validator = PenaliseValidatorCrosslink(timeSinceConfirmation, rewardQuotient, validator)
	expectedBalance := 1e18 - (1e18/rewardQuotient + 1e18*timeSinceConfirmation/quadraticQuotient)

	if validator.Balance != expectedBalance {
		t.Fatalf("balances not updated correctly %d, %d", validator.Balance, expectedBalance)
	}
}
