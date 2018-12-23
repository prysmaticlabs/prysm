package db

import (
	"bytes"
	"testing"

	"github.com/golang/protobuf/proto"
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
	if b.GetSlot() != 0 {
		t.Fatalf("Expected block height to equal 1. Got %d", b.GetSlot())
	}

	beaconState, err := db.GetState()
	if err != nil {
		t.Fatalf("Failed to get state: %v", err)
	}
	if beaconState == nil {
		t.Fatalf("Failed to retrieve state: %v", beaconState)
	}
	beaconStateEnc, err := proto.Marshal(beaconState)
	if err != nil {
		t.Fatalf("Failed to encode state: %v", err)
	}

	statePrime, err := db.GetState()
	if err != nil {
		t.Fatalf("Failed to get state: %v", err)
	}
	statePrimeEnc, err := proto.Marshal(statePrime)
	if err != nil {
		t.Fatalf("Failed to encode state: %v", err)
	}

	if !bytes.Equal(beaconStateEnc, statePrimeEnc) {
		t.Fatalf("Expected %#x and %#x to be equal", beaconStateEnc, statePrimeEnc)
	}
}
