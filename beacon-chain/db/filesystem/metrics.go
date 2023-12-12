package filesystem

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	blobSaveLatency = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "blob_storage_save_latency_seconds",
		Help:    "Latency of blob storage save operations",
		Buckets: prometheus.DefBuckets,
	})
	blobFetchLatency = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "blob_storage_get_latency_seconds",
		Help:    "Latency of blob storage get operations",
		Buckets: prometheus.DefBuckets,
	})
	blobsPrunedCounter = promauto.NewCounter(prometheus.CounterOpts{
		Name: "blob_pruned_blobs_total",
		Help: "Total number of pruned blobs.",
	})
	blobsTotalCounter = promauto.NewCounter(prometheus.CounterOpts{
		Name: "blob_total_number",
		Help: "Total number of blobs in filesystem.",
	})
)
