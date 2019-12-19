package db

import (
	// "math/big"
	// "reflect"
	"testing"

	"github.com/prysmaticlabs/go-bitfield"
	ethpb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func TestSetProposedForEpoch_SetsBit(t *testing.T) {
	c := params.BeaconConfig()
	c.WeakSubjectivityPeriod = 128
	params.OverrideBeaconConfig(c)

	proposals := &ethpb.ValidatorProposalHistory{
		ProposalHistory:  bitfield.NewBitlist(c.WeakSubjectivityPeriod),
		LastEpochWritten: 0,
	}
	epoch := uint64(4)
	proposals.SetProposedForEpoch(epoch)
	proposed := proposals.HasProposedForEpoch(epoch)
	if !proposed {
		t.Fatal("fuck")
	}
	// Make sure no other bits are changed.
	for i := uint64(0); i < c.WeakSubjectivityPeriod; i++ {
		if i == epoch {
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

	proposals := &ethpb.ValidatorProposalHistory{
		ProposalHistory:  bitfield.NewBitlist(c.WeakSubjectivityPeriod),
		LastEpochWritten: 0,
	}
	prunedEpoch := uint64(3)
	proposals.SetProposedForEpoch(prunedEpoch)

	if proposals.LastEpochWritten != prunedEpoch {
		t.Fatal(proposals.LastEpochWritten)
	}

	epoch := uint64(132)
	proposals.SetProposedForEpoch(epoch)
	proposed := proposals.HasProposedForEpoch(epoch)
	if !proposed {
		t.Fatal("fuck")
	}
	if proposals.LastEpochWritten != epoch {
		t.Fatal(proposals.LastEpochWritten)
	}
	// Make sure no other bits are changed.
	for i := uint64(epoch - c.WeakSubjectivityPeriod); i < epoch; i++ {
		if i == epoch {
			continue
		}
		proposed = proposals.HasProposedForEpoch(i)
		if proposed {
			t.Fatal(i)
		}
	}
}

func TestSetProposedForEpoch_54KEpochsKeepsHistory(t *testing.T) {
	wsPeriod := params.BeaconConfig().WeakSubjectivityPeriod
	proposals := &ethpb.ValidatorProposalHistory{
		ProposalHistory:  bitfield.NewBitlist(wsPeriod),
		LastEpochWritten: 0,
	}
	randomIndexes := []uint64{23, 423, 8900, 11347, 25033, 52225, 53999}
	for i := 0; i < len(randomIndexes); i++ {
		proposals.SetProposedForEpoch(randomIndexes[i])
	}
	if proposals.LastEpochWritten != 53999 {
		t.Fatal(proposals.LastEpochWritten)
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

		if setIndex != proposals.HasProposedForEpoch(i) {
			t.Fatal(i)
		}
	}

	// Set a past epoch as proposed, and make sure the recent data isn't changed.
	proposals.SetProposedForEpoch(randomIndexes[1] + 5)
	if proposals.LastEpochWritten != randomIndexes[len(randomIndexes)-1] {
		t.Fatal("fuck")
	}
	// Proposal just marked should be true.
	if !proposals.HasProposedForEpoch(randomIndexes[1] + 5) {
		t.Fatal(proposals.LastEpochWritten)
	}
	// Previously marked proposal should stay true.
	if !proposals.HasProposedForEpoch(randomIndexes[1]) {
		t.Fatal(proposals.LastEpochWritten)
	}
}

func TestSetProposedForEpoch_PreventsProposingFutureEpochs(t *testing.T) {
	wsPeriod := params.BeaconConfig().WeakSubjectivityPeriod
	proposals := &ethpb.ValidatorProposalHistory{
		ProposalHistory:  bitfield.NewBitlist(wsPeriod),
		LastEpochWritten: 0,
	}
	proposals.SetProposedForEpoch(200)
	if proposals.HasProposedForEpoch(wsPeriod + 200) {
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
		history *ethpb.ValidatorProposalHistory
	}{
		{
			pubkey: []byte{0},
			epoch:  uint64(0),
			history: &ethpb.ValidatorProposalHistory{
				ProposalHistory: bitfield.NewBitlist,
			},
		},
		{
			pubkey: []byte{1},
			epoch:  uint64(0),
			history: &ethpb.ValidatorProposalHistory{
				ProposalHistory: big.NewInt(1),
			},
		},
		{
			pubkey: []byte{0},
			epoch:  uint64(1),
			history: &ethpb.ValidatorProposalHistory{
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
		history *ethpb.ValidatorProposalHistory
	}{
		{
			pubkey: []byte{0},
			epoch:  uint64(0),
			history: &ethpb.ValidatorProposalHistory{
				ProposalHistory: big.NewInt(1),
			},
		},
		{
			pubkey: []byte{1},
			epoch:  uint64(0),
			history: &ethpb.ValidatorProposalHistory{
				ProposalHistory: big.NewInt(1),
			},
		},
		{
			pubkey: []byte{0},
			epoch:  uint64(1),
			history: &ethpb.ValidatorProposalHistory{
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
