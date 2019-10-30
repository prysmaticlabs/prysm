package kv

import (
	"context"
	"testing"

	"github.com/prysmaticlabs/go-ssz"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
)

var genesisBlockRoot = bytesutil.ToBytes32([]byte{'G', 'E', 'N', 'E', 'S', 'I', 'S'})

func init() {
	fc := featureconfig.Get()
	fc.EnableFinalizedBlockRootIndex = true
	featureconfig.Init(fc)
}

func TestStore_IsFinalizedBlock(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)
	ctx := context.Background()

	if err := db.SaveGenesisBlockRoot(ctx, genesisBlockRoot); err != nil {
		t.Fatal(err)
	}

	blks := makeBlocks(t, 128)
	if err := db.SaveBlocks(ctx, blks); err != nil {
		t.Fatal(err)
	}

	root, err := ssz.SigningRoot(blks[64])
	if err != nil {
		t.Fatal(err)
	}

	cp := &ethpb.Checkpoint{
		Epoch: 1,
		Root:  root[:],
	}

	// a state is required to save checkpoint
	if err := db.SaveState(ctx, &pb.BeaconState{}, root); err != nil {
		t.Fatal(err)
	}

	if err := db.SaveFinalizedCheckpoint(ctx, cp); err != nil {
		t.Fatal(err)
	}

	// All blocks up to 64 should be in the finalized index.
	for i := 0; i <= 64; i++ {
		root, err := ssz.SigningRoot(blks[i])
		if err != nil {
			t.Fatal(err)
		}
		if !db.IsFinalizedBlock(ctx, root) {
			t.Errorf("Block at index %d was not considered finalized in the index", i)
		}
	}
	for i := 65; i < len(blks); i++ {
		root, err := ssz.SigningRoot(blks[i])
		if err != nil {
			t.Fatal(err)
		}
		if db.IsFinalizedBlock(ctx, root) {
			t.Errorf("Block at index %d was considered finalized in the index, but should not have", i)
		}
	}
}

func makeBlocks(t *testing.T, n int) []*ethpb.BeaconBlock {
	previousRoot := genesisBlockRoot
	blocks := make([]*ethpb.BeaconBlock, n)
	for i := 0; i < n; i++ {
		parentRoot := make([]byte, 32)
		copy(parentRoot, previousRoot[:])
		blocks[i] = &ethpb.BeaconBlock{
			Slot:       uint64(i + 1),
			ParentRoot: parentRoot,
		}
		var err error
		previousRoot, err = ssz.SigningRoot(blocks[i])
		if err != nil {
			t.Fatal(err)
		}
	}
	return blocks
}
