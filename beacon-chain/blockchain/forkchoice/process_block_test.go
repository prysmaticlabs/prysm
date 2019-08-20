package forkchoice

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/prysmaticlabs/prysm/beacon-chain/db/kv"
	testDB "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
)

func TestStore_OnBlock(t *testing.T) {
	ctx := context.Background()
	db := testDB.SetupDB(t)
	kv := db.(*kv.Store)
	defer testDB.TeardownDB(t, kv)

	store := NewForkChoiceService(ctx, kv)

	roots, err := blockTree1(db)
	if err != nil {
		t.Fatal(err)
	}

	randaomParentRoot := []byte{'a'}
	if err := store.db.SaveState(ctx, &pb.BeaconState{}, bytesutil.ToBytes32(randaomParentRoot)); err != nil {
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
			blk:           &ethpb.BeaconBlock{ParentRoot: randaomParentRoot, Slot: 1},
			s:             &pb.BeaconState{},
			wantErrString: "could not process block from the future, slot time 6 > current time 0",
		},
		{
			name:          "could not get finalized block",
			blk:           &ethpb.BeaconBlock{ParentRoot: randaomParentRoot},
			s:             &pb.BeaconState{},
			wantErrString: "block from slot 0 is not a descendent of the current finalized block",
		},
		{
			name:          "same slot as finalized block",
			blk:           &ethpb.BeaconBlock{Slot: 0, ParentRoot: validGenesisRoot},
			s:             &pb.BeaconState{},
			wantErrString: "block is equal or earlier than finalized block, slot 0 < slot 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := store.GenesisStore(ctx, tt.s); err != nil {
				t.Fatal(err)
			}
			store.finalizedCheckpt.Root = roots[0]
			store.lastProcessedTime = time.Unix(int64(tt.time), 0)

			err := store.OnBlock(ctx, tt.blk)
			fmt.Println(tt.wantErrString)
			if err.Error() != tt.wantErrString {
				t.Errorf("Store.OnBlock() error = %v, wantErr = %v", err, tt.wantErrString)
			}
		})
	}
}
