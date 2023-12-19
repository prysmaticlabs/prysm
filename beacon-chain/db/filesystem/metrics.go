package filesystem

import (
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	"github.com/spf13/afero"
)

var (
	blobBuckets     = []float64{0.00003, 0.00005, 0.00007, 0.00009, 0.00011, 0.00013, 0.00015}
	blobSaveLatency = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "blob_storage_save_latency",
		Help:    "Latency of blob storage save operations in seconds",
		Buckets: blobBuckets,
	})
	blobFetchLatency = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "blob_storage_get_latency",
		Help:    "Latency of blob storage get operations in seconds",
		Buckets: blobBuckets,
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

func (bs *BlobStorage) Initialize(lastFinalizedSlot primitives.Slot) error {
	if err := bs.Prune(lastFinalizedSlot); err != nil {
		return fmt.Errorf("failed to prune from finalized slot %d: %w", lastFinalizedSlot, err)
	}
	if err := bs.collectTotalBlobMetric(); err != nil {
		return fmt.Errorf("failed to initialize blob metrics: %w", err)
	}
	return nil
}

// CollectTotalBlobMetric set the number of blobs currently present in the filesystem
// to the blobsTotalGauge metric.
func (bs *BlobStorage) collectTotalBlobMetric() error {
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
