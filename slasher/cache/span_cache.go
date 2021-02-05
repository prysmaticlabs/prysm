package cache

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// epochSpansCacheSize defines the max number of epoch spans the cache can hold.
	epochSpansCacheSize = 256
	// Metrics for the span cache.
	epochSpansCacheHit = promauto.NewCounter(prometheus.CounterOpts{
		Name: "epoch_spans_cache_hit",
		Help: "The total number of cache hits on the epoch spans cache.",
	})
	epochSpansCacheMiss = promauto.NewCounter(prometheus.CounterOpts{
		Name: "epoch_spans_cache_miss",
		Help: "The total number of cache misses on the epoch spans cache.",
	})
)
