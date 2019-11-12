package helpers

import (
	"github.com/prysmaticlabs/prysm/beacon-chain/cache"
)

// ClearActiveCountCache restarts the active validator count cache from scratch.
func ClearActiveCountCache() {
	activeCountCache = cache.NewActiveCountCache()
}

// ClearActiveIndicesCache restarts the active validator indices cache from scratch.
func ClearActiveIndicesCache() {
	activeIndicesCache = cache.NewActiveIndicesCache()
}

// ActiveIndicesKeys returns the keys of the active indices cache.
func ActiveIndicesKeys() []string {
	return activeIndicesCache.ActiveIndicesKeys()
}

// ClearAllCaches clears all the helpers caches from scratch.
func ClearAllCaches() {
	ClearActiveIndicesCache()
	ClearActiveCountCache()
}
