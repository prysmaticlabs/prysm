package db

import (
	"github.com/boltdb/bolt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	dbPageSize = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "page_size",
		Help: "Current page size of the whole db",
	})

	numberOfTx = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "number_of_tx",
		Help: "The total number of started bolt read transactions",
	})
	numberOfOpenTx = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "number_of_open_tx",
		Help: "The number of open bolt read transactions",
	})
	allocationInFreePages = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "allocations_in_free_pages",
		Help: "total bytes allocated in free pages",
	})
	numberOfFreePages = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "number_of_free_pages",
		Help: "total number of free pages on the freelist",
	})
	pendingPages = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "pending_pages_in_freelist",
		Help: "total number of pending pages on the freelist",
	})
	sizeOfFreeList = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "size_of_FreeList",
		Help: "total bytes used by the freelist",
	})
	numOfPageAllocs = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "num_of_page_allocations",
		Help: "number of page allocations",
	})
	bytesAllocated = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "bytes_allocated_in_pages",
		Help: "total bytes allocated in pages",
	})
	numOfWrites = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "num_of_db_writes",
		Help: "number of writes performed in the db",
	})
	totalWriteTime = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "time_spent_writing_to_db",
		Help: "total time spent writing to disk in the db",
	})
)

// publishes internal metrics from boltdb to prometheus.
func publishMetrics(db *bolt.DB) {
	dbPageSize.Set(float64(db.Info().PageSize))
	numberOfTx.Set(float64(db.Stats().TxN))
	numberOfOpenTx.Set(float64(db.Stats().OpenTxN))
	allocationInFreePages.Set(float64(db.Stats().FreeAlloc))
	numberOfFreePages.Set(float64(db.Stats().FreePageN))
	pendingPages.Set(float64(db.Stats().PendingPageN))
	sizeOfFreeList.Set(float64(db.Stats().FreelistInuse))
	numOfPageAllocs.Set(float64(db.Stats().TxStats.PageCount))
	bytesAllocated.Set(float64(db.Stats().TxStats.PageAlloc))
	numOfWrites.Set(float64(db.Stats().TxStats.Write))
	totalWriteTime.Set(float64(db.Stats().TxStats.WriteTime))
}
