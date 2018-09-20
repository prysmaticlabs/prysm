package casper

import (
	"github.com/prysmaticlabs/prysm/beacon-chain/params"
	"math"
	"testing"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

func TestComputeValidatorRewardsAndPenalties(t *testing.T) {
	var validators []*pb.ValidatorRecord
	for i := 0; i < 40; i++ {
		validator := &pb.ValidatorRecord{Balance: 32, StartDynasty: 1, EndDynasty: 10}
		validators = append(validators, validator)
	}

	data := &pb.CrystallizedState{
		Validators:        validators,
		CurrentDynasty:    1,
		TotalDeposits:     100,
		LastJustifiedSlot: 4,
		LastFinalizedSlot: 3,
	}

	rewardedValidators, err := CalculateRewards(
		0,
		[]uint32{},
		data.Validators,
		data.CurrentDynasty,
		data.TotalDeposits,
		1000)

	if err != nil {
		t.Fatalf("could not compute validator rewards and penalties: %v", err)
	}
	if rewardedValidators[0].Balance != uint64(33) {
		t.Fatalf("validator balance not updated: %d", rewardedValidators[0].Balance)
	}
	if rewardedValidators[7].Balance != uint64(31) {
		t.Fatalf("validator balance not updated: %d", rewardedValidators[7].Balance)
	}
	if rewardedValidators[29].Balance != uint64(31) {
		t.Fatalf("validator balance not updated: %d", rewardedValidators[29].Balance)
	}
}

func TestRewardQuotient(t *testing.T) {
	validators := []*pb.ValidatorRecord{
		&pb.ValidatorRecord{Balance: 1e18,
			StartDynasty: 0,
			EndDynasty:   2},
	}
	rewQuotient := RewardQuotient(0, validators)

	if rewQuotient != params.BaseRewardQuotient {
		t.Errorf("incorrect reward quotient: %f", rewQuotient)
	}
}

func TestSlotMaxInterestRate(t *testing.T) {
	validators := []*pb.ValidatorRecord{
		&pb.ValidatorRecord{Balance: 1e18,
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

	if penaltyQuotient != math.Pow(math.Pow(2, 17), 0.5) {
		t.Errorf("incorrect penalty quotient %f", penaltyQuotient)
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
