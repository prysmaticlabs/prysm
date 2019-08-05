package cache

import (
	"errors"
	"reflect"
	"testing"

	"github.com/petar/GoLLRB/llrb"
)

func TestInsertNoReplaceActiveIndicesTree(t *testing.T) {
	activeIndicesTree := NewActiveIndicesTree()
	activeIndices := []uint64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}

	activeIndicesTree.InsertNoReplaceActiveIndicesTree(activeIndices)

	i := 1
	activeIndicesTree.tree.AscendGreaterOrEqual(index(1),
		func(item llrb.Item) bool {
			if item.(index) != index(i) {
				t.Errorf("bad order: got %d, expect %d", item.(index), index(i))
			}
			i++
			return true
		})

	if activeIndicesTree.tree.Len() != 10 {
		t.Errorf("Wanted: %v, got: %v", 10, activeIndicesTree.tree.Len())
	}

	activeIndicesTree.InsertNoReplaceActiveIndicesTree(activeIndices)

	if activeIndicesTree.tree.Len() != 20 {
		t.Errorf("Wanted: %v, got: %v", 20, activeIndicesTree.tree.Len())
	}

}

func TestInsertReplaceActiveIndicesTree(t *testing.T) {
	activeIndicesTree := NewActiveIndicesTree()
	activeIndices := []uint64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}

	activeIndicesTree.InsertReplaceActiveIndicesTree(activeIndices)

	if activeIndicesTree.tree.Len() != 10 {
		t.Errorf("Wanted: %v, got: %v", 10, activeIndicesTree.tree.Len())
	}

	activeIndicesTree.InsertReplaceActiveIndicesTree(activeIndices)

	if activeIndicesTree.tree.Len() != 10 {
		t.Errorf("Wanted: %v, got: %v", 10, activeIndicesTree.tree.Len())
	}

	i := 1
	activeIndicesTree.tree.AscendGreaterOrEqual(index(1),
		func(item llrb.Item) bool {
			if item.(index) != index(i) {
				t.Errorf("bad order: got %d, expect %d", item.(index), index(i))
			}
			i++
			return true
		})

}

func TestRetrieveActiveIndicesTree(t *testing.T) {

	var ErrRetrievedIndicesEmpty = errors.New("retrievedIndices are empty")
	activeIndicesTree := NewActiveIndicesTree()
	activeIndices := []uint64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}

	activeIndicesTree.InsertReplaceActiveIndicesTree(activeIndices)

	retrievedIndices, err := activeIndicesTree.RetrieveActiveIndicesTree()
	if err != nil {
		t.Errorf("No error expected")
	}
	if !reflect.DeepEqual(activeIndices, retrievedIndices) {
		t.Errorf("Wrong retrieved indices received")
	}

	for i := uint64(1); i < 11; i++ {
		activeIndicesTree.tree.Delete(index(i))
	}

	if activeIndicesTree.tree.Len() != 0 {
		t.Errorf("Wanted: %v, got: %v", 0, activeIndicesTree.tree.Len())
	}

    retrievedIndices, err = activeIndicesTree.RetrieveActiveIndicesTree()
    if retrievedIndices != nil  {
		t.Errorf("retrievedIndices expected to be nil")
	}
    if !reflect.DeepEqual(err, ErrRetrievedIndicesEmpty) {
        t.Errorf("Wanted: %v, got: %v", ErrRetrievedIndicesEmpty, err)
	}

}
