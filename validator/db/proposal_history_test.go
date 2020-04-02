package db

import (
	"bytes"
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func TestProposalHistoryForEpoch_InitializesNewPubKeys(t *testing.T) {
	pubkeys := [][48]byte{{30}, {25}, {20}}
	db := SetupDB(t, pubkeys)
	defer TeardownDB(t, db)

	for _, pub := range pubkeys {
		slotBits, err := db.ProposalHistoryForEpoch(context.Background(), pub[:], 0)
		if err != nil {
			t.Fatal(err)
		}

		cleanBits := bitfield.NewBitlist(params.BeaconConfig().SlotsPerEpoch)
		if !bytes.Equal(slotBits.Bytes(), cleanBits.Bytes()) {
			t.Fatalf("Expected proposal history slot bits to be empty, received %v", slotBits.Bytes())
		}
	}
}

func TestProposalHistoryForEpoch_NilDB(t *testing.T) {
	valPubkey := [48]byte{1, 2, 3}
	db := SetupDB(t, [][48]byte{})
	defer TeardownDB(t, db)

	_, err := db.ProposalHistoryForEpoch(context.Background(), valPubkey[:], 0)
	if err == nil {
		t.Fatal("unexpected non-error")
	}

	if !strings.Contains(err.Error(), "validator history empty for public key") {
		t.Fatalf("Unexpected error for nil DB, received: %v", err)
	}
}

func TestSaveProposalHistoryForEpoch_OK(t *testing.T) {
	pubkey := [48]byte{3}
	db := SetupDB(t, [][48]byte{pubkey})
	defer TeardownDB(t, db)

	epoch := uint64(2)
	slot := uint64(2)
	slotBits := bitfield.Bitlist{0x04, 0x04}

	if err := db.SaveProposalHistoryForEpoch(context.Background(), pubkey[:], epoch, slotBits); err != nil {
		t.Fatalf("Saving proposal history failed: %v", err)
	}
	savedBits, err := db.ProposalHistoryForEpoch(context.Background(), pubkey[:], epoch)
	if err != nil {
		t.Fatalf("Failed to get proposal history: %v", err)
	}

	if savedBits == nil || !bytes.Equal(slotBits.Bytes(), savedBits.Bytes()) {
		t.Fatalf("Expected DB to keep object the same, received: %v", savedBits)
	}
	if !savedBits.BitAt(slot) {
		t.Fatalf("Expected slot %d to be marked as proposed", slot)
	}
	if savedBits.BitAt(slot + 1) {
		t.Fatalf("Expected slot %d to not be marked as proposed", slot+1)
	}
	if savedBits.BitAt(slot - 1) {
		t.Fatalf("Expected slot %d to not be marked as proposed", slot-1)
	}
}

func TestSaveProposalHistoryForEpoch_Overwrites(t *testing.T) {
	pubkey := [48]byte{0}
	tests := []struct {
		slot     uint64
		slotBits bitfield.Bitlist
	}{
		{
			slot:     uint64(1),
			slotBits: bitfield.Bitlist{0x02, 0x02},
		},
		{
			slot:     uint64(2),
			slotBits: bitfield.Bitlist{0x04, 0x04},
		},
		{
			slot:     uint64(3),
			slotBits: bitfield.Bitlist{0x08, 0x08},
		},
	}

	for _, tt := range tests {
		db := SetupDB(t, [][48]byte{pubkey})
		defer TeardownDB(t, db)
		if err := db.SaveProposalHistoryForEpoch(context.Background(), pubkey[:], 0, tt.slotBits); err != nil {
			t.Fatalf("Saving proposal history failed: %v", err)
		}
		savedBits, err := db.ProposalHistoryForEpoch(context.Background(), pubkey[:], 0)
		if err != nil {
			t.Fatalf("Failed to get proposal history: %v", err)
		}

		if savedBits == nil || !reflect.DeepEqual(savedBits.Bytes(), tt.slotBits.Bytes()) {
			t.Fatalf("Expected DB to keep object the same, received: %v, expected %v", savedBits.Bytes(), tt.slotBits.Bytes())
		}
		if !savedBits.BitAt(tt.slot) {
			t.Fatalf("Expected slot %d to be marked as proposed", tt.slot)
		}
		if savedBits.BitAt(tt.slot + 1) {
			t.Fatalf("Expected slot %d to not be marked as proposed", tt.slot+1)
		}
		if savedBits.BitAt(tt.slot - 1) {
			t.Fatalf("Expected slot %d to not be marked as proposed", tt.slot-1)
		}
	}
}

func TestProposalHistoryForEpoch_MultipleEpochs(t *testing.T) {
	pubKey := [48]byte{0}
	tests := []struct {
		slots        []uint64
		expectedBits []bitfield.Bitlist
	}{
		{
			slots:        []uint64{1, 2, 8},
			expectedBits: []bitfield.Bitlist{{0x02, 0x14}},
		},
		{
			slots:        []uint64{1, 33, 8},
			expectedBits: []bitfield.Bitlist{{0x02, 0x10}, {0x02, 0x04}},
		},
		{
			slots:        []uint64{2, 34, 36},
			expectedBits: []bitfield.Bitlist{{0x02, 0x04}, {0x02, 0x06}},
		},
		{
			slots:        []uint64{32, 33, 34},
			expectedBits: []bitfield.Bitlist{{0x02, 0x00}, {0x02, 0x05}},
		},
	}

	for _, tt := range tests {
		db := SetupDB(t, [][48]byte{pubKey})
		defer TeardownDB(t, db)
		for _, slot := range tt.slots {
			slotBits, err := db.ProposalHistoryForEpoch(context.Background(), pubKey[:], helpers.SlotToEpoch(slot))
			if err != nil {
				t.Fatalf("Failed to get proposal history: %v", err)
			}
			slotBits.SetBitAt(slot%params.BeaconConfig().SlotsPerEpoch, true)
			if err := db.SaveProposalHistoryForEpoch(context.Background(), pubKey[:], helpers.SlotToEpoch(slot), slotBits); err != nil {
				t.Fatalf("Saving proposal history failed: %v", err)
			}
		}

		for i, slotBits := range tt.expectedBits {
			savedBits, err := db.ProposalHistoryForEpoch(context.Background(), pubKey[:], uint64(i))
			if err != nil {
				t.Fatalf("Failed to get proposal history: %v", err)
			}
			if !bytes.Equal(savedBits.Bytes(), slotBits.Bytes()) {
				t.Fatalf("unexpected difference in bytes, expected %#x vs received %#x", savedBits.Bytes(), slotBits.Bytes())
			}
		}
	}
}

func TestDeleteProposalHistory_OK(t *testing.T) {
	pubkey := [48]byte{2}
	db := SetupDB(t, [][48]byte{pubkey})
	defer TeardownDB(t, db)

	slotBits := bitfield.Bitlist{0x01, 0x02}

	if err := db.SaveProposalHistoryForEpoch(context.Background(), pubkey[:], 0, slotBits); err != nil {
		t.Fatalf("Save proposal history failed: %v", err)
	}
	// Making sure everything is saved.
	savedHistory, err := db.ProposalHistoryForEpoch(context.Background(), pubkey[:], 0)
	if err != nil {
		t.Fatalf("Failed to get proposal history: %v", err)
	}
	if savedHistory == nil || !bytes.Equal(savedHistory.Bytes(), slotBits.Bytes()) {
		t.Fatalf("Expected DB to keep object the same, received: %v, expected %v", savedHistory, slotBits)
	}
	if err := db.DeleteProposalHistory(context.Background(), pubkey[:]); err != nil {
		t.Fatal(err)
	}

	// Check after deleting from DB.
	_, err = db.ProposalHistoryForEpoch(context.Background(), pubkey[:], 0)
	if err == nil {
		t.Fatalf("Unexpected success in deleting history: %v", err)
	}
	if !strings.Contains(err.Error(), "validator history empty for public key ") {
		t.Fatalf("Unexpected error, received %v", err)
	}
}
