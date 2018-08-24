package casper

import (
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/types"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

func TestComputeValidatorRewardsAndPenalties(t *testing.T) {
	var validators []*pb.ValidatorRecord
	for i := 0; i < 40; i++ {
		validator := &pb.ValidatorRecord{Balance: 32, StartDynasty: 1, EndDynasty: 10}
		validators = append(validators, validator)
	}

	block := NewBlock(t, &pb.BeaconBlock{
		SlotNumber: 5,
	})

	data := &pb.CrystallizedState{
		Validators:        validators,
		CurrentDynasty:    1,
		TotalDeposits:     100,
		LastJustifiedSlot: 4,
		LastFinalizedSlot: 3,
	}
	crystallized := types.NewCrystallizedState(data)

	// Binary representation of bitfield: 11001000 10010100 10010010 10110011 00110001
	active := types.NewActiveState(&pb.ActiveState{})
	if err := CalculateRewards(active, crystallized, block); err != nil {
		t.Fatalf("error should be nil as function should have simply returned if no pending attestations: %v", err)
	}

	testAttesterBitfield := []byte{200, 148, 146, 179, 49}
	active = types.NewActiveState(&pb.ActiveState{PendingAttestations: []*pb.AttestationRecord{{AttesterBitfield: testAttesterBitfield}}})
	if err := CalculateRewards(active, crystallized, block); err != nil {
		t.Fatalf("could not compute validator rewards and penalties: %v", err)
	}
	if crystallized.LastJustifiedSlot() != uint64(5) {
		t.Fatalf("unable to update last justified Slot: %d", crystallized.LastJustifiedSlot())
	}
	if crystallized.LastFinalizedSlot() != uint64(4) {
		t.Fatalf("unable to update last finalized Slot: %d", crystallized.LastFinalizedSlot())
	}
	if crystallized.Validators()[0].Balance != uint64(33) {
		t.Fatalf("validator balance not updated: %d", crystallized.Validators()[1].Balance)
	}
	if crystallized.Validators()[7].Balance != uint64(31) {
		t.Fatalf("validator balance not updated: %d", crystallized.Validators()[1].Balance)
	}
	if crystallized.Validators()[29].Balance != uint64(31) {
		t.Fatalf("validator balance not updated: %d", crystallized.Validators()[1].Balance)
	}
}
