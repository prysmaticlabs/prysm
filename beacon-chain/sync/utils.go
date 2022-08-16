package sync

import (
	"errors"
	"sort"

	"github.com/prysmaticlabs/prysm/v3/consensus-types/interfaces"
)

// A type to represent beacon blocks and roots which have methods
// which satisfy the Interface in `Sort` so that this type can
// be sorted in ascending order.
type sortedObj struct {
	blks  []interfaces.SignedBeaconBlock
	roots [][32]byte
}

// Less reports whether the element with index i must sort before the element with index j.
func (s sortedObj) Less(i, j int) bool {
	return s.blks[i].Block().Slot() < s.blks[j].Block().Slot()
}

// Swap swaps the elements with indexes i and j.
func (s sortedObj) Swap(i, j int) {
	s.blks[i], s.blks[j] = s.blks[j], s.blks[i]
	s.roots[i], s.roots[j] = s.roots[j], s.roots[i]
}

// Len is the number of elements in the collection.
func (s sortedObj) Len() int {
	return len(s.blks)
}

// removes duplicates from provided blocks and roots.
func (_ *Service) dedupBlocksAndRoots(blks []interfaces.SignedBeaconBlock, roots [][32]byte) ([]interfaces.SignedBeaconBlock, [][32]byte, error) {
	if len(blks) != len(roots) {
		return nil, nil, errors.New("input blks and roots are diff lengths")
	}

	// Remove duplicate blocks received
	rootMap := make(map[[32]byte]bool, len(blks))
	newBlks := make([]interfaces.SignedBeaconBlock, 0, len(blks))
	newRoots := make([][32]byte, 0, len(roots))
	for i, r := range roots {
		if rootMap[r] {
			continue
		}
		rootMap[r] = true
		newRoots = append(newRoots, roots[i])
		newBlks = append(newBlks, blks[i])
	}
	return newBlks, newRoots, nil
}

func (_ *Service) dedupRoots(roots [][32]byte) [][32]byte {
	newRoots := make([][32]byte, 0, len(roots))
	rootMap := make(map[[32]byte]bool, len(roots))
	for i, r := range roots {
		if rootMap[r] {
			continue
		}
		rootMap[r] = true
		newRoots = append(newRoots, roots[i])
	}
	return newRoots
}

// sort the provided blocks and roots in ascending order. This method assumes that the size of
// block slice and root slice is equal.
func (_ *Service) sortBlocksAndRoots(blks []interfaces.SignedBeaconBlock, roots [][32]byte) ([]interfaces.SignedBeaconBlock, [][32]byte) {
	obj := sortedObj{
		blks:  blks,
		roots: roots,
	}
	sort.Sort(obj)
	return obj.blks, obj.roots
}
