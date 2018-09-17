// Package params defines important configuration options to be used when instantiating
// objects within the sharding package. For example, it defines objects such as a
// Config that will be useful when creating new shard instances.
package params

import (
	"math"
)

// SlotDuration in seconds.
var SlotDuration = 8

// DefaultConfig returns pointer to a Config value with same defaults.
func DefaultConfig() *Config {
	return &Config{
		CollationSizeLimit: DefaultCollationSizeLimit(),
	}
}

// DefaultCollationSizeLimit is the integer value representing the maximum
// number of bytes allowed in a given collation.
func DefaultCollationSizeLimit() int64 {
	return int64(math.Pow(float64(2), float64(20)))
}

// Config contains configs for node to participate in the sharded universe.
type Config struct {
	CollationSizeLimit int64 // CollationSizeLimit is the maximum size the serialized blobs in a collation can take.
}
