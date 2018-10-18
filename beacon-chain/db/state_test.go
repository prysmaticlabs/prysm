package db

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/boltdb/bolt"
	"github.com/prysmaticlabs/prysm/beacon-chain/types"
	"github.com/prysmaticlabs/prysm/beacon-chain/utils"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
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

	aState, err := db.GetActiveState()
	if err != nil {
		t.Fatalf("Failed to get active state: %v", err)
	}
	cState, err := db.GetCrystallizedState()
	if err != nil {
		t.Fatalf("Failed to get crystallized state: %v", err)
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

func TestGetUnfinalizedBlockState(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)
	aState := types.NewActiveState(&pb.ActiveState{}, map[[32]byte]*utils.VoteCache{})
	cState := types.NewCrystallizedState(&pb.CrystallizedState{})
	if err := db.SaveUnfinalizedBlockState(aState, cState); err != nil {
		t.Fatalf("Could not save unfinalized block state: %v", err)
	}

	aStateHash, err := aState.Hash()
	if err != nil {
		t.Fatal(err)
	}
	cStateHash, err := cState.Hash()
	if err != nil {
		t.Fatal(err)
	}
	got1, got2, err := db.GetUnfinalizedBlockState(aStateHash, cStateHash)
	if err != nil {
		t.Errorf("Unexpected error: wanted nil, received %v", err)
		return
	}
	if !reflect.DeepEqual(got1, aState) {
		t.Errorf("ActiveState not equal: got = %v, want %v", got1, aState)
	}
	if !reflect.DeepEqual(got2, cState) {
		t.Errorf("CrystallizedState not equal: got = %v, want %v", got2, cState)
	}
}

func TestBeaconDB_SaveUnfinalizedBlockState(t *testing.T) {
	type fields struct {
		db *bolt.DB
	}
	type args struct {
		aState *types.ActiveState
		cState *types.CrystallizedState
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := &BeaconDB{
				db: tt.fields.db,
			}
			if err := db.SaveUnfinalizedBlockState(tt.args.aState, tt.args.cState); (err != nil) != tt.wantErr {
				t.Errorf("BeaconDB.SaveUnfinalizedBlockState() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
