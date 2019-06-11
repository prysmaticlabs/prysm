package db

import (
	"github.com/boltdb/bolt"
	"github.com/prometheus/client_golang/prometheus"
)

// register's the databases metrics
func registerDBMetrics(db *bolt.DB) {

	err := prometheus.Register(prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Name: "beaconchain_db_page_size",
		Help: "Current page size of the whole db",
	}, func() float64 {
		return float64(db.Info().PageSize)
	}))
	if err != nil {
		log.Errorf("Could not register metric: %v", err)
	}

	err = prometheus.Register(prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Name: "beaconchain_db_number_of_tx",
		Help: "The total number of started bolt read transactions",
	}, func() float64 {
		return float64(db.Stats().TxN)
	}))
	if err != nil {
		log.Errorf("Could not register metric: %v", err)
	}

	err = prometheus.Register(prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Name: "beaconchain_db_number_of_open_tx",
		Help: "The number of open bolt read transactions",
	}, func() float64 {
		return float64(db.Stats().OpenTxN)
	}))
	if err != nil {
		log.Errorf("Could not register metric: %v", err)
	}

	err = prometheus.Register(prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Name: "beaconchain_db_allocations_in_free_pages",
		Help: "total bytes allocated in free pages",
	}, func() float64 {
		return float64(db.Stats().FreeAlloc)
	}))
	if err != nil {
		log.Errorf("Could not register metric: %v", err)
	}

	err = prometheus.Register(prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Name: "beaconchain_db_number_of_free_pages",
		Help: "total number of free pages on the freelist",
	}, func() float64 {
		return float64(db.Stats().FreePageN)
	}))
	if err != nil {
		log.Errorf("Could not register metric: %v", err)
	}
	err = prometheus.Register(prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Name: "beaconchain_db_pending_pages_in_freelist",
		Help: "total number of pending pages on the freelist",
	}, func() float64 {
		return float64(db.Stats().PendingPageN)
	}))
	if err != nil {
		log.Errorf("Could not register metric: %v", err)
	}

	err = prometheus.Register(prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Name: "beaconchain_db_size_of_FreeList",
		Help: "total bytes used by the freelist",
	}, func() float64 {
		return float64(db.Stats().FreelistInuse)
	}))
	if err != nil {
		log.Errorf("Could not register metric: %v", err)
	}

	err = prometheus.Register(prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Name: "beaconchain_db_num_of_page_allocations",
		Help: "number of page allocations",
	}, func() float64 {
		return float64(db.Stats().TxStats.PageCount)
	}))
	if err != nil {
		log.Errorf("Could not register metric: %v", err)
	}

	err = prometheus.Register(prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Name: "beaconchain_db_bytes_allocated_in_pages",
		Help: "total bytes allocated in pages",
	}, func() float64 {
		return float64(db.Stats().TxStats.PageAlloc)
	}))
	if err != nil {
		log.Errorf("Could not register metric: %v", err)
	}

	err = prometheus.Register(prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Name: "beaconchain_db_num_of_db_writes",
		Help: "number of writes performed in the db",
	}, func() float64 {
		return float64(db.Stats().TxStats.Write)
	}))
	if err != nil {
		log.Errorf("Could not register metric: %v", err)
	}

	err = prometheus.Register(prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Name: "beaconchain_db_time_spent_writing_to_db",
		Help: "total time spent writing to disk in the db",
	}, func() float64 {
		return float64(db.Stats().TxStats.WriteTime)
	}))

	if err != nil {
		log.Errorf("Could not register metric: %v", err)
	}
}
