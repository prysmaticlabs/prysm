package db

import (
	"reflect"
	"testing"

	"github.com/prysmaticlabs/go-bitfield"
	slashpb "github.com/prysmaticlabs/prysm/proto/slashing"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func TestSetProposedForEpoch_SetsBit(t *testing.T) {
	c := params.BeaconConfig()
	c.WeakSubjectivityPeriod = 128
	params.OverrideBeaconConfig(c)

	proposals := &slashpb.ValidatorProposalHistory{
		ProposalHistory:    bitfield.NewBitlist(c.WeakSubjectivityPeriod),
		LatestEpochWritten: 0,
	}
	epoch := uint64(4)
	SetProposedForEpoch(proposals, epoch)
	proposed := HasProposedForEpoch(proposals, epoch)
	if !proposed {
		t.Fatal("fuck")
	}
	// Make sure no other bits are changed.
	for i := uint64(0); i < c.WeakSubjectivityPeriod; i++ {
		if i == epoch {
			continue
		}
		proposed = HasProposedForEpoch(proposals, i)
		if proposed {
			t.Fatal("fuck")
		}
	}
}

func TestSetProposedForEpoch_PrunesOverWSPeriod(t *testing.T) {
	c := params.BeaconConfig()
	c.WeakSubjectivityPeriod = 128
	params.OverrideBeaconConfig(c)

	proposals := &slashpb.ValidatorProposalHistory{
		ProposalHistory:    bitfield.NewBitlist(c.WeakSubjectivityPeriod),
		LatestEpochWritten: 0,
	}
	prunedEpoch := uint64(3)
	SetProposedForEpoch(proposals, prunedEpoch)

	if proposals.LatestEpochWritten != prunedEpoch {
		t.Fatal(proposals.LatestEpochWritten)
	}

	epoch := uint64(132)
	SetProposedForEpoch(proposals, epoch)
	proposed := HasProposedForEpoch(proposals, epoch)
	if !proposed {
		t.Fatal("fuck")
	}
	if proposals.LatestEpochWritten != epoch {
		t.Fatal(proposals.LatestEpochWritten)
	}
	// Make sure no other bits are changed.
	for i := uint64(epoch - c.WeakSubjectivityPeriod); i < epoch; i++ {
		if i == epoch {
			continue
		}
		proposed = HasProposedForEpoch(proposals, i)
		if proposed {
			t.Fatal(i)
		}
	}
}

func TestSetProposedForEpoch_54KEpochsKeepsHistory(t *testing.T) {
	wsPeriod := params.BeaconConfig().WeakSubjectivityPeriod
	proposals := &slashpb.ValidatorProposalHistory{
		ProposalHistory:    bitfield.NewBitlist(wsPeriod),
		LatestEpochWritten: 0,
	}
	randomIndexes := []uint64{23, 423, 8900, 11347, 25033, 52225, 53999}
	for i := 0; i < len(randomIndexes); i++ {
		SetProposedForEpoch(proposals, randomIndexes[i])
	}
	if proposals.LatestEpochWritten != 53999 {
		t.Fatal(proposals.LatestEpochWritten)
	}

	// Make sure no other bits are changed.
	for i := uint64(0); i < wsPeriod; i++ {
		setIndex := false
		for r := 0; r < len(randomIndexes); r++ {
			if i == randomIndexes[r] {
				setIndex = true
				break
			}
		}

		if setIndex != HasProposedForEpoch(proposals, i) {
			t.Fatal(i)
		}
	}

	// Set a past epoch as proposed, and make sure the recent data isn't changed.
	SetProposedForEpoch(proposals, randomIndexes[1]+5)
	if proposals.LatestEpochWritten != randomIndexes[len(randomIndexes)-1] {
		t.Fatal("fuck")
	}
	// Proposal just marked should be true.
	if !HasProposedForEpoch(proposals, randomIndexes[1]+5) {
		t.Fatal(proposals.LatestEpochWritten)
	}
	// Previously marked proposal should stay true.
	if !HasProposedForEpoch(proposals, randomIndexes[1]) {
		t.Fatal(proposals.LatestEpochWritten)
	}
}

func TestSetProposedForEpoch_PreventsProposingFutureEpochs(t *testing.T) {
	wsPeriod := params.BeaconConfig().WeakSubjectivityPeriod
	proposals := &slashpb.ValidatorProposalHistory{
		ProposalHistory:    bitfield.NewBitlist(wsPeriod),
		LatestEpochWritten: 0,
	}
	SetProposedForEpoch(proposals, 200)
	if HasProposedForEpoch(proposals, wsPeriod+200) {
		t.Fatal("fuck")
	}
}

func TestProposalHistory_NilDB(t *testing.T) {
	db := SetupDB(t)
	defer TeardownDB(t, db)

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
		history *slashpb.ValidatorProposalHistory
	}{
		{
			pubkey: []byte{0},
			epoch:  uint64(0),
			history: &slashpb.ValidatorProposalHistory{
				ProposalHistory: bitfield.Bitlist{0x01, 0x01},
			},
		},
		{
			pubkey: []byte{1},
			epoch:  uint64(0),
			history: &slashpb.ValidatorProposalHistory{
				ProposalHistory: bitfield.Bitlist{0x01, 0x01},
			},
		},
		{
			pubkey: []byte{0},
			epoch:  uint64(1),
			history: &slashpb.ValidatorProposalHistory{
				ProposalHistory: bitfield.Bitlist{0x01, 0x02},
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
		history *slashpb.ValidatorProposalHistory
	}{
		{
			pubkey: []byte{0},
			epoch:  uint64(0),
			history: &slashpb.ValidatorProposalHistory{
				ProposalHistory: big.NewInt(1),
			},
		},
		{
			pubkey: []byte{1},
			epoch:  uint64(0),
			history: &slashpb.ValidatorProposalHistory{
				ProposalHistory: big.NewInt(1),
			},
		},
		{
			pubkey: []byte{0},
			epoch:  uint64(1),
			history: &slashpb.ValidatorProposalHistory{
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
