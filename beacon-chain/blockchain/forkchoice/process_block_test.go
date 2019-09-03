package forkchoice

import (
	"context"
	"strings"
	"testing"

	testDB "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func TestStore_OnBlock(t *testing.T) {
	ctx := context.Background()
	db := testDB.SetupDB(t)
	defer testDB.TeardownDB(t, db)

	store := NewForkChoiceService(ctx, db)

	roots, err := blockTree1(db)
	if err != nil {
		t.Fatal(err)
	}

	randomParentRoot := []byte{'a'}
	if err := store.db.SaveState(ctx, &pb.BeaconState{}, bytesutil.ToBytes32(randomParentRoot)); err != nil {
		t.Fatal(err)
	}
	randomParentRoot2 := roots[1]
	if err := store.db.SaveState(ctx, &pb.BeaconState{}, bytesutil.ToBytes32(randomParentRoot2)); err != nil {
		t.Fatal(err)
	}
	validGenesisRoot := []byte{'g'}
	if err := store.db.SaveState(ctx, &pb.BeaconState{}, bytesutil.ToBytes32(validGenesisRoot)); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name          string
		blk           *ethpb.BeaconBlock
		s             *pb.BeaconState
		time          uint64
		wantErrString string
	}{
		{
			name:          "parent block root does not have a state",
			blk:           &ethpb.BeaconBlock{},
			s:             &pb.BeaconState{},
			wantErrString: "pre state of slot 0 does not exist",
		},
		{
			name:          "block is from the feature",
			blk:           &ethpb.BeaconBlock{ParentRoot: randomParentRoot, Slot: params.BeaconConfig().FarFutureEpoch},
			s:             &pb.BeaconState{},
			wantErrString: "could not process block from the future",
		},
		{
			name:          "could not get finalized block",
			blk:           &ethpb.BeaconBlock{ParentRoot: randomParentRoot},
			s:             &pb.BeaconState{},
			wantErrString: "block from slot 0 is not a descendent of the current finalized block",
		},
		{
			name:          "same slot as finalized block",
			blk:           &ethpb.BeaconBlock{Slot: 0, ParentRoot: randomParentRoot2},
			s:             &pb.BeaconState{},
			wantErrString: "block is equal or earlier than finalized block, slot 0 < slot 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := store.GenesisStore(ctx, &ethpb.Checkpoint{}, &ethpb.Checkpoint{}); err != nil {
				t.Fatal(err)
			}
			store.finalizedCheckpt.Root = roots[0]

			err := store.OnBlock(ctx, tt.blk)
			if !strings.Contains(err.Error(), tt.wantErrString) {
				t.Errorf("Store.OnBlock() error = %v, wantErr = %v", err, tt.wantErrString)
			}
		})
	}
}

func TestStore_SaveNewValidators(t *testing.T) {
	ctx := context.Background()
	db := testDB.SetupDB(t)
	defer testDB.TeardownDB(t, db)

	store := NewForkChoiceService(ctx, db)
	preCount := 2 // validators 0 and validators 1
	s := &pb.BeaconState{Validators: []*ethpb.Validator{
		{PublicKey: []byte{0}}, {PublicKey: []byte{1}},
		{PublicKey: []byte{2}}, {PublicKey: []byte{3}},
	}}
	if err := store.saveNewValidators(ctx, preCount, s); err != nil {
		t.Fatal(err)
	}

	if !db.HasValidatorIndex(ctx, bytesutil.ToBytes48([]byte{2})) {
		t.Error("Wanted validator saved in db")
	}
	if !db.HasValidatorIndex(ctx, bytesutil.ToBytes48([]byte{3})) {
		t.Error("Wanted validator saved in db")
	}
	if db.HasValidatorIndex(ctx, bytesutil.ToBytes48([]byte{1})) {
		t.Error("validator not suppose to be saved in db")
	}
}
