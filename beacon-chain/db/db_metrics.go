package db

import (
	"github.com/boltdb/bolt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/log"
)

// register's the databases metrics
func registerDBMetrics(db *bolt.DB) {
	errChan := make(chan error, 10)

	err := prometheus.Register(prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Name: "beaconchainDB_page_size",
		Help: "Current page size of the whole db",
	}, func() float64 {
		return float64(db.Info().PageSize)
	}))
	errChan <- err

	err = prometheus.Register(prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Name: "beaconchainDB_number_of_tx",
		Help: "The total number of started bolt read transactions",
	}, func() float64 {
		return float64(db.Stats().TxN)
	}))
	errChan <- err

	err = prometheus.Register(prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Name: "beaconchainDB_number_of_open_tx",
		Help: "The number of open bolt read transactions",
	}, func() float64 {
		return float64(db.Stats().OpenTxN)
	}))
	errChan <- err

	err = prometheus.Register(prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Name: "beaconchainDB_allocations_in_free_pages",
		Help: "total bytes allocated in free pages",
	}, func() float64 {
		return float64(db.Stats().FreeAlloc)
	}))
	errChan <- err

	err = prometheus.Register(prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Name: "beaconchainDB_number_of_free_pages",
		Help: "total number of free pages on the freelist",
	}, func() float64 {
		return float64(db.Stats().FreePageN)
	}))
	errChan <- err

	err = prometheus.Register(prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Name: "beaconchainDB_pending_pages_in_freelist",
		Help: "total number of pending pages on the freelist",
	}, func() float64 {
		return float64(db.Stats().PendingPageN)
	}))
	errChan <- err

	err = prometheus.Register(prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Name: "beaconchainDB_size_of_FreeList",
		Help: "total bytes used by the freelist",
	}, func() float64 {
		return float64(db.Stats().FreelistInuse)
	}))
	errChan <- err

	err = prometheus.Register(prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Name: "beaconchainDB_num_of_page_allocations",
		Help: "number of page allocations",
	}, func() float64 {
		return float64(db.Stats().TxStats.PageCount)
	}))
	errChan <- err

	err = prometheus.Register(prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Name: "beaconchainDB_bytes_allocated_in_pages",
		Help: "total bytes allocated in pages",
	}, func() float64 {
		return float64(db.Stats().TxStats.PageAlloc)
	}))
	errChan <- err

	err = prometheus.Register(prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Name: "beaconchainDB_num_of_db_writes",
		Help: "number of writes performed in the db",
	}, func() float64 {
		return float64(db.Stats().TxStats.Write)
	}))
	errChan <- err

	err = prometheus.Register(prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Name: "beaconchainDB_time_spent_writing_to_db",
		Help: "total time spent writing to disk in the db",
	}, func() float64 {
		return float64(db.Stats().TxStats.WriteTime)
	}))
	errChan <- err

	for err := range errChan {
		if err != nil {
			log.Errorf("Could not register metric: %v", err)
		}
	}
}
