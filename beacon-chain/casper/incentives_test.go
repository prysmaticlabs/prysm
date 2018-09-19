package casper

import (
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

	// Binary representation of bitfield: 11001000 10010100 10010010 10110011 00110001
	testAttesterBitfield := []*pb.AggregatedAttestation{{AttesterBitfield: []byte{200, 148, 146, 179, 49}}}
	rewardedValidators, err := CalculateRewards(
		testAttesterBitfield,
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
