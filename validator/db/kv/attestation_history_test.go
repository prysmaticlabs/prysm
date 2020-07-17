package kv

import (
	"context"
	"reflect"
	"testing"

	slashpb "github.com/prysmaticlabs/prysm/proto/slashing"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func TestAttestationHistoryForPubKeys_EmptyVals(t *testing.T) {
	pubkeys := [][48]byte{{30}, {25}, {20}}
	db := setupDB(t, pubkeys)

	historyForPubKeys, err := db.AttestationHistoryForPubKeys(context.Background(), pubkeys)
	if err != nil {
		t.Fatal(err)
	}

	newMap := make(map[uint64]uint64)
	newMap[0] = params.BeaconConfig().FarFutureEpoch
	clean := &slashpb.AttestationHistory{
		TargetToSource: newMap,
	}
	cleanAttHistoryForPubKeys := make(map[[48]byte]*slashpb.AttestationHistory)
	for _, pubKey := range pubkeys {
		cleanAttHistoryForPubKeys[pubKey] = clean
	}

	if !reflect.DeepEqual(cleanAttHistoryForPubKeys, historyForPubKeys) {
		t.Fatalf(
			"Expected attestation history epoch bits to be empty, expected %v received %v",
			cleanAttHistoryForPubKeys,
			historyForPubKeys,
		)
	}
}

func TestSaveAttestationHistory_OK(t *testing.T) {
	pubKeys := [][48]byte{{3}, {4}}
	db := setupDB(t, pubKeys)

	farFuture := params.BeaconConfig().FarFutureEpoch
	newMap := make(map[uint64]uint64)
	// The validator attested at target epoch 2 but had no attestations for target epochs 0 and 1.
	newMap[0] = farFuture
	newMap[1] = farFuture
	newMap[2] = 1
	history := &slashpb.AttestationHistory{
		TargetToSource:     newMap,
		LatestEpochWritten: 2,
	}

	newMap2 := make(map[uint64]uint64)
	// The validator attested at target epoch 1 and 3 but had no attestations for target epochs 0 and 2.
	newMap2[0] = farFuture
	newMap2[1] = 0
	newMap2[2] = farFuture
	newMap2[3] = 2
	history2 := &slashpb.AttestationHistory{
		TargetToSource:     newMap2,
		LatestEpochWritten: 3,
	}

	attestationHistory := make(map[[48]byte]*slashpb.AttestationHistory)
	attestationHistory[pubKeys[0]] = history
	attestationHistory[pubKeys[1]] = history2

	if err := db.SaveAttestationHistoryForPubKeys(context.Background(), attestationHistory); err != nil {
		t.Fatalf("Saving attestation history failed: %v", err)
	}
	savedHistories, err := db.AttestationHistoryForPubKeys(context.Background(), pubKeys)
	if err != nil {
		t.Fatalf("Failed to get attestation history: %v", err)
	}

	if savedHistories == nil || !reflect.DeepEqual(attestationHistory, savedHistories) {
		t.Fatalf("Expected DB to keep object the same, received: %v", history)
	}

	savedHistory := savedHistories[pubKeys[0]]
	if savedHistory.TargetToSource[2] != newMap[2] {
		t.Fatalf("Expected target epoch %d to have the same marked source epoch, received %d", 2, savedHistory.TargetToSource[2])
	}
	if savedHistory.TargetToSource[1] != newMap[1] {
		t.Fatalf("Expected target epoch %d to not be marked as attested for, received %d ", 1, savedHistory.TargetToSource[1])
	}
	if savedHistory.TargetToSource[0] != newMap[0] {
		t.Fatalf("Expected target epoch %d to not be marked as attested for, received %d", 0, savedHistory.TargetToSource[0])
	}

	savedHistory = savedHistories[pubKeys[1]]
	if savedHistory.TargetToSource[3] != newMap2[3] {
		t.Fatalf("Expected target epoch %d to have the same marked source epoch, received %d", 3, savedHistory.TargetToSource[3])
	}
	if savedHistory.TargetToSource[2] != newMap2[2] {
		t.Fatalf("Expected target epoch %d to not be marked as attested for, received %d ", 2, savedHistory.TargetToSource[2])
	}
	if savedHistory.TargetToSource[1] != newMap2[1] {
		t.Fatalf("Expected target epoch %d to not be marked as attested for, received %d", 1, savedHistory.TargetToSource[1])
	}
}

func TestSaveAttestationHistory_Overwrites(t *testing.T) {
	db := setupDB(t, [][48]byte{})
	farFuture := params.BeaconConfig().FarFutureEpoch
	newMap1 := make(map[uint64]uint64)
	newMap1[0] = farFuture
	newMap1[1] = 0
	newMap2 := make(map[uint64]uint64)
	newMap2[0] = farFuture
	newMap2[1] = farFuture
	newMap2[2] = 1
	newMap3 := make(map[uint64]uint64)
	newMap3[0] = farFuture
	newMap3[1] = farFuture
	newMap3[2] = farFuture
	newMap3[3] = 2
	tests := []struct {
		pubkey  [48]byte
		epoch   uint64
		history *slashpb.AttestationHistory
	}{
		{
			pubkey: [48]byte{0},
			epoch:  uint64(1),
			history: &slashpb.AttestationHistory{
				TargetToSource:     newMap1,
				LatestEpochWritten: 1,
			},
		},
		{
			pubkey: [48]byte{0},
			epoch:  uint64(2),
			history: &slashpb.AttestationHistory{
				TargetToSource:     newMap2,
				LatestEpochWritten: 2,
			},
		},
		{
			pubkey: [48]byte{0},
			epoch:  uint64(3),
			history: &slashpb.AttestationHistory{
				TargetToSource:     newMap3,
				LatestEpochWritten: 3,
			},
		},
	}

	for _, tt := range tests {
		attHistory := make(map[[48]byte]*slashpb.AttestationHistory)
		attHistory[tt.pubkey] = tt.history
		if err := db.SaveAttestationHistoryForPubKeys(context.Background(), attHistory); err != nil {
			t.Fatalf("Saving attestation history failed: %v", err)
		}
		histories, err := db.AttestationHistoryForPubKeys(context.Background(), [][48]byte{tt.pubkey})
		if err != nil {
			t.Fatalf("Failed to get attestation history: %v", err)
		}

		history := histories[tt.pubkey]
		if history == nil || !reflect.DeepEqual(history, tt.history) {
			t.Fatalf("Expected DB to keep object the same, received: %v", history)
		}
		if history.TargetToSource[tt.epoch] != tt.epoch-1 {
			t.Fatalf("Expected target epoch %d to be marked with correct source epoch %d", tt.epoch, history.TargetToSource[tt.epoch])
		}
		if history.TargetToSource[tt.epoch-1] != farFuture {
			t.Fatalf("Expected target epoch %d to not be marked as attested for, received %d", tt.epoch-1, history.TargetToSource[tt.epoch-1])
		}
	}
}

func TestDeleteAttestationHistory_OK(t *testing.T) {
	pubkey := [48]byte{2}
	db := setupDB(t, [][48]byte{pubkey})

	newMap := make(map[uint64]uint64)
	newMap[0] = params.BeaconConfig().FarFutureEpoch
	newMap[1] = 0
	history := &slashpb.AttestationHistory{
		TargetToSource:     newMap,
		LatestEpochWritten: 1,
	}

	histories := make(map[[48]byte]*slashpb.AttestationHistory)
	histories[pubkey] = history
	if err := db.SaveAttestationHistoryForPubKeys(context.Background(), histories); err != nil {
		t.Fatalf("Save attestation history failed: %v", err)
	}
	// Making sure everything is saved.
	savedHistories, err := db.AttestationHistoryForPubKeys(context.Background(), [][48]byte{pubkey})
	if err != nil {
		t.Fatalf("Failed to get attestation history: %v", err)
	}
	savedHistory := savedHistories[pubkey]
	if savedHistory == nil || !reflect.DeepEqual(savedHistory, history) {
		t.Fatalf("Expected DB to keep object the same, received: %v, expected %v", savedHistory, history)
	}
	if err := db.DeleteAttestationHistory(context.Background(), pubkey[:]); err != nil {
		t.Fatal(err)
	}

	// Check after deleting from DB.
	savedHistories, err = db.AttestationHistoryForPubKeys(context.Background(), [][48]byte{pubkey})
	if err != nil {
		t.Fatalf("Failed to get attestation history: %v", err)
	}
	cleanMap := make(map[uint64]uint64)
	cleanMap[0] = params.BeaconConfig().FarFutureEpoch
	clean := &slashpb.AttestationHistory{
		TargetToSource: cleanMap,
	}
	if !reflect.DeepEqual(savedHistories[pubkey], clean) {
		t.Fatalf("Expected attestation history to be %v, received %v", clean, savedHistory)
	}
}
