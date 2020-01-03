package db

import (
	"reflect"
	"testing"

	"github.com/prysmaticlabs/go-bitfield"
	slashpb "github.com/prysmaticlabs/prysm/proto/slashing"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func TestProposalHistory_InitializesNewPubKeys(t *testing.T) {
	pubkeys := [][]byte{[]byte{30}, []byte{25}, []byte{20}}
	db := SetupDB(t, pubkeys)
	defer TeardownDB(t, db)

	for _, pub := range pubkeys {
		proposalHistory, err := db.ProposalHistory(pub)
		if err != nil {
			t.Fatal(err)
		}

		clean := &slashpb.ProposalHistory{
			EpochBits: bitfield.NewBitlist(params.BeaconConfig().WeakSubjectivityPeriod),
		}
		if !reflect.DeepEqual(proposalHistory, clean) {
			t.Fatalf("Expected proposal history epoch bits to be empty, received %v", proposalHistory)
		}
	}
}

func TestProposalHistory_NilDB(t *testing.T) {
	db := SetupDB(t, [][]byte{})
	defer TeardownDB(t, db)

	balPubkey := []byte{1, 2, 3}

	proposalHistory, err := db.ProposalHistory(balPubkey)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(proposalHistory, &slashpb.ProposalHistory{}) {
		t.Fatalf("Expected proposal history to be empty, received: %v", proposalHistory)
	}
}

func TestSaveProposalHistory_OK(t *testing.T) {
	db := SetupDB(t, [][]byte{})
	defer TeardownDB(t, db)
	tests := []struct {
		pubkey  []byte
		epoch   uint64
		history *slashpb.ProposalHistory
	}{
		{
			pubkey: []byte{0},
			epoch:  uint64(1),
			history: &slashpb.ProposalHistory{
				EpochBits:          bitfield.Bitlist{0x02, 0x02},
				LatestEpochWritten: 1,
			},
		},
		{
			pubkey: []byte{1},
			epoch:  uint64(1),
			history: &slashpb.ProposalHistory{
				EpochBits:          bitfield.Bitlist{0x02, 0x02},
				LatestEpochWritten: 1,
			},
		},
		{
			pubkey: []byte{3},
			epoch:  uint64(2),
			history: &slashpb.ProposalHistory{
				EpochBits:          bitfield.Bitlist{0x04, 0x04},
				LatestEpochWritten: 2,
			},
		},
	}

	for _, tt := range tests {
		if err := db.SaveProposalHistory(tt.pubkey, tt.history); err != nil {
			t.Fatalf("Saving proposal history failed: %v", err)
		}
		history, err := db.ProposalHistory(tt.pubkey)
		if err != nil {
			t.Fatalf("Failed to get proposal history: %v", err)
		}

		if history == nil || !reflect.DeepEqual(history, tt.history) {
			t.Fatalf("Expected DB to keep object the same, received: %v", history)
		}
		if !history.EpochBits.BitAt(tt.epoch) {
			t.Fatalf("Expected epoch %d to be marked as proposed", history.EpochBits.Count())
		}
		if history.EpochBits.BitAt(tt.epoch+1) {
			t.Fatalf("Expected epoch %d to not be marked as proposed", tt.epoch+1)
		}
		if history.EpochBits.BitAt(tt.epoch-1) {
			t.Fatalf("Expected epoch %d to not be marked as proposed", tt.epoch-1)
		}
	}
}

func TestSaveProposalHistory_Overwrites(t *testing.T) {
	db := SetupDB(t, [][]byte{})
	defer TeardownDB(t, db)
	tests := []struct {
		pubkey  []byte
		epoch   uint64
		history *slashpb.ProposalHistory
	}{
		{
			pubkey: []byte{0},
			epoch:  uint64(1),
			history: &slashpb.ProposalHistory{
				EpochBits:          bitfield.Bitlist{0x02, 0x02},
				LatestEpochWritten: 1,
			},
		},
		{
			pubkey: []byte{0},
			epoch:  uint64(2),
			history: &slashpb.ProposalHistory{
				EpochBits:          bitfield.Bitlist{0x04, 0x04},
				LatestEpochWritten: 2,
			},
		},
		{
			pubkey: []byte{0},
			epoch:  uint64(3),
			history: &slashpb.ProposalHistory{
				EpochBits:          bitfield.Bitlist{0x08, 0x08},
				LatestEpochWritten: 3,
			},
		},
	}

	for _, tt := range tests {
		if err := db.SaveProposalHistory(tt.pubkey, tt.history); err != nil {
			t.Fatalf("Saving proposal history failed: %v", err)
		}
		history, err := db.ProposalHistory(tt.pubkey)
		if err != nil {
			t.Fatalf("Failed to get proposal history: %v", err)
		}

		if history == nil || !reflect.DeepEqual(history, tt.history) {
			t.Fatalf("Expected DB to keep object the same, received: %v", history)
		}
		if !history.EpochBits.BitAt(tt.epoch) {
			t.Fatalf("Expected epoch %d to be marked as proposed", history.EpochBits.Count())
		}
		if history.EpochBits.BitAt(tt.epoch+1) {
			t.Fatalf("Expected epoch %d to not be marked as proposed", tt.epoch+1)
		}
		if history.EpochBits.BitAt(tt.epoch-1) {
			t.Fatalf("Expected epoch %d to not be marked as proposed", tt.epoch-1)
		}
	}
}

func TestDeleteProposalHistory_OK(t *testing.T) {
	db := SetupDB(t, [][]byte{})
	defer TeardownDB(t, db)
	tests := []struct {
		pubkey  []byte
		epoch   uint64
		history *slashpb.ProposalHistory
	}{
		{
			pubkey: []byte{0},
			epoch:  uint64(0),
			history: &slashpb.ProposalHistory{
				EpochBits: bitfield.Bitlist{0x01, 0x01},
			},
		},
		{
			pubkey: []byte{1},
			epoch:  uint64(0),
			history: &slashpb.ProposalHistory{
				EpochBits: bitfield.Bitlist{0x01, 0x01},
			},
		},
		{
			pubkey: []byte{2},
			epoch:  uint64(1),
			history: &slashpb.ProposalHistory{
				EpochBits:          bitfield.Bitlist{0x01, 0x02},
				LatestEpochWritten: 1,
			},
		},
	}

	for _, tt := range tests {
		if err := db.SaveProposalHistory(tt.pubkey, tt.history); err != nil {
			t.Fatalf("Save proposal history failed: %v", err)
		}
		// Making sure everything is saved.
		history, err := db.ProposalHistory(tt.pubkey)
		if err != nil {
			t.Fatalf("Failed to get proposal history: %v", err)
		}
		if history == nil || !reflect.DeepEqual(history, tt.history) {
			t.Fatalf("Expected DB to keep object the same, received: %v, expected %v", history, tt.history)
		}
		if err := db.DeleteProposalHistory(tt.pubkey); err != nil {
			t.Fatal(err)
		}

		// Check after deleting from DB.
		history, err = db.ProposalHistory(tt.pubkey)
		if err != nil {
			t.Fatalf("Failed to get proposal history: %v", err)
		}
		if reflect.DeepEqual(history, tt.history) {
			t.Fatalf("Expected proposal history to be nil, received %v", history)
		}
	}
}
