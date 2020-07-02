package stategen

import (
	"errors"
	"sync"

	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
	"k8s.io/client-go/tools/cache"
)

var (
	// maxCacheSize is 8. That means 8 epochs and roughly an hour
	// of no finality can be endured.
	maxCacheSize = uint64(8)
)

// stateInfo specifies the state info in the epoch boundary cache.
type stateInfo struct {
	root     [32]byte
	state   *stateTrie.BeaconState
}

// rootKeyFn takes the string representation of the block root to be used as key
// to retrieve state.
func rootKeyFn(obj interface{}) (string, error) {
	s, ok := obj.(*stateInfo)
	if !ok {
		return "", errors.New("input is not state info type")
	}
	return string(s.root[:]), nil
}

// epochBoundaryState struct with one queue by looking up by root.
type epochBoundaryState struct {
	cache   *cache.FIFO
	lock        sync.RWMutex
}

// newBoundaryStateCache creates a new block cache for storing/accessing epoch boundary states from
// memory.
func newBoundaryStateCache() *epochBoundaryState {
	return &epochBoundaryState{
		cache:   cache.NewFIFO(rootKeyFn),
	}
}

// gets epoch boundary state by its block root. Returns state if exists. Otherwise returns nil.
func (e *epochBoundaryState) get(r [32]byte) (*stateTrie.BeaconState, bool, error) {
	e.lock.RLock()
	defer e.lock.RUnlock()

	obj, exists, err := e.cache.GetByKey(string(r[:]))
	if err != nil {
		return nil, false, err
	}
	if !exists {
		return nil, false, nil
	}

	s, ok := obj.(*stateInfo)
	if !ok {
		return nil, false, errors.New("obj is not state info type")
	}

	return s.state, true, nil
}

// put adds a stateInfo object to the cache. This method also trims the
// least recently added state info if the cache size has reached the max cache
// size limit.
func (e *epochBoundaryState) put(r [32]byte, s *stateTrie.BeaconState) error {
	e.lock.Lock()

	if err := e.cache.AddIfNotPresent(&stateInfo{
		root:  r,
		state: s,
	}); err != nil {
		return err
	}

	e.lock.Unlock()

	trim(e.cache, maxCacheSize)

	return nil
}

// clear deletes all the epoch boundary states in the cache.
func (e *epochBoundaryState) clear() error {
	e.lock.Lock()
	defer e.lock.Unlock()

	items := e.cache.List()
	for _, item := range items {
		if err := e.cache.Delete(item); err != nil {
			return err
		}
	}

	return nil
}

// trim the FIFO queue to the maxSize.
func trim(queue *cache.FIFO, maxSize uint64) {
	for s := uint64(len(queue.ListKeys())); s > maxSize; s-- {
		// #nosec G104 popProcessNoopFunc never returns an error
		if _, err := queue.Pop(popProcessNoopFunc); err != nil { // This never returns an error, but we'll handle anyway for sanity.
			panic(err)
		}
	}
}

// popProcessNoopFunc is a no-op function that never returns an error.
func popProcessNoopFunc(obj interface{}) error {
	return nil
}
