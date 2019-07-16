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
// https://github.com/seppestas/data-structure-bench/blob/master/llrb/llrb.go
// https://github.com/kubernetes/client-go/blob/master/tools/cache/fifo.go
 func (i index) Less(than Item) bool {
	 return i < than.(index)
}


// type indices []index
// func (idxs indices) Less(than Item) bool {
// 	for _, i := range idxs {
// 		return i < than.(index)
// 	}
// }



//analogue AddActiveIndicesList
func (t *ActiveIndicesTree) InsertActiveIndicesTree(activeIndices []uint64) (error) {
	t.lock.Lock()
	defer t.lock.Unlock()
	for _, i := range activeIndices {
		if _, err := t.tree.ReplaceOrInsert(index(i)); err != nil {
			return err
		}
	}
	return nil
}

//analogue ActiveIndicesInEpoch
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
























