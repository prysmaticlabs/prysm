package db

import (
	"bytes"
	"testing"
)

func TestInitializeState(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)

	if err := db.InitializeState(nil); err != nil {
		t.Fatalf("Failed to initialize state: %v", err)
	}
	b, err := db.GetChainHead()
	if err != nil {
		t.Fatalf("Failed to get chain head: %v", err)
	}
	if b.SlotNumber() != 0 {
		t.Fatalf("Expected block height to equal 1. Got %d", b.SlotNumber())
	}

	beaconState, err := db.GetState()
	if err != nil {
		t.Fatalf("Failed to get state: %v", err)
	}
	if beaconState == nil {
		t.Fatalf("Failed to retrieve state: %v", beaconState)
	}
	beaconStateEnc, err := beaconState.Marshal()
	if err != nil {
		t.Fatalf("Failed to encode state: %v", err)
	}

	statePrime, err := db.GetState()
	if err != nil {
		t.Fatalf("Failed to get state: %v", err)
	}
	statePrimeEnc, err := statePrime.Marshal()
	if err != nil {
		t.Fatalf("Failed to encode state: %v", err)
	}

	if !bytes.Equal(beaconStateEnc, statePrimeEnc) {
		t.Fatalf("Expected %#x and %#x to be equal", beaconStateEnc, statePrimeEnc)
	}
}
