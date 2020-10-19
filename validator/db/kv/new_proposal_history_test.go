package kv

import (
	"context"
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestProposalHistoryForSlot_InitializesNewPubKeys(t *testing.T) {
	pubkeys := [][48]byte{{30}, {25}, {20}}
	db := setupDB(t, pubkeys)

	for _, pub := range pubkeys {
		signingRoot, err := db.ProposalHistoryForSlot(context.Background(), pub[:], 0)
		require.NoError(t, err)
		expected := bytesutil.PadTo([]byte{}, 32)
		require.DeepEqual(t, expected, signingRoot, "Expected proposal history slot signing root to be empty")
	}
}

func TestNewProposalHistoryForSlot_NilDB(t *testing.T) {
	valPubkey := [48]byte{1, 2, 3}
	db := setupDB(t, [][48]byte{})

	_, err := db.ProposalHistoryForSlot(context.Background(), valPubkey[:], 0)
	require.ErrorContains(t, "validator history empty for public key", err, "Unexpected error for nil DB")
}

func TestSaveProposalHistoryForSlot_OK(t *testing.T) {
	pubkey := [48]byte{3}
	db := setupDB(t, [][48]byte{pubkey})

	slot := uint64(2)

	err := db.SaveProposalHistoryForSlot(context.Background(), pubkey[:], slot, []byte{1})
	require.NoError(t, err, "Saving proposal history failed: %v")
	signingRoot, err := db.ProposalHistoryForSlot(context.Background(), pubkey[:], slot)
	require.NoError(t, err, "Failed to get proposal history")

	require.NotNil(t, signingRoot)
	require.DeepEqual(t, bytesutil.PadTo([]byte{1}, 32), signingRoot, "Expected DB to keep object the same")
}

func TestSaveProposalHistoryForSlot_Empty(t *testing.T) {
	pubkey := [48]byte{3}
	db := setupDB(t, [][48]byte{pubkey})

	slot := uint64(2)
	emptySlot := uint64(120)
	err := db.SaveProposalHistoryForSlot(context.Background(), pubkey[:], slot, []byte{1})
	require.NoError(t, err, "Saving proposal history failed: %v")
	signingRoot, err := db.ProposalHistoryForSlot(context.Background(), pubkey[:], emptySlot)
	require.NoError(t, err, "Failed to get proposal history")

	require.NotNil(t, signingRoot)
	require.DeepEqual(t, bytesutil.PadTo([]byte{}, 32), signingRoot, "Expected DB to keep object the same")
}

func TestSaveProposalHistoryForSlot_Overwrites(t *testing.T) {
	pubkey := [48]byte{0}
	tests := []struct {
		slot        uint64
		signingRoot []byte
	}{
		{
			slot:        uint64(1),
			signingRoot: bytesutil.PadTo([]byte{1}, 32),
		},
		{
			slot:        uint64(2),
			signingRoot: bytesutil.PadTo([]byte{2}, 32),
		},
		{
			slot:        uint64(1),
			signingRoot: bytesutil.PadTo([]byte{3}, 32),
		},
	}

	for _, tt := range tests {
		db := setupDB(t, [][48]byte{pubkey})
		err := db.SaveProposalHistoryForSlot(context.Background(), pubkey[:], 0, tt.signingRoot)
		require.NoError(t, err, "Saving proposal history failed")
		signingRoot, err := db.ProposalHistoryForSlot(context.Background(), pubkey[:], 0)
		require.NoError(t, err, "Failed to get proposal history")

		require.NotNil(t, signingRoot)
		require.DeepEqual(t, tt.signingRoot, signingRoot, "Expected DB to keep object the same")
	}
}

func TestPruneProposalHistoryBySlot_OK(t *testing.T) {
	slotsPerEpoch := params.BeaconConfig().SlotsPerEpoch
	wsPeriod := params.BeaconConfig().WeakSubjectivityPeriod
	pubKey := [48]byte{0}
	tests := []struct {
		slots        []uint64
		storedSlots  []uint64
		removedSlots []uint64
	}{
		{
			// Go 2 epochs past pruning point.
			slots:        []uint64{slotsPerEpoch / 2, slotsPerEpoch*5 + 6, (wsPeriod+3)*slotsPerEpoch + 8},
			storedSlots:  []uint64{slotsPerEpoch*5 + 6, (wsPeriod+3)*slotsPerEpoch + 8},
			removedSlots: []uint64{slotsPerEpoch / 2},
		},
		{
			// Go 10 epochs past pruning point.
			slots: []uint64{
				slotsPerEpoch + 4,
				slotsPerEpoch * 2,
				slotsPerEpoch * 3,
				slotsPerEpoch * 4,
				slotsPerEpoch * 5,
				(wsPeriod+10)*slotsPerEpoch + 8,
			},
			storedSlots: []uint64{(wsPeriod+10)*slotsPerEpoch + 8},
			removedSlots: []uint64{
				slotsPerEpoch + 4,
				slotsPerEpoch * 2,
				slotsPerEpoch * 3,
				slotsPerEpoch * 4,
				slotsPerEpoch * 5,
			},
		},
		{
			// Prune none.
			slots:       []uint64{slotsPerEpoch + 4, slotsPerEpoch*2 + 3, slotsPerEpoch*3 + 4, slotsPerEpoch*4 + 3, slotsPerEpoch*5 + 3},
			storedSlots: []uint64{slotsPerEpoch + 4, slotsPerEpoch*2 + 3, slotsPerEpoch*3 + 4, slotsPerEpoch*4 + 3, slotsPerEpoch*5 + 3},
		},
	}
	signedRoot := bytesutil.PadTo([]byte{1}, 32)

	for _, tt := range tests {
		db := setupDB(t, [][48]byte{pubKey})
		for _, slot := range tt.slots {
			err := db.SaveProposalHistoryForSlot(context.Background(), pubKey[:], slot, signedRoot)
			require.NoError(t, err, "Saving proposal history failed")
		}

		for _, slot := range tt.removedSlots {
			sr, err := db.ProposalHistoryForSlot(context.Background(), pubKey[:], slot)
			require.NoError(t, err, "Failed to get proposal history")
			require.DeepEqual(t, bytesutil.PadTo([]byte{}, 32), sr, "Unexpected difference in bytes for epoch %d", slot)
		}
		for _, slot := range tt.storedSlots {
			sr, err := db.ProposalHistoryForSlot(context.Background(), pubKey[:], slot)
			require.NoError(t, err, "Failed to get proposal history")
			require.DeepEqual(t, signedRoot, sr, "Unexpected difference in bytes for epoch %d", slot)
		}
	}
}

func TestStore_ImportProposalHistory(t *testing.T) {
	pubkey := [48]byte{3}
	ctx := context.Background()
	db := setupDB(t, [][48]byte{pubkey})
	proposedSlots := make(map[uint64]bool)
	proposedSlots[0] = true
	proposedSlots[1] = true
	proposedSlots[20] = true
	proposedSlots[31] = true
	proposedSlots[32] = true
	proposedSlots[33] = true
	proposedSlots[1023] = true
	proposedSlots[1024] = true
	proposedSlots[1025] = true
	lastIndex := 1025 + params.BeaconConfig().SlotsPerEpoch

	for slot := range proposedSlots {
		slotBitlist, err := db.ProposalHistoryForEpoch(context.Background(), pubkey[:], helpers.SlotToEpoch(slot))
		require.NoError(t, err)
		slotBitlist.SetBitAt(slot%params.BeaconConfig().SlotsPerEpoch, true)
		err = db.SaveProposalHistoryForEpoch(context.Background(), pubkey[:], helpers.SlotToEpoch(slot), slotBitlist)
		require.NoError(t, err)
	}
	err := db.MigrateV2ProposalFormat(ctx)
	require.NoError(t, err)

	for slot := uint64(0); slot <= lastIndex; slot++ {
		if _, ok := proposedSlots[slot]; ok {
			root, err := db.ProposalHistoryForSlot(ctx, pubkey[:], slot)
			require.NoError(t, err)
			require.DeepEqual(t, bytesutil.PadTo([]byte{1}, 32), root, "slot: %d", slot)
			continue
		}
		root, err := db.ProposalHistoryForSlot(ctx, pubkey[:], slot)
		require.NoError(t, err)
		require.DeepEqual(t, bytesutil.PadTo([]byte{}, 32), root)
	}
}

func TestShouldImportProposals(t *testing.T) {
	pubkey := [48]byte{3}
	db := setupDB(t, nil)
	//ctx := context.Background()

	shouldImport, err := db.shouldImportProposals()
	require.NoError(t, err)
	require.Equal(t, false, shouldImport, "Empty bucket should not be imported")
	err = db.OldUpdatePublicKeysBuckets([][48]byte{pubkey})
	require.NoError(t, err)
	shouldImport, err = db.shouldImportProposals()
	require.NoError(t, err)
	require.Equal(t, true, shouldImport, "Bucket with content should be imported")
}

func TestStore_UpdateProposalsProtectionDb(t *testing.T) {
	pubkey := [48]byte{3}
	db := setupDB(t, [][48]byte{pubkey})
	ctx := context.Background()
	err := db.OldUpdatePublicKeysBuckets([][48]byte{pubkey})
	require.NoError(t, err)
	shouldImport, err := db.shouldImportProposals()
	require.NoError(t, err)
	require.Equal(t, true, shouldImport, "Bucket with content should be imported")
	err = db.MigrateV2ProposalsProtectionDb(ctx)
	require.NoError(t, err)
	shouldImport, err = db.shouldImportProposals()
	require.NoError(t, err)
	require.Equal(t, false, shouldImport, "Proposals should not be re-imported")
}
