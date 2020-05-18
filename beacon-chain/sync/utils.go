package sync

import (
	"sort"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
)

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

func (r *Service) sortBlocksAndRoots(blks []*ethpb.SignedBeaconBlock, roots [][32]byte) ([]*ethpb.SignedBeaconBlock, [][32]byte) {
	obj := sortedObj{
		blks:  blks,
		roots: roots,
	}
	sort.Sort(obj)
	return obj.blks, obj.roots
}
