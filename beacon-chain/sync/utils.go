package sync

import (
	"sort"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
)

// A type to represent beacon blocks and roots which have methods
// which satisfy the Interface in `Sort` so that this type can
// be sorted in ascending order.
type sortedObj struct {
	blks  []*ethpb.SignedBeaconBlock
	roots [][32]byte
}

func (s sortedObj) Less(i, j int) bool {
	return s.blks[i].Block.Slot < s.blks[j].Block.Slot
}

func (s sortedObj) Swap(i, j int) {
	s.blks[i], s.blks[j] = s.blks[j], s.blks[i]
	s.roots[i], s.roots[j] = s.roots[j], s.roots[i]
}

func (s sortedObj) Len() int {
	return len(s.blks)
}

// removes duplicates from provided blocks and roots.
func (s *Service) dedupBlocksAndRoots(blks []*ethpb.SignedBeaconBlock, roots [][32]byte) ([]*ethpb.SignedBeaconBlock, [][32]byte) {
	// Remove duplicate blocks received
	rootMap := make(map[[32]byte]bool)
	newBlks := make([]*ethpb.SignedBeaconBlock, 0, len(blks))
	newRoots := make([][32]byte, 0, len(roots))
	for i, r := range roots {
		if rootMap[r] {
			continue
		}
		rootMap[r] = true
		newRoots = append(newRoots, roots[i])
		newBlks = append(newBlks, blks[i])
	}
	return newBlks, newRoots
}

// sort the provided blocks and roots in ascending order. This method assumes that the size of
// block slice and root slice is equal.
func (s *Service) sortBlocksAndRoots(blks []*ethpb.SignedBeaconBlock, roots [][32]byte) ([]*ethpb.SignedBeaconBlock, [][32]byte) {
	obj := sortedObj{
		blks:  blks,
		roots: roots,
	}
	sort.Sort(obj)
	return obj.blks, obj.roots
}
