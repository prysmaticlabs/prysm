package kv

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
	db := setupDB(t, pubkeys)

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
	db := setupDB(t, [][48]byte{})

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
	db := setupDB(t, [][48]byte{pubkey})

	epoch := uint64(2)
	slot := uint64(2)
	slotBits := bitfield.Bitlist{0x04, 0x00, 0x00, 0x00, 0x04}

	if err := db.SaveProposalHistoryForEpoch(context.Background(), pubkey[:], epoch, slotBits); err != nil {
		t.Fatalf("Saving proposal history failed: %v", err)
	}
	savedBits, err := db.ProposalHistoryForEpoch(context.Background(), pubkey[:], epoch)
	if err != nil {
		t.Fatalf("Failed to get proposal history: %v", err)
	}

	if savedBits == nil || !bytes.Equal(slotBits, savedBits) {
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
			slotBits: bitfield.Bitlist{0x02, 0x00, 0x00, 0x00, 0x02},
		},
		{
			slot:     uint64(2),
			slotBits: bitfield.Bitlist{0x04, 0x00, 0x00, 0x00, 0x04},
		},
		{
			slot:     uint64(3),
			slotBits: bitfield.Bitlist{0x08, 0x00, 0x00, 0x00, 0x08},
		},
	}

	for _, tt := range tests {
		db := setupDB(t, [][48]byte{pubkey})
		if err := db.SaveProposalHistoryForEpoch(context.Background(), pubkey[:], 0, tt.slotBits); err != nil {
			t.Fatalf("Saving proposal history failed: %v", err)
		}
		savedBits, err := db.ProposalHistoryForEpoch(context.Background(), pubkey[:], 0)
		if err != nil {
			t.Fatalf("Failed to get proposal history: %v", err)
		}

		if savedBits == nil || !reflect.DeepEqual(savedBits, tt.slotBits) {
			t.Fatalf("Expected DB to keep object the same, received: %v, expected %v", savedBits, tt.slotBits)
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
			slots:        []uint64{1, 2, 8, 31},
			expectedBits: []bitfield.Bitlist{{0b00000110, 0b00000001, 0b00000000, 0b10000000, 0b00000001}},
		},
		{
			slots: []uint64{1, 33, 8},
			expectedBits: []bitfield.Bitlist{
				{0b00000010, 0b00000001, 0b00000000, 0b00000000, 0b00000001},
				{0b00000010, 0b00000000, 0b00000000, 0b00000000, 0b00000001},
			},
		},
		{
			slots: []uint64{2, 34, 36},
			expectedBits: []bitfield.Bitlist{
				{0b00000100, 0b00000000, 0b00000000, 0b00000000, 0b00000001},
				{0b00010100, 0b00000000, 0b00000000, 0b00000000, 0b00000001},
			},
		},
		{
			slots: []uint64{32, 33, 34},
			expectedBits: []bitfield.Bitlist{
				{0, 0, 0, 0, 1},
				{0b00000111, 0b00000000, 0b00000000, 0b00000000, 0b00000001},
			},
		},
	}

	for _, tt := range tests {
		db := setupDB(t, [][48]byte{pubKey})
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
			if !bytes.Equal(slotBits, savedBits) {
				t.Fatalf("unexpected difference in bytes for slots %v, expected %v vs received %v", tt.slots, slotBits, savedBits)
			}
		}
	}
}

func TestPruneProposalHistory_OK(t *testing.T) {
	slotsPerEpoch := params.BeaconConfig().SlotsPerEpoch
	wsPeriod := params.BeaconConfig().WeakSubjectivityPeriod
	pubKey := [48]byte{0}
	tests := []struct {
		slots         []uint64
		storedEpochs  []uint64
		removedEpochs []uint64
	}{
		{
			// Go 2 epochs past pruning point.
			slots:         []uint64{slotsPerEpoch / 2, slotsPerEpoch*5 + 6, (wsPeriod+3)*slotsPerEpoch + 8},
			storedEpochs:  []uint64{5, 54003},
			removedEpochs: []uint64{0},
		},
		{
			// Go 10 epochs past pruning point.
			slots:         []uint64{slotsPerEpoch + 4, slotsPerEpoch * 2, slotsPerEpoch * 3, slotsPerEpoch * 4, slotsPerEpoch * 5, (wsPeriod+10)*slotsPerEpoch + 8},
			storedEpochs:  []uint64{54010},
			removedEpochs: []uint64{1, 2, 3, 4},
		},
		{
			// Prune none.
			slots:        []uint64{slotsPerEpoch + 4, slotsPerEpoch*2 + 3, slotsPerEpoch*3 + 4, slotsPerEpoch*4 + 3, slotsPerEpoch*5 + 3},
			storedEpochs: []uint64{1, 2, 3, 4, 5},
		},
	}

	for _, tt := range tests {
		db := setupDB(t, [][48]byte{pubKey})
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

		for _, epoch := range tt.removedEpochs {
			savedBits, err := db.ProposalHistoryForEpoch(context.Background(), pubKey[:], epoch)
			if err != nil {
				t.Fatalf("Failed to get proposal history: %v", err)
			}
			if !bytes.Equal(bitfield.NewBitlist(slotsPerEpoch), savedBits) {
				t.Fatalf("unexpected difference in bytes for epoch %d, expected %#x vs received %v", epoch, bitfield.NewBitlist(slotsPerEpoch), savedBits)
			}
		}
		for _, epoch := range tt.storedEpochs {
			savedBits, err := db.ProposalHistoryForEpoch(context.Background(), pubKey[:], epoch)
			if err != nil {
				t.Fatalf("Failed to get proposal history: %v", err)
			}
			if bytes.Equal(bitfield.NewBitlist(slotsPerEpoch), savedBits) {
				t.Fatalf("unexpected difference in bytes for epoch %d, expected %v vs received %v", epoch, bitfield.NewBitlist(slotsPerEpoch), savedBits)
			}
		}
	}
}
