package  cache

import (
	"sync"
	"errors"
)

type ActiveIndicesTree struct {
	tree 	*LLRB
	lock    sync.RWMutex
}

func NewActiveIndicesTree() *ActiveIndicesTree {
	return &ActiveIndicesTree{tree: New()}
}

type index uint64
 func (i index) Less(than Item) bool {
	 return i < than.(index)
}


//InsertNoReplaceActiveIndicesTree inserts items in Left-Leaning Red-Black (LLRB) tree
//  If an element has the same order, both elements remain in the tree.
func (t *ActiveIndicesTree) InsertNoReplaceActiveIndicesTree(activeIndices []uint64) (error) {
	t.lock.Lock()
	defer t.lock.Unlock()
	for _, i := range activeIndices {
		if err := t.tree.InsertNoReplace(index(i)); err != nil {
			return err
		}
	}
	return nil
}



// InsertReplaceActiveIndicesTree inserts items into Left-Leaning Red-Black (LLRB) tree. If an 
// element has the same order, it is removed from the tree.
func (t *ActiveIndicesTree) InsertReplaceActiveIndicesTree(activeIndices []uint64) (error) {
	t.lock.Lock()
	defer t.lock.Unlock()
	for _, i := range activeIndices {
		if _, err := t.tree.ReplaceOrInsert(index(i)); err != nil {
			return err
		}
	}
	return nil
}


//RetrieveActiveIndicesTree retrieves all items from a Left-Leaning Red-Black (LLRB) tree and returns them
func (t *ActiveIndicesTree) RetrieveActiveIndicesTree() ([]index, error)  {
	t.lock.RLock()
	defer t.lock.RUnlock()

	retrievedIndices := make([]index, 0, t.tree.Len())
	t.tree.AscendGreaterOrEqual(index(1), func(i Item) bool {
		retrievedIndices = append(retrievedIndices, i.(index))
		return true
	})
    if len(retrievedIndices) == 0  {
		return nil, errors.New("retrievedIndices are empty")
    }
	return retrievedIndices, nil

}
























