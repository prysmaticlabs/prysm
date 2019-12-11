package helpers

import (
	"github.com/prysmaticlabs/prysm/beacon-chain/cache"
)

// ClearActiveCountCache restarts the active validator count cache from scratch.
func ClearActiveCountCache() {
	activeCountCache = cache.NewActiveCountCache()
}

// ClearAllCaches clears all the helpers caches from scratch.
func ClearAllCaches() {
	ClearActiveCountCache()
}
