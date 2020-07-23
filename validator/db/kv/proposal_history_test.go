package kv

import (
	"bytes"
	"context"
	"testing"

	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestProposalHistoryForEpoch_InitializesNewPubKeys(t *testing.T) {
	pubkeys := [][48]byte{{30}, {25}, {20}}
	db := setupDB(t, pubkeys)

	for _, pub := range pubkeys {
		slotBits, err := db.ProposalHistoryForEpoch(context.Background(), pub[:], 0)
		require.NoError(t, err)

		cleanBits := bitfield.NewBitlist(params.BeaconConfig().SlotsPerEpoch)
		require.DeepEqual(t, cleanBits.Bytes(), slotBits.Bytes(), "Expected proposal history slot bits to be empty")
	}
}

func TestProposalHistoryForEpoch_NilDB(t *testing.T) {
	valPubkey := [48]byte{1, 2, 3}
	db := setupDB(t, [][48]byte{})

	_, err := db.ProposalHistoryForEpoch(context.Background(), valPubkey[:], 0)
	require.ErrorContains(t, "validator history empty for public key", err, "Unexpected error for nil DB")
}

func TestSaveProposalHistoryForEpoch_OK(t *testing.T) {
	pubkey := [48]byte{3}
	db := setupDB(t, [][48]byte{pubkey})

	epoch := uint64(2)
	slot := uint64(2)
	slotBits := bitfield.Bitlist{0x04, 0x00, 0x00, 0x00, 0x04}

	err := db.SaveProposalHistoryForEpoch(context.Background(), pubkey[:], epoch, slotBits)
	require.NoError(t, err, "Saving proposal history failed: %v")
	savedBits, err := db.ProposalHistoryForEpoch(context.Background(), pubkey[:], epoch)
	require.NoError(t, err, "Failed to get proposal history")

	require.NotNil(t, savedBits)
	require.DeepEqual(t, slotBits, savedBits, "Expected DB to keep object the same")
	require.Equal(t, true, savedBits.BitAt(slot), "Expected slot %d to be marked as proposed", slot)
	require.Equal(t, false, savedBits.BitAt(slot+1), "Expected slot %d to not be marked as proposed", slot+1)
	require.Equal(t, false, savedBits.BitAt(slot-1), "Expected slot %d to not be marked as proposed", slot-1)
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
		err := db.SaveProposalHistoryForEpoch(context.Background(), pubkey[:], 0, tt.slotBits)
		require.NoError(t, err, "Saving proposal history failed")
		savedBits, err := db.ProposalHistoryForEpoch(context.Background(), pubkey[:], 0)
		require.NoError(t, err, "Failed to get proposal history")

		require.NotNil(t, savedBits)
		require.DeepEqual(t, tt.slotBits, savedBits, "Expected DB to keep object the same")
		require.Equal(t, true, savedBits.BitAt(tt.slot), "Expected slot %d to be marked as proposed", tt.slot)
		require.Equal(t, false, savedBits.BitAt(tt.slot+1), "Expected slot %d to not be marked as proposed", tt.slot+1)
		require.Equal(t, false, savedBits.BitAt(tt.slot-1), "Expected slot %d to not be marked as proposed", tt.slot-1)
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
			require.NoError(t, err, "Failed to get proposal history")
			slotBits.SetBitAt(slot%params.BeaconConfig().SlotsPerEpoch, true)
			err = db.SaveProposalHistoryForEpoch(context.Background(), pubKey[:], helpers.SlotToEpoch(slot), slotBits)
			require.NoError(t, err, "Saving proposal history failed")
		}

		for i, slotBits := range tt.expectedBits {
			savedBits, err := db.ProposalHistoryForEpoch(context.Background(), pubKey[:], uint64(i))
			require.NoError(t, err, "Failed to get proposal history")
			require.DeepEqual(t, slotBits, savedBits, "Unexpected difference in bytes for slots %v", tt.slots)
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
			slots: []uint64{
				slotsPerEpoch + 4, slotsPerEpoch * 2,
				slotsPerEpoch * 3, slotsPerEpoch * 4,
				slotsPerEpoch * 5, (wsPeriod+10)*slotsPerEpoch + 8,
			},
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
			require.NoError(t, err, "Failed to get proposal history")
			slotBits.SetBitAt(slot%params.BeaconConfig().SlotsPerEpoch, true)
			err = db.SaveProposalHistoryForEpoch(context.Background(), pubKey[:], helpers.SlotToEpoch(slot), slotBits)
			require.NoError(t, err, "Saving proposal history failed")
		}

		for _, epoch := range tt.removedEpochs {
			savedBits, err := db.ProposalHistoryForEpoch(context.Background(), pubKey[:], epoch)
			require.NoError(t, err, "Failed to get proposal history")
			require.DeepEqual(t, bitfield.NewBitlist(slotsPerEpoch), savedBits, "Unexpected difference in bytes for epoch %d", epoch)
		}
		for _, epoch := range tt.storedEpochs {
			savedBits, err := db.ProposalHistoryForEpoch(context.Background(), pubKey[:], epoch)
			require.NoError(t, err, "Failed to get proposal history")
			if bytes.Equal(bitfield.NewBitlist(slotsPerEpoch), savedBits) {
				t.Fatalf("unexpected difference in bytes for epoch %d, expected %v vs received %v", epoch, bitfield.NewBitlist(slotsPerEpoch), savedBits)
			}
		}
	}
}
