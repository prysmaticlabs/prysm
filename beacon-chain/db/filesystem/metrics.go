package filesystem

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/spf13/afero"
)

var (
	blobSaveLatency = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "blob_storage_save_latency",
		Help:    "Latency of blob storage save operations in milliseconds",
		Buckets: prometheus.DefBuckets,
	})
	blobFetchLatency = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "blob_storage_get_latency",
		Help:    "Latency of blob storage get operations in milliseconds",
		Buckets: prometheus.DefBuckets,
	})
	blobsPrunedCounter = promauto.NewCounter(prometheus.CounterOpts{
		Name: "blob_pruned_blobs_total",
		Help: "Total number of pruned blobs.",
	})
	blobsTotalGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "blobs_on_disk_total",
		Help: "Total number of blobs in filesystem.",
	})
)

// CollectTotalBlobMetric set the number of blobs currently present in the filesystem
// to the blobsTotalGauge metric.
func (bs *BlobStorage) CollectTotalBlobMetric() error {
	totalBlobs := 0
	folders, err := afero.ReadDir(bs.fs, ".")
	if err != nil {
		return err
	}
	for _, folder := range folders {
		num, err := bs.countFiles(folder.Name())
		if err != nil {
			return err
		}
		totalBlobs = totalBlobs + num
	}
	blobsTotalGauge.Set(float64(totalBlobs))
	return nil
}

// countFiles returns the length of blob files for a given directory.
func (bs *BlobStorage) countFiles(folderName string) (int, error) {
	files, err := afero.ReadDir(bs.fs, folderName)
	if err != nil {
		return 0, err
	}
	return len(files), nil
}
