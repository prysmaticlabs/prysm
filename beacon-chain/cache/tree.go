package  cache

import (
    "sync"
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


//analogue AddActiveIndicesList
func (t *ActiveIndicesTree) InsertActiveIndicesTree(activeIndices []uint64) (error) {
	t.lock.Lock()
	defer t.lock.Unlock()
	if err := t.tree.ReplaceOrInsertBulk(activeIndices); err != nil {
		return err
	}
	return nil
}

//analogue ActiveIndicesInEpoch
func (t *ActiveIndicesTree) RetrieveActiveIndices() ([]uint64, error)  {
	t.lock.RLock()
	defer t.lock.RUnlock()



}










//add error handling in rrlb.go
// add erro handling in InsertActiveIndicesTree
// fix does 

//write benchmark








*/









