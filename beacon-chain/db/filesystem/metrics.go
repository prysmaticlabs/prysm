package filesystem

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	blobBuckets     = []float64{0.00003, 0.00005, 0.00007, 0.00009, 0.00011, 0.00013, 0.00015}
	blobSaveLatency = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "blob_storage_save_latency",
		Help:    "Latency of BlobSidecar storage save operations in seconds",
		Buckets: blobBuckets,
	})
	blobFetchLatency = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "blob_storage_get_latency",
		Help:    "Latency of BlobSidecar storage get operations in seconds",
		Buckets: blobBuckets,
	})
	blobsPrunedCounter = promauto.NewCounter(prometheus.CounterOpts{
		Name: "blob_pruned",
		Help: "Number of BlobSidecar files pruned.",
	})
	blobsWrittenCounter = promauto.NewCounter(prometheus.CounterOpts{
		Name: "blobs_written",
		Help: "Number of BlobSidecar files written.",
	})
)
