package cache

import (
	"github.com/prysmaticlabs/prysm/shared/params"
	"k8s.io/client-go/tools/cache"
)

var (
	// maxCacheSize is 4x of the epoch length for additional cache padding.
	// Requests should be only accessing committees within defined epoch length.
	maxCacheSize = 4 * params.BeaconConfig().SlotsPerEpoch
)

// trim the FIFO queue to the maxSize.
func trim(queue *cache.FIFO, maxSize uint64) {
	for s := uint64(len(queue.ListKeys())); s > maxSize; s-- {
		_, err := queue.Pop(popProcessNoopFunc)
		if err != nil {
			// popProcessNoopFunc never returns an error, but we handle this anyway to make linter
			// happy.
			return
		}
	}
}

// popProcessNoopFunc is a no-op function that never returns an error.
func popProcessNoopFunc(obj interface{}) error {
	return nil
}
