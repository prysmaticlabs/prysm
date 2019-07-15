package  cache

import (
    "sync"

	"github.com/petar/GoLLRB/llrb"
)

type ActiveIndicesTree struct {
	epoch   uint64
	tree 	*llrb.LLRB
	lock    sync.RWMutex
}

func lessUInt64(a, b interface{}) bool { return a.(uint64) < b.(uint64) }

func NewActiveIndicesTree() *ActiveIndicesTree {
	return &ActiveIndicesTree{tree: llrb.New()}
}

func (t *ActiveIndicesTree) InsertActiveIndicesTree(activeIndices []uint64)  {
	   //add error handling
       t.tree.ReplaceOrInsertBulk(activeIndices)

}



//TODO
//copy rrlb.go here
//add error handling in rrlb.go
// add erro handling in InsertActiveIndicesTree
//

//write benchmark

/*Example 

package main

import (
	"fmt"
	"github.com/petar/GoLLRB/llrb"
)

func lessInt(a, b interface{}) bool { return a.(int) < b.(int) }

func main() {
	tree := llrb.New(lessInt)
	tree.ReplaceOrInsert(1)
	tree.ReplaceOrInsert(2)
	tree.ReplaceOrInsert(3)
	tree.ReplaceOrInsert(4)
	tree.DeleteMin()
	tree.Delete(4)
	c := tree.IterAscend()
	for {
		u := <-c
		if u == nil {
			break
		}
		fmt.Printf("%d\n", int(u.(int)))
	}
}








*/








