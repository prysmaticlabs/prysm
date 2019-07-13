package helpers

import (
	"github.com/prysmaticlabs/prysm/beacon-chain/cache"
)

// ClearShuffledValidatorCache clears the shuffled indices cache from scratch.
func ClearShuffledValidatorCache() {
	shuffledIndicesCache = cache.NewShuffledIndicesCache()
}

// ClearStartShardCache clears the start shard cache from scratch.
func ClearStartShardCache() {
	startShardCache = cache.NewStartShardCache()
}

// ClearTotalActiveBalanceCache restarts the total active validator balance cache from scratch.
func ClearTotalActiveBalanceCache() {
	totalActiveBalanceCache = cache.NewActiveBalanceCache()
}

// ClearCurrentEpochSeed clears the current epoch seed.
func ClearCurrentEpochSeed() {
	currentEpochSeed = cache.NewSeedCache()
}

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
	ClearStartShardCache()
	ClearShuffledValidatorCache()
	ClearTotalActiveBalanceCache()
	ClearCurrentEpochSeed()
}
