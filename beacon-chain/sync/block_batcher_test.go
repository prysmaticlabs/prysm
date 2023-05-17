package sync

import (
	"math/rand"
	"testing"

	"github.com/prysmaticlabs/prysm/v4/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v4/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v4/testing/require"
)

func TestSortedObj_SortBlocksRoots(t *testing.T) {
	source := rand.NewSource(33)
	randGen := rand.New(source)
	randFunc := func() int64 {
		return randGen.Int63n(50)
	}

	var blks []blocks.ROBlock
	for i := 0; i < 10; i++ {
		slot := primitives.Slot(randFunc())
		newBlk, err := blocks.NewSignedBeaconBlock(&ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{Slot: slot, Body: &ethpb.BeaconBlockBody{}}})
		require.NoError(t, err)
		root := bytesutil.ToBytes32(bytesutil.Bytes32(uint64(slot)))
		b, err := blocks.NewROBlockWithRoot(newBlk, root)
		require.NoError(t, err)
		blks = append(blks, b)
	}

	newBlks := sortedUniqueBlocks(blks)
	previousSlot := primitives.Slot(0)
	for _, b := range newBlks {
		if b.Block().Slot() < previousSlot {
			t.Errorf("Block list is not sorted as %d is smaller than previousSlot %d", b.Block().Slot(), previousSlot)
		}
		previousSlot = b.Block().Slot()
	}
}

func TestSortedObj_NoDuplicates(t *testing.T) {
	source := rand.NewSource(33)
	randGen := rand.New(source)
	var blks []blocks.ROBlock
	randFunc := func() int64 {
		return randGen.Int63n(50)
	}

	for i := 0; i < 10; i++ {
		slot := primitives.Slot(randFunc())
		newBlk := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{Slot: slot, Body: &ethpb.BeaconBlockBody{}}}
		// append twice
		wsb, err := blocks.NewSignedBeaconBlock(newBlk)
		require.NoError(t, err)
		wsbCopy, err := wsb.Copy()
		require.NoError(t, err)
		root := bytesutil.ToBytes32(bytesutil.Bytes32(uint64(slot)))
		b, err := blocks.NewROBlockWithRoot(wsb, root)
		require.NoError(t, err)
		b2, err := blocks.NewROBlockWithRoot(wsbCopy, root)
		require.NoError(t, err)
		blks = append(blks, b, b2)
	}

	dedup := sortedUniqueBlocks(blks)
	roots := make(map[[32]byte]int)
	for i, b := range dedup {
		if di, dup := roots[b.Root()]; dup {
			t.Errorf("Duplicated root %#x at index %d and %d", b.Root(), di, i)
		}
		roots[b.Root()] = i
	}
}
