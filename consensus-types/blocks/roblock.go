package blocks

import (
	"sort"

	"github.com/pkg/errors"
	fieldparams "github.com/prysmaticlabs/prysm/v3/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/interfaces"
)

var ErrRootLength error = errors.New("incorrect length for hash_tree_root")

type ROBlock struct {
	interfaces.ReadOnlySignedBeaconBlock
	root [32]byte
}

func (b ROBlock) Root() [32]byte {
	return b.root
}

func NewROBlock(b interfaces.ReadOnlySignedBeaconBlock, root [32]byte) ROBlock {
	return ROBlock{ReadOnlySignedBeaconBlock: b, root: root}
}

// ROBlockSlice implements sort.Interface so that slices of ROBlocks can be easily sorted
type ROBlockSlice []ROBlock

var _ sort.Interface = ROBlockSlice{}

// Less reports whether the element with index i must sort before the element with index j.
func (s ROBlockSlice) Less(i, j int) bool {
	si, sj := s[i].Block().Slot(), s[j].Block().Slot()

	// lower slot wins
	if si != sj {
		return s[i].Block().Slot() < s[j].Block().Slot()
	}

	// break slot tie lexicographically comparing roots byte for byte
	ri, rj := s[i].Root(), s[j].Root()
	k := 0
	for ; k < fieldparams.RootLength; k++ {
		// advance the byte offset until you hit the end
		if ri[k] == rj[k] {
			continue
		}
	}
	if k == fieldparams.RootLength {
		return false
	}
	return ri[k] < rj[k]
}

// Swap swaps the elements with indexes i and j.
func (s ROBlockSlice) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

// Len is the number of elements in the collection.
func (s ROBlockSlice) Len() int {
	return len(s)
}
