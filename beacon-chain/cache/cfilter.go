package cache

import (
	"sync"
	
	"github.com/irfansharif/cfilter"
)


type ActiveIndicesCFilter struct {
	filter *cfilter.CFilter
	lock sync.RWMutex
}




func NewCFilter() *ActiveIndicesCFilter {
	return &ActiveIndicesCFilter{filter : cfilter.New()}
}


func (cf *ActiveIndicesCFilter) InsertActiveIndicesCFilter(byteIndices [][]byte)  {
	cf.lock.Lock()
	defer cf.lock.Unlock()
	for _, i := range byteIndices {
		cf.filter.Insert(i)
	}
}

// think about delete func (cf *CFilter) Delete(item []byte) bool { insteaf of lookup
func (cf *ActiveIndicesCFilter) LookupActiveIndicesCFilter(byteIndices [][]byte) bool {
	cf.lock.Lock()
	defer cf.lock.Unlock()
	for _, i := range byteIndices {
		if !cf.filter.Lookup(i) {
			return false
		}
	}
	return true
}




