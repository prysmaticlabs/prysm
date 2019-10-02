package initialsync

import (
	"context"
	"testing"

	"github.com/prysmaticlabs/go-ssz"
	dbTest "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
)

func TestService_HasParent(t *testing.T) {
	db := dbTest.SetupDB(t)
	defer dbTest.TeardownDB(t, db)
	ctx := context.Background()
	s := &InitialSync{
		db: db,
	}

	block := &ethpb.BeaconBlock{
		Slot:       20,
		ParentRoot: []byte{1, 2, 3},
	}
	parentRoot, err := ssz.SigningRoot(block)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.SaveBlock(ctx, block); err != nil {
		t.Fatal(err)
	}
	if s.parentExists(ctx, block) {
		t.Error("Expected block parent not to exist in the db")
	}
	childBlock := &ethpb.BeaconBlock{
		Slot:       20,
		ParentRoot: parentRoot[:],
	}
	if err := db.SaveBlock(ctx, childBlock); err != nil {
		t.Fatal(err)
	}

}
