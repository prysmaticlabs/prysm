package incentives

import (
	"math"
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/utils"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/mathutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func newValidators() []*pb.ValidatorRecord {
	var validators []*pb.ValidatorRecord
	for i := 0; i < 10; i++ {
		validator := &pb.ValidatorRecord{Balance: 32 * 1e9, Status: uint64(params.Active)}
		validators = append(validators, validator)
	}
	return validators
}

func TestTallyVoteBalances(t *testing.T) {
	var validators []*pb.ValidatorRecord
	var blockHash [32]byte

	blockVoteCache := utils.NewBlockVoteCache()
	initialBalance := uint64(1e9)
	for i := 0; i < 1000; i++ {
		validator := &pb.ValidatorRecord{
			WithdrawalShard: 0,
			Balance:         initialBalance}

		validators = append(validators, validator)
	}

	validators[20].Status = uint64(params.Active)
	validators[10].Status = uint64(params.Active)

	blockVote := &utils.BlockVote{
		VoterIndices:     []uint32{20, 10},
		VoteTotalDeposit: 1e9,
	}
	copy(blockHash[:], []byte{'t', 'e', 's', 't', 'i', 'n', 'g'})

	blockVoteCache[blockHash] = blockVote

	zeroBalance, _ := TallyVoteBalances([32]byte{}, blockVoteCache, validators, []uint32{10, 20}, 1e9, 2)

	if zeroBalance != 0 {
		t.Fatalf("votes have been calculated despite blockhash not existing in cache")
	}

	voteBalance, newValidators := TallyVoteBalances(blockHash, blockVoteCache, validators, []uint32{10, 20}, 1e9, 2)
	if voteBalance != 1e9 {
		t.Fatalf("vote balances is not the amount expected %d", voteBalance)
	}

	if newValidators[1].Balance != initialBalance {
		t.Fatalf("validator balance changed %d ", newValidators[1].Balance)
	}

	if newValidators[20].Balance == initialBalance {
		t.Fatalf("validator balance not changed %d ", newValidators[20].Balance)
	}

	if newValidators[10].Balance == initialBalance {
		t.Errorf("validator balance not changed %d ", newValidators[10].Balance)
	}
}

func TestApplyCrosslinkRewardsAndPenalties(t *testing.T) {
	var validators []*pb.ValidatorRecord
	initialBalance := uint64(1e9)
	totalBalance := uint64(5e9)
	voteBalance := uint64(4e9)
	indices := []uint32{20, 10}

	for i := 0; i < 1000; i++ {
		validator := &pb.ValidatorRecord{
			WithdrawalShard: 0,
			Balance:         initialBalance}

		validators = append(validators, validator)
	}

	validators[20].Status = uint64(params.Active)
	validators[10].Status = uint64(params.Active)

	crossLinks := []*pb.CrosslinkRecord{
		{
			ShardBlockHash: []byte{'A'},
			Slot:           10,
		},
		{
			ShardBlockHash: []byte{'B'},
			Slot:           10,
		},
		{
			ShardBlockHash: []byte{'C'},
			Slot:           10,
		},
		{
			ShardBlockHash: []byte{'D'},
			Slot:           10,
		},
	}

	attestation := &pb.AggregatedAttestation{
		Slot:             10,
		Shard:            1,
		AttesterBitfield: []byte{100, 128, 8},
	}

	totalActiveValidatorDeposit := uint64(1e9)
	ApplyCrosslinkRewardsAndPenalties(
		crossLinks,
		12,
		indices,
		attestation,
		validators,
		totalActiveValidatorDeposit,
		totalBalance,
		voteBalance,
	)

	if validators[20].Balance <= initialBalance {
		t.Fatalf("validator balance has not been updated %d", validators[20].Balance)
	}

	if validators[10].Balance >= initialBalance {
		t.Fatalf("validator balance has not been updated %d", validators[10].Balance)
	}

	if validators[1].Balance != initialBalance {
		t.Fatalf("validator balance updated when it was not supposed to %d", validators[1].Balance)
	}
}

func TestComputeValidatorRewardsAndPenalties(t *testing.T) {
	validators := newValidators()
	defaultBalance := uint64(32 * 1e9)

	participatedDeposit := 4 * defaultBalance
	totalDeposit := 10 * defaultBalance
	rewQuotient := RewardQuotient(totalDeposit)
	penaltyQuotient := QuadraticPenaltyQuotient()
	timeSinceFinality := uint64(5)

	data := &pb.CrystallizedState{
		Validators:             validators,
		ValidatorSetChangeSlot: 1,
		LastJustifiedSlot:      4,
		LastFinalizedSlot:      3,
	}

	activeValidatorIndices := make([]uint32, 0, len(validators))
	for i, v := range validators {
		if v.Status == uint64(params.Active) {
			activeValidatorIndices = append(activeValidatorIndices, uint32(i))
		}
	}

	rewardedValidators := CalculateRewards(
		[]uint32{2, 3, 6, 9},
		activeValidatorIndices,
		data.Validators,
		totalDeposit,
		participatedDeposit,
		timeSinceFinality,
	)

	expectedBalance := defaultBalance - defaultBalance/uint64(rewQuotient)

	if rewardedValidators[0].Balance != expectedBalance {
		t.Fatalf("validator balance not updated correctly: %d, %d", rewardedValidators[0].Balance, expectedBalance)
	}

	expectedBalance = calculateBalance(defaultBalance, rewQuotient, participatedDeposit, totalDeposit)

	if rewardedValidators[6].Balance != expectedBalance {
		t.Fatalf("validator balance not updated correctly: %d, %d", rewardedValidators[6].Balance, expectedBalance)
	}

	if rewardedValidators[9].Balance != expectedBalance {
		t.Fatalf("validator balance not updated correctly: %d, %d", rewardedValidators[9].Balance, expectedBalance)
	}

	validators = newValidators()
	timeSinceFinality = 200

	activeValidatorIndices = make([]uint32, 0, len(validators))
	for i, v := range validators {
		if v.Status == uint64(params.Active) {
			activeValidatorIndices = append(activeValidatorIndices, uint32(i))
		}
	}

	rewardedValidators = CalculateRewards(
		[]uint32{1, 2, 7, 8},
		activeValidatorIndices,
		validators,
		totalDeposit,
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
