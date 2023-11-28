package blocks

import (
	"bytes"
	"sort"

	"github.com/prysmaticlabs/prysm/v4/consensus-types/interfaces"
)

// ROBlock is a value that embeds a ReadOnlySignedBeaconBlock along with its block root ([32]byte).
// This allows the block root to be cached within a value that satisfies the ReadOnlySignedBeaconBlock interface.
// Since the root and slot for each ROBlock is known, slices can be efficiently sorted using ROBlockSlice.
type ROBlock struct {
	interfaces.ReadOnlySignedBeaconBlock
	root [32]byte
}

// Root returns the block hash_tree_root for the embedded ReadOnlySignedBeaconBlock.Block().
func (b ROBlock) Root() [32]byte {
	return b.root
}

// NewROBlockWithRoot creates an ROBlock embedding the given block with its root. It accepts the root as parameter rather than
// computing it internally, because in some cases a block is retrieved by its root and recomputing it is a waste.
func NewROBlockWithRoot(b interfaces.ReadOnlySignedBeaconBlock, root [32]byte) (ROBlock, error) {
	if err := BeaconBlockIsNil(b); err != nil {
		return ROBlock{}, err
	}
	return ROBlock{ReadOnlySignedBeaconBlock: b, root: root}, nil
}

// NewROBlock creates a ROBlock from a ReadOnlySignedBeaconBlock. It uses the HashTreeRoot method of the given
// ReadOnlySignedBeaconBlock.Block to compute the cached root.
func NewROBlock(b interfaces.ReadOnlySignedBeaconBlock) (ROBlock, error) {
	if err := BeaconBlockIsNil(b); err != nil {
		return ROBlock{}, err
	}
	root, err := b.Block().HashTreeRoot()
	if err != nil {
		return ROBlock{}, err
	}
	return ROBlock{ReadOnlySignedBeaconBlock: b, root: root}, nil
}

// ROBlockSlice implements sort.Interface so that slices of ROBlocks can be easily sorted.
// A slice of ROBlock is sorted first by slot, with ties broken by cached block roots.
type ROBlockSlice []ROBlock

var _ sort.Interface = ROBlockSlice{}

// Less reports whether the element with index i must sort before the element with index j.
// ROBlocks are ordered first by their slot,
// with a lexicographic sort of roots breaking ties for slots with duplicate blocks.
func (s ROBlockSlice) Less(i, j int) bool {
	si, sj := s[i].Block().Slot(), s[j].Block().Slot()

	// lower slot wins
	if si != sj {
		return si < sj
	}

	// break slot tie lexicographically comparing roots byte for byte
	ri, rj := s[i].Root(), s[j].Root()
	return bytes.Compare(ri[:], rj[:]) < 0
}

// Swap swaps the elements with indexes i and j.
func (s ROBlockSlice) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

// Len is the number of elements in the collection.
func (s ROBlockSlice) Len() int {
	return len(s)
}

type BlockWithVerifiedBlobs struct {
	Block ROBlock
	Blobs []ROBlob
}

type BlockWithVerifiedBlobsSlice []BlockWithVerifiedBlobs

func (s BlockWithVerifiedBlobsSlice) ROBlocks() []ROBlock {
	r := make([]ROBlock, len(s))
	for i := range s {
		r[i] = s[i].Block
	}
	return r
}

// Less reports whether the element with index i must sort before the element with index j.
// ROBlocks are ordered first by their slot,
// with a lexicographic sort of roots breaking ties for slots with duplicate blocks.
func (s BlockWithVerifiedBlobsSlice) Less(i, j int) bool {
	si, sj := s[i].Block.Block().Slot(), s[j].Block.Block().Slot()

	// lower slot wins
	if si != sj {
		return si < sj
	}

	// break slot tie lexicographically comparing roots byte for byte
	ri, rj := s[i].Block.Root(), s[j].Block.Root()
	return bytes.Compare(ri[:], rj[:]) < 0
}

// Swap swaps the elements with indexes i and j.
func (s BlockWithVerifiedBlobsSlice) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

// Len is the number of elements in the collection.
func (s BlockWithVerifiedBlobsSlice) Len() int {
	return len(s)
}
