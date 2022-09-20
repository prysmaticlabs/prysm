package sync

import (
	"math/rand"
	"testing"

	"github.com/prysmaticlabs/prysm/v3/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/interfaces"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v3/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
)

func TestSortedObj_SortBlocksRoots(t *testing.T) {
	source := rand.NewSource(33)
	randGen := rand.New(source)
	var blks []interfaces.SignedBeaconBlock
	var roots [][32]byte
	randFunc := func() int64 {
		return randGen.Int63n(50)
	}

	for i := 0; i < 10; i++ {
		slot := types.Slot(randFunc())
		newBlk, err := blocks.NewSignedBeaconBlock(&ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{Slot: slot, Body: &ethpb.BeaconBlockBody{}}})
		require.NoError(t, err)
		blks = append(blks, newBlk)
		root := bytesutil.ToBytes32(bytesutil.Bytes32(uint64(slot)))
		roots = append(roots, root)
	}

	r := &Service{}

	newBlks, newRoots := r.sortBlocksAndRoots(blks, roots)

	previousSlot := types.Slot(0)
	for i, b := range newBlks {
		if b.Block().Slot() < previousSlot {
			t.Errorf("Block list is not sorted as %d is smaller than previousSlot %d", b.Block().Slot(), previousSlot)
		}
		if bytesutil.FromBytes8(newRoots[i][:]) != uint64(b.Block().Slot()) {
			t.Errorf("root doesn't match stored slot in block: wanted %d but got %d", b.Block().Slot(), bytesutil.FromBytes8(newRoots[i][:]))
		}
		previousSlot = b.Block().Slot()
	}
}

func TestSortedObj_NoDuplicates(t *testing.T) {
	source := rand.NewSource(33)
	randGen := rand.New(source)
	var blks []interfaces.SignedBeaconBlock
	var roots [][32]byte
	randFunc := func() int64 {
		return randGen.Int63n(50)
	}

	for i := 0; i < 10; i++ {
		slot := types.Slot(randFunc())
		newBlk := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{Slot: slot, Body: &ethpb.BeaconBlockBody{}}}
		// append twice
		wsb, err := blocks.NewSignedBeaconBlock(newBlk)
		require.NoError(t, err)
		wsbCopy, err := wsb.Copy()
		require.NoError(t, err)
		blks = append(blks, wsb, wsbCopy)

		// append twice
		root := bytesutil.ToBytes32(bytesutil.Bytes32(uint64(slot)))
		roots = append(roots, root, root)
	}

	r := &Service{}

	newBlks, newRoots, err := r.dedupBlocksAndRoots(blks, roots)
	require.NoError(t, err)

	rootMap := make(map[[32]byte]bool)
	for i, b := range newBlks {
		if rootMap[newRoots[i]] {
			t.Errorf("Duplicated root exists %#x with block %v", newRoots[i], b)
		}
		rootMap[newRoots[i]] = true
	}
}
