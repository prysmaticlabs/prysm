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

	aState, cState, err := db.GetState()
	if err != nil {
		t.Fatalf("Failed to get state: %v", err)
	}
	if aState == nil || cState == nil {
		t.Fatalf("Failed to retrieve state: %v, %v", aState, cState)
	}
	aStateEnc, err := aState.Marshal()
	if err != nil {
		t.Fatalf("Failed to encode active state: %v", err)
	}
	cStateEnc, err := cState.Marshal()
	if err != nil {
		t.Fatalf("Failed t oencode crystallized state: %v", err)
	}

	aStatePrime, err := db.GetActiveState()
	if err != nil {
		t.Fatalf("Failed to get active state: %v", err)
	}
	aStatePrimeEnc, err := aStatePrime.Marshal()
	if err != nil {
		t.Fatalf("Failed to encode active state: %v", err)
	}

	cStatePrime, err := db.GetCrystallizedState()
	if err != nil {
		t.Fatalf("Failed to get crystallized state: %v", err)
	}
	cStatePrimeEnc, err := cStatePrime.Marshal()
	if err != nil {
		t.Fatalf("Failed to encode crystallized state: %v", err)
	}

	if !bytes.Equal(aStateEnc, aStatePrimeEnc) {
		t.Fatalf("Expected %#x and %#x to be equal", aStateEnc, aStatePrimeEnc)
	}
	if !bytes.Equal(cStateEnc, cStatePrimeEnc) {
		t.Fatalf("Expected %#x and %#x to be equal", cStateEnc, cStatePrimeEnc)
	}
}
