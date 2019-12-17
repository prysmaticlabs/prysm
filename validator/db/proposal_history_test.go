package db

import (
	"math/big"
	"reflect"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/params"
	// ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
)

func TestSetProposedForEpoch_SetsBit(t *testing.T) {
	c := params.BeaconConfig()
	c.WeakSubjectivityPeriod = 128
	params.OverrideBeaconConfig(c)

	proposals := &ValidatorProposalHistory{
		ProposalHistory: big.NewInt(0),
		EpochAtFirstBit: 0,
	}
	proposals.SetProposedForEpoch(4)
	proposed := proposals.HasProposedForEpoch(4)
	if !proposed {
		t.Fatal("fuck")
	}
	// Make sure no other bits are changed.
	for i := uint64(0); i < c.WeakSubjectivityPeriod; i++ {
		if i == 4 {
			continue
		}
		proposed = proposals.HasProposedForEpoch(i)
		if proposed {
			t.Fatal("fuck")
		}
	}
}

func TestSetProposedForEpoch_PrunesOverWSPeriod(t *testing.T) {
	c := params.BeaconConfig()
	c.WeakSubjectivityPeriod = 128
	params.OverrideBeaconConfig(c)

	epoch := uint64(132)
	proposals := &ValidatorProposalHistory{
		ProposalHistory: big.NewInt(0),
		EpochAtFirstBit: 0,
	}
	proposals.SetProposedForEpoch(epoch)

	if proposals.EpochAtFirstBit != 4 {
		t.Fatal(proposals.EpochAtFirstBit)
	}

	proposed := proposals.HasProposedForEpoch(epoch)
	if !proposed {
		t.Fatal("fuck")
	}
	// Make sure no other bits are changed.
	for i := uint64(proposals.EpochAtFirstBit); i < c.WeakSubjectivityPeriod+proposals.EpochAtFirstBit; i++ {
		if i == epoch {
			continue
		}
		proposed = proposals.HasProposedForEpoch(i)
		if proposed {
			t.Fatal("fuck")
		}
	}
}

func TestSetProposedForEpoch_54KEpochsPrunes(t *testing.T) {
	wsPeriod := params.BeaconConfig().WeakSubjectivityPeriod
	proposals := &ValidatorProposalHistory{
		ProposalHistory: big.NewInt(0),
		EpochAtFirstBit: 0,
	}
	randomIndexes := []uint64{23, 423, 8900, 11347, 25033, 52225, 53999}
	for i := 0; i < len(randomIndexes); i++ {
		proposals.SetProposedForEpoch(randomIndexes[i])
	}
	if proposals.EpochAtFirstBit != 0 {
		t.Fatal(proposals.EpochAtFirstBit)
	}

	// Make sure no other bits are changed.
	for i := uint64(0); i < params.BeaconConfig().WeakSubjectivityPeriod; i++ {
		setIndex := false
		for r := 0; r < len(randomIndexes); r++ {
			if i == randomIndexes[r] {
				setIndex = true
			}
		}
		if !setIndex {
			proposed := proposals.HasProposedForEpoch(i)
			if proposed {
				t.Fatal("fuck")
			}
		} else {
			proposed := proposals.HasProposedForEpoch(i)
			if !proposed {
				t.Fatal("fuck")
			}
		}
	}

	// Set proposed after 54K epochs to prune.
	proposals.SetProposedForEpoch(wsPeriod + randomIndexes[1])
	if proposals.EpochAtFirstBit != randomIndexes[1] {
		t.Fatal("fuck")
	}
	// Should be able to access epoch at first bit.
	if !proposals.HasProposedForEpoch(randomIndexes[1]) {
		t.Fatal("fuck")
	}
}

func TestProposalHistory_NilDB(t *testing.T) {
	db := SetupDB(t)
	defer TeardownDB(t, db)

	epoch := uint64(1)
	balPubkey := []byte{1, 2, 3}

	proposalHistory, err := db.ProposalHistory(balPubkey)
	if err != nil {
		t.Fatal(err)
	}

	if proposalHistory.ProposalHistory != nil {
		t.Fatal("expected proposal history to be nil")
	}
}

func TestSaveProposalHistory_OK(t *testing.T) {
	db := SetupDB(t)
	defer TeardownDB(t, db)
	tests := []struct {
		pubkey  []byte
		epoch   uint64
		history *ValidatorProposalHistory
	}{
		{
			pubkey: []byte{0},
			epoch:  uint64(0),
			history: &ValidatorProposalHistory{
				ProposalHistory: big.NewInt(1),
			},
		},
		{
			pubkey: []byte{1},
			epoch:  uint64(0),
			history: &ValidatorProposalHistory{
				ProposalHistory: big.NewInt(1),
			},
		},
		{
			pubkey: []byte{0},
			epoch:  uint64(1),
			history: &ValidatorProposalHistory{
				ProposalHistory: big.NewInt(2),
			},
		},
	}

	for _, tt := range tests {
		if err := db.SaveProposalHistory(tt.pubkey, tt.history); err != nil {
			t.Fatalf("save block failed: %v", err)
		}
		history, err := db.ProposalHistory(tt.pubkey)
		if err != nil {
			t.Fatalf("failed to get block: %v", err)
		}

		if history == nil || !reflect.DeepEqual(history, tt.history) {
			t.Fatalf("Expected DB to keep object the same, received: %v", history)
		}
		if !history.HasProposedForEpoch(tt.epoch) {
			t.Fatalf("Expected epoch %d to be marked as proposed for", tt.epoch)
		}
		if history.HasProposedForEpoch(tt.epoch + 1) {
			t.Fatalf("Expected epoch %d to not be marked as proposed", tt.epoch+1)
		}
	}
}

func TestDeleteProposalHistory_OK(t *testing.T) {
	db := SetupDB(t)
	defer TeardownDB(t, db)
	tests := []struct {
		pubkey  []byte
		epoch   uint64
		history *ValidatorProposalHistory
	}{
		{
			pubkey: []byte{0},
			epoch:  uint64(0),
			history: &ValidatorProposalHistory{
				ProposalHistory: big.NewInt(1),
			},
		},
		{
			pubkey: []byte{1},
			epoch:  uint64(0),
			history: &ValidatorProposalHistory{
				ProposalHistory: big.NewInt(1),
			},
		},
		{
			pubkey: []byte{0},
			epoch:  uint64(1),
			history: &ValidatorProposalHistory{
				ProposalHistory: big.NewInt(2),
			},
		},
	}

	for _, tt := range tests {
		if err := db.SaveProposalHistory(tt.pubkey, tt.history); err != nil {
			t.Fatalf("save block failed: %v", err)
		}
	}

	for _, tt := range tests {
		// Making sure everything is saved.
		history, err := db.ProposalHistory(tt.pubkey)
		if err != nil {
			t.Fatalf("failed to get block: %v", err)
		}
		if history == nil || !reflect.DeepEqual(history, tt.history) {
			t.Fatalf("Expected DB to keep object the same, received: %v", history)
		}
		if err := db.DeleteProposalHistory(tt.pubkey); err != nil {
			t.Fatal(err)
		}
	}
}
