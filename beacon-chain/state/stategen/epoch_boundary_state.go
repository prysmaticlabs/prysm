package stategen

import (
	"errors"
	"strconv"
	"sync"

	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
	"k8s.io/client-go/tools/cache"
)

var (
	// maxCacheSize is 8. That means 8 epochs and roughly an hour
	// of no finality can be endured.
	maxCacheSize        = uint64(8)
	errNotSlotRootInfo  = errors.New("not slot root info type")
	errNotRootStateInfo = errors.New("root state info type")
)

// rootStateInfo specifies the root state info in the epoch boundary state cache.
type rootStateInfo struct {
	root  [32]byte
	state *stateTrie.BeaconState
}

// slotRootInfo specifies the slot root info in the epoch boundary state cache.
type slotRootInfo struct {
	root [32]byte
	slot uint64
}

// rootKeyFn takes the string representation of the block root to be used as key
// to retrieve epoch boundary state.
func rootKeyFn(obj interface{}) (string, error) {
	s, ok := obj.(*rootStateInfo)
	if !ok {
		return "", errNotRootStateInfo
	}
	return string(s.root[:]), nil
}

// slotKeyFn takes the string representation of the slot to be used as key
// to retrieve root.
func slotKeyFn(obj interface{}) (string, error) {
	s, ok := obj.(*slotRootInfo)
	if !ok {
		return "", errNotSlotRootInfo
	}
	return slotToString(s.slot), nil
}

// epochBoundaryState struct with two queues by looking up by slot or root.
type epochBoundaryState struct {
	rootStateCache *cache.FIFO
	slotRootCache  *cache.FIFO
	lock           sync.RWMutex
}

// newBoundaryStateCache creates a new block epochBoundaryState for storing/accessing epoch boundary states from
// memory.
func newBoundaryStateCache() *epochBoundaryState {
	return &epochBoundaryState{
		rootStateCache: cache.NewFIFO(rootKeyFn),
		slotRootCache:  cache.NewFIFO(slotKeyFn),
	}
}

// get epoch boundary state by its block root. Returns copied state in state info object if exists. Otherwise returns nil.
func (e *epochBoundaryState) getByRoot(r [32]byte) (*rootStateInfo, bool, error) {
	e.lock.RLock()
	defer e.lock.RUnlock()

	obj, exists, err := e.rootStateCache.GetByKey(string(r[:]))
	if err != nil {
		return nil, false, err
	}
	if !exists {
		return nil, false, nil
	}
	s, ok := obj.(*rootStateInfo)
	if !ok {
		return nil, false, errNotRootStateInfo
	}

	return &rootStateInfo{
		root:  r,
		state: s.state.Copy(),
	}, true, nil
}

// get epoch boundary state by its slot. Returns copied state in state info object if exists. Otherwise returns nil.
func (e *epochBoundaryState) getBySlot(s uint64) (*rootStateInfo, bool, error) {
	e.lock.RLock()
	defer e.lock.RUnlock()

	obj, exists, err := e.slotRootCache.GetByKey(slotToString(s))
	if err != nil {
		return nil, false, err
	}
	if !exists {
		return nil, false, nil
	}
	info, ok := obj.(*slotRootInfo)
	if !ok {
		return nil, false, errNotSlotRootInfo
	}

	return e.getByRoot(info.root)
}

// put adds a state to the epoch boundary state cache. This method also trims the
// least recently added state info if the cache size has reached the max cache
// size limit.
func (e *epochBoundaryState) put(r [32]byte, s *stateTrie.BeaconState) error {
	e.lock.Lock()

	if err := e.rootStateCache.AddIfNotPresent(&rootStateInfo{
		root:  r,
		state: s.Copy(),
	}); err != nil {
		return err
	}
	if err := e.slotRootCache.AddIfNotPresent(&slotRootInfo{
		root: r,
		slot: s.Slot(),
	}); err != nil {
		return err
	}

	e.lock.Unlock()

	trim(e.rootStateCache, maxCacheSize)
	trim(e.slotRootCache, maxCacheSize)

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

// Converts input uint64 to string. To be used as key for slot to get root.
func slotToString(s uint64) string {
	return strconv.FormatUint(s, 10)
}
