package cache

import (
	"errors"
	"sync"

	"github.com/petar/GoLLRB/llrb"
)

type ActiveIndicesTree struct {
	tree *llrb.LLRB
	lock sync.RWMutex
}

func NewActiveIndicesTree() *ActiveIndicesTree {
	return &ActiveIndicesTree{tree: llrb.New()}
}

type index uint64

func (i index) Less(than llrb.Item) bool {
	return i < than.(index)
}

//InsertNoReplaceActiveIndicesTree inserts items in Left-Leaning Red-Black (LLRB) tree
//  If an element has the same order, both elements remain in the tree.
func (t *ActiveIndicesTree) InsertNoReplaceActiveIndicesTree(activeIndices []uint64) {
	t.lock.Lock()
	defer t.lock.Unlock()
	for _, i := range activeIndices {
		t.tree.InsertNoReplace(index(i))
	}
}

// InsertReplaceActiveIndicesTree inserts items into Left-Leaning Red-Black (LLRB) tree. If an
// element has the same order, it is removed from the tree.
func (t *ActiveIndicesTree) InsertReplaceActiveIndicesTree(activeIndices []uint64) {
	t.lock.Lock()
	defer t.lock.Unlock()
	for _, i := range activeIndices {
		t.tree.ReplaceOrInsert(index(i))
	}
}

//RetrieveActiveIndicesTree retrieves all items from a Left-Leaning Red-Black (LLRB) tree and returns them
func (t *ActiveIndicesTree) RetrieveActiveIndicesTree() ([]uint64, error) {
	t.lock.RLock()
	defer t.lock.RUnlock()

	retrievedIndices := make([]uint64, 0, t.tree.Len())
	t.tree.AscendGreaterOrEqual(index(0), func(i llrb.Item) bool {
		item := i.(index)
		retrievedIndices = append(retrievedIndices, uint64(item))
		return true
	})
	if len(retrievedIndices) == 0 {
		return nil, errors.New("retrievedIndices are empty")
	}
	return retrievedIndices, nil

}
