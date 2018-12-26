package incentives

import (
	"math"
	"testing"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/mathutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

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
