package sync

import (
	"math/rand"
	"testing"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
)

func TestSortedObj_SortBlocksRoots(t *testing.T) {
	source := rand.NewSource(33)
	randGen := rand.New(source)
	blks := []*ethpb.SignedBeaconBlock{}
	roots := [][32]byte{}
	randFunc := func() int64 {
		return randGen.Int63n(50)
	}

	for i := 0; i < 10; i++ {
		slot := uint64(randFunc())
		newBlk := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{Slot: slot}}
		blks = append(blks, newBlk)
		root := bytesutil.ToBytes32(bytesutil.Bytes32(slot))
		roots = append(roots, root)
	}

	r := &Service{}

	newBlks, newRoots := r.sortBlocksAndRoots(blks, roots)

	previousSlot := uint64(0)
	for i, b := range newBlks {
		if b.Block.Slot < previousSlot {
			t.Errorf("Block list is not sorted as %d is smaller than previousSlot %d", b.Block.Slot, previousSlot)
		}
		if bytesutil.FromBytes8(newRoots[i][:]) != b.Block.Slot {
			t.Errorf("root doesn't match stored slot in block: wanted %d but got %d", b.Block.Slot, bytesutil.FromBytes8(newRoots[i][:]))
		}
		previousSlot = b.Block.Slot
	}
}

func TestSortedObj_NoDuplicates(t *testing.T) {
	source := rand.NewSource(33)
	randGen := rand.New(source)
	blks := []*ethpb.SignedBeaconBlock{}
	roots := [][32]byte{}
	randFunc := func() int64 {
		return randGen.Int63n(50)
	}

	for i := 0; i < 10; i++ {
		slot := uint64(randFunc())
		newBlk := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{Slot: slot}}
		// append twice
		blks = append(blks, newBlk)
		blks = append(blks, newBlk)

		// append twice
		root := bytesutil.ToBytes32(bytesutil.Bytes32(slot))
		roots = append(roots, root)
		roots = append(roots, root)

	}

	r := &Service{}

	newBlks, newRoots := r.dedupBlocksAndRoots(blks, roots)

	rootMap := make(map[[32]byte]bool)
	for i, b := range newBlks {
		if rootMap[newRoots[i]] {
			t.Errorf("Duplicated root exists %#x with block %v", newRoots[i], b)
		}
		rootMap[newRoots[i]] = true
	}
}
