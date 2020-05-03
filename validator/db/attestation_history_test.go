package db

import (
	"context"
	"reflect"
	"testing"

	slashpb "github.com/prysmaticlabs/prysm/proto/slashing"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func TestAttestationHistory_EmptyVal(t *testing.T) {
	pubkeys := [][48]byte{{30}, {25}, {20}}
	db := SetupDB(t, pubkeys)

	for _, pub := range pubkeys {
		attestationHistory, err := db.AttestationHistory(context.Background(), pub[:])
		if err != nil {
			t.Fatal(err)
		}

		newMap := make(map[uint64]uint64)
		newMap[0] = params.BeaconConfig().FarFutureEpoch
		clean := &slashpb.AttestationHistory{
			TargetToSource: newMap,
		}
		if !reflect.DeepEqual(attestationHistory, clean) {
			t.Fatalf("Expected attestation history epoch bits to be empty, received %v", attestationHistory)
		}
	}
}

func TestSaveAttestationHistory_OK(t *testing.T) {
	db := SetupDB(t, [][48]byte{})

	pubkey := []byte{3}
	epoch := uint64(2)
	farFuture := params.BeaconConfig().FarFutureEpoch
	newMap := make(map[uint64]uint64)
	// The validator attested at target epoch 2 but had no attestations for target epochs 0 and 1.
	newMap[0] = farFuture
	newMap[1] = farFuture
	newMap[epoch] = 1
	history := &slashpb.AttestationHistory{
		TargetToSource:     newMap,
		LatestEpochWritten: 2,
	}

	if err := db.SaveAttestationHistory(context.Background(), pubkey, history); err != nil {
		t.Fatalf("Saving attestation history failed: %v", err)
	}
	savedHistory, err := db.AttestationHistory(context.Background(), pubkey)
	if err != nil {
		t.Fatalf("Failed to get attestation history: %v", err)
	}

	if savedHistory == nil || !reflect.DeepEqual(history, savedHistory) {
		t.Fatalf("Expected DB to keep object the same, received: %v", history)
	}
	if savedHistory.TargetToSource[epoch] != newMap[epoch] {
		t.Fatalf("Expected target epoch %d to have the same marked source epoch, received %d", epoch, savedHistory.TargetToSource[epoch])
	}
	if savedHistory.TargetToSource[epoch-1] != farFuture {
		t.Fatalf("Expected target epoch %d to not be marked as attested for, received %d ", epoch-1, savedHistory.TargetToSource[epoch-1])
	}
	if savedHistory.TargetToSource[epoch-2] != farFuture {
		t.Fatalf("Expected target epoch %d to not be marked as attested for, received %d", epoch-2, savedHistory.TargetToSource[epoch-2])
	}
}

func TestSaveAttestationHistory_Overwrites(t *testing.T) {
	db := SetupDB(t, [][48]byte{})
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
		pubkey  []byte
		epoch   uint64
		history *slashpb.AttestationHistory
	}{
		{
			pubkey: []byte{0},
			epoch:  uint64(1),
			history: &slashpb.AttestationHistory{
				TargetToSource:     newMap1,
				LatestEpochWritten: 1,
			},
		},
		{
			pubkey: []byte{0},
			epoch:  uint64(2),
			history: &slashpb.AttestationHistory{
				TargetToSource:     newMap2,
				LatestEpochWritten: 2,
			},
		},
		{
			pubkey: []byte{0},
			epoch:  uint64(3),
			history: &slashpb.AttestationHistory{
				TargetToSource:     newMap3,
				LatestEpochWritten: 3,
			},
		},
	}

	for _, tt := range tests {
		if err := db.SaveAttestationHistory(context.Background(), tt.pubkey, tt.history); err != nil {
			t.Fatalf("Saving attestation history failed: %v", err)
		}
		history, err := db.AttestationHistory(context.Background(), tt.pubkey)
		if err != nil {
			t.Fatalf("Failed to get attestation history: %v", err)
		}

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
	db := SetupDB(t, [][48]byte{})

	pubkey := []byte{2}
	newMap := make(map[uint64]uint64)
	newMap[0] = params.BeaconConfig().FarFutureEpoch
	newMap[1] = 0
	history := &slashpb.AttestationHistory{
		TargetToSource:     newMap,
		LatestEpochWritten: 1,
	}

	if err := db.SaveAttestationHistory(context.Background(), pubkey, history); err != nil {
		t.Fatalf("Save attestation history failed: %v", err)
	}
	// Making sure everything is saved.
	savedHistory, err := db.AttestationHistory(context.Background(), pubkey)
	if err != nil {
		t.Fatalf("Failed to get attestation history: %v", err)
	}
	if savedHistory == nil || !reflect.DeepEqual(savedHistory, history) {
		t.Fatalf("Expected DB to keep object the same, received: %v, expected %v", savedHistory, history)
	}
	if err := db.DeleteAttestationHistory(context.Background(), pubkey); err != nil {
		t.Fatal(err)
	}

	// Check after deleting from DB.
	savedHistory, err = db.AttestationHistory(context.Background(), pubkey)
	if err != nil {
		t.Fatalf("Failed to get attestation history: %v", err)
	}
	cleanMap := make(map[uint64]uint64)
	cleanMap[0] = params.BeaconConfig().FarFutureEpoch
	clean := &slashpb.AttestationHistory{
		TargetToSource: cleanMap,
	}
	if !reflect.DeepEqual(savedHistory, clean) {
		t.Fatalf("Expected attestation history to be %v, received %v", clean, savedHistory)
	}
}
