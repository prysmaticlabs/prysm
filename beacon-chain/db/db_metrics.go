package db

import (
	"github.com/boltdb/bolt"
	"github.com/prometheus/client_golang/prometheus"
)

// Registers the boltDB data metrics.
func registerDBMetrics(db *bolt.DB) {
	metricFuncs := []prometheus.Collector{
		prometheus.NewGaugeFunc(prometheus.GaugeOpts{
			Name: "beaconchain_db_page_size",
			Help: "Current page size of the whole db",
		}, func() float64 {
			return float64(db.Info().PageSize)
		}),
		prometheus.NewCounterFunc(prometheus.CounterOpts{
			Name: "beaconchain_db_number_of_tx",
			Help: "The total number of started bolt read transactions",
		}, func() float64 {
			return float64(db.Stats().TxN)
		}),
		prometheus.NewGaugeFunc(prometheus.GaugeOpts{
			Name: "beaconchain_db_number_of_open_tx",
			Help: "The number of open bolt read transactions",
		}, func() float64 {
			return float64(db.Stats().OpenTxN)
		}),
		prometheus.NewGaugeFunc(prometheus.GaugeOpts{
			Name: "beaconchain_db_allocations_in_free_pages",
			Help: "total bytes allocated in free pages",
		}, func() float64 {
			return float64(db.Stats().FreeAlloc)
		}),
		prometheus.NewGaugeFunc(prometheus.GaugeOpts{
			Name: "beaconchain_db_number_of_free_pages",
			Help: "total number of free pages on the freelist",
		}, func() float64 {
			return float64(db.Stats().FreePageN)
		}),
		prometheus.NewGaugeFunc(prometheus.GaugeOpts{
			Name: "beaconchain_db_pending_pages_in_freelist",
			Help: "total number of pending pages on the freelist",
		}, func() float64 {
			return float64(db.Stats().PendingPageN)
		}),
		prometheus.NewGaugeFunc(prometheus.GaugeOpts{
			Name: "beaconchain_db_size_of_FreeList",
			Help: "total bytes used by the freelist",
		}, func() float64 {
			return float64(db.Stats().FreelistInuse)
		}),
		prometheus.NewGaugeFunc(prometheus.GaugeOpts{
			Name: "beaconchain_db_num_of_page_allocations",
			Help: "number of page allocations",
		}, func() float64 {
			return float64(db.Stats().TxStats.PageCount)
		}),
		prometheus.NewGaugeFunc(prometheus.GaugeOpts{
			Name: "beaconchain_db_bytes_allocated_in_pages",
			Help: "total bytes allocated in pages",
		}, func() float64 {
			return float64(db.Stats().TxStats.PageAlloc)
		}),
		prometheus.NewCounterFunc(prometheus.CounterOpts{
			Name: "beaconchain_db_num_of_db_writes",
			Help: "number of writes performed in the db",
		}, func() float64 {
			return float64(db.Stats().TxStats.Write)
		}),
		prometheus.NewCounterFunc(prometheus.CounterOpts{
			Name: "beaconchain_db_time_spent_writing_to_db",
			Help: "total time spent writing to disk in the db",
		}, func() float64 {
			return db.Stats().TxStats.WriteTime.Seconds()
		}),
		prometheus.NewGaugeFunc(prometheus.GaugeOpts{
			Name: "beaconchain_db_attestation_bucket_number_of_keys",
			Help: "number of keys/value pairs in the attestation bucket",
		}, func() float64 {
			var keys float64
			if err := db.View(func(tx *bolt.Tx) error {
				keys = float64(tx.Bucket(attestationBucket).Stats().KeyN)
				return nil
			}); err != nil {
				return 0
			}
			return keys
		}),
		prometheus.NewGaugeFunc(prometheus.GaugeOpts{
			Name: "beaconchain_db_attestation_bucket_depth",
			Help: "number of levels in B+tree in the attestation bucket",
		}, func() float64 {
			var depth float64
			if err := db.View(func(tx *bolt.Tx) error {
				depth = float64(tx.Bucket(attestationBucket).Stats().Depth)
				return nil
			}); err != nil {
				return 0
			}
			return depth
		}),
		prometheus.NewGaugeFunc(prometheus.GaugeOpts{
			Name: "beaconchain_db_attestation_bucket_total_leaf_size",
			Help: "bytes actually used for leaf data in attestation bucket",
		}, func() float64 {
			var leafInUse float64
			if err := db.View(func(tx *bolt.Tx) error {
				leafInUse = float64(tx.Bucket(attestationBucket).Stats().LeafInuse)
				return nil
			}); err != nil {
				return 0
			}
			return leafInUse
		}),
		prometheus.NewGaugeFunc(prometheus.GaugeOpts{
			Name: "beaconchain_db_attestation_target_bucket_number_of_keys",
			Help: "number of keys/value pairs in the attestation target bucket",
		}, func() float64 {
			var keys float64
			if err := db.View(func(tx *bolt.Tx) error {
				keys = float64(tx.Bucket(attestationTargetBucket).Stats().KeyN)
				return nil
			}); err != nil {
				return 0
			}
			return keys
		}),
		prometheus.NewGaugeFunc(prometheus.GaugeOpts{
			Name: "beaconchain_db_attestation_target_bucket_depth",
			Help: "number of levels in B+tree in the attestation target bucket",
		}, func() float64 {
			var depth float64
			if err := db.View(func(tx *bolt.Tx) error {
				depth = float64(tx.Bucket(attestationTargetBucket).Stats().Depth)
				return nil
			}); err != nil {
				return 0
			}
			return depth
		}),
		prometheus.NewGaugeFunc(prometheus.GaugeOpts{
			Name: "beaconchain_db_attestation_target_bucket_total_leaf_size",
			Help: "bytes actually used for leaf data in attestation target bucket",
		}, func() float64 {
			var leafInUse float64
			if err := db.View(func(tx *bolt.Tx) error {
				leafInUse = float64(tx.Bucket(attestationTargetBucket).Stats().LeafInuse)
				return nil
			}); err != nil {
				return 0
			}
			return leafInUse
		}),
		prometheus.NewGaugeFunc(prometheus.GaugeOpts{
			Name: "beaconchain_db_block_bucket_number_of_keys",
			Help: "number of keys/value pairs in the block bucket",
		}, func() float64 {
			var keys float64
			if err := db.View(func(tx *bolt.Tx) error {
				keys = float64(tx.Bucket(blockBucket).Stats().KeyN)
				return nil
			}); err != nil {
				return 0
			}
			return keys
		}),
		prometheus.NewGaugeFunc(prometheus.GaugeOpts{
			Name: "beaconchain_db_block_bucket_depth",
			Help: "number of levels in B+tree in the block bucket",
		}, func() float64 {
			var depth float64
			if err := db.View(func(tx *bolt.Tx) error {
				depth = float64(tx.Bucket(blockBucket).Stats().Depth)
				return nil
			}); err != nil {
				return 0
			}
			return depth
		}),
		prometheus.NewGaugeFunc(prometheus.GaugeOpts{
			Name: "beaconchain_db_block_bucket_total_leaf_size",
			Help: "bytes actually used for leaf data in block bucket",
		}, func() float64 {
			var leafInUse float64
			if err := db.View(func(tx *bolt.Tx) error {
				leafInUse = float64(tx.Bucket(blockBucket).Stats().LeafInuse)
				return nil
			}); err != nil {
				return 0
			}
			return leafInUse
		}),
		prometheus.NewGaugeFunc(prometheus.GaugeOpts{
			Name: "beaconchain_db_mainchain_bucket_number_of_keys",
			Help: "number of keys/value pairs in the mainchain bucket",
		}, func() float64 {
			var keys float64
			if err := db.View(func(tx *bolt.Tx) error {
				keys = float64(tx.Bucket(mainChainBucket).Stats().KeyN)
				return nil
			}); err != nil {
				return 0
			}
			return keys
		}),
		prometheus.NewGaugeFunc(prometheus.GaugeOpts{
			Name: "beaconchain_db_mainchain_bucket_depth",
			Help: "number of levels in B+tree in the mainchain bucket",
		}, func() float64 {
			var depth float64
			if err := db.View(func(tx *bolt.Tx) error {
				depth = float64(tx.Bucket(mainChainBucket).Stats().Depth)
				return nil
			}); err != nil {
				return 0
			}
			return depth
		}),
		prometheus.NewGaugeFunc(prometheus.GaugeOpts{
			Name: "beaconchain_db_mainchain_bucket_total_leaf_size",
			Help: "bytes actually used for leaf data in mainchain bucket",
		}, func() float64 {
			var leafInUse float64
			if err := db.View(func(tx *bolt.Tx) error {
				leafInUse = float64(tx.Bucket(mainChainBucket).Stats().LeafInuse)
				return nil
			}); err != nil {
				return 0
			}
			return leafInUse
		}),
		prometheus.NewGaugeFunc(prometheus.GaugeOpts{
			Name: "beaconchain_db_historical_state_bucket_number_of_keys",
			Help: "number of keys/value pairs in the historical state bucket",
		}, func() float64 {
			var keys float64
			if err := db.View(func(tx *bolt.Tx) error {
				keys = float64(tx.Bucket(histStateBucket).Stats().KeyN)
				return nil
			}); err != nil {
				return 0
			}
			return keys
		}),
		prometheus.NewGaugeFunc(prometheus.GaugeOpts{
			Name: "beaconchain_db_historical_state_bucket_depth",
			Help: "number of levels in B+tree in the historical state bucket",
		}, func() float64 {
			var depth float64
			if err := db.View(func(tx *bolt.Tx) error {
				depth = float64(tx.Bucket(histStateBucket).Stats().Depth)
				return nil
			}); err != nil {
				return 0
			}
			return depth
		}),
		prometheus.NewGaugeFunc(prometheus.GaugeOpts{
			Name: "beaconchain_db_historical_state_bucket_total_leaf_size",
			Help: "bytes actually used for leaf data in historical state bucket",
		}, func() float64 {
			var leafInUse float64
			if err := db.View(func(tx *bolt.Tx) error {
				leafInUse = float64(tx.Bucket(histStateBucket).Stats().LeafInuse)
				return nil
			}); err != nil {
				return 0
			}
			return leafInUse
		}),
		prometheus.NewGaugeFunc(prometheus.GaugeOpts{
			Name: "beaconchain_db_chainInfo_bucket_number_of_keys",
			Help: "number of keys/value pairs in the chainInfo bucket",
		}, func() float64 {
			var keys float64
			if err := db.View(func(tx *bolt.Tx) error {
				keys = float64(tx.Bucket(chainInfoBucket).Stats().KeyN)
				return nil
			}); err != nil {
				return 0
			}
			return keys
		}),
		prometheus.NewGaugeFunc(prometheus.GaugeOpts{
			Name: "beaconchain_db_chainInfo_bucket_depth",
			Help: "number of levels in B+tree in the chainInfo bucket",
		}, func() float64 {
			var depth float64
			if err := db.View(func(tx *bolt.Tx) error {
				depth = float64(tx.Bucket(chainInfoBucket).Stats().Depth)
				return nil
			}); err != nil {
				return 0
			}
			return depth
		}),
		prometheus.NewGaugeFunc(prometheus.GaugeOpts{
			Name: "beaconchain_db_chainInfo_bucket_total_leaf_size",
			Help: "bytes actually used for leaf data in chainInfo bucket",
		}, func() float64 {
			var leafInUse float64
			if err := db.View(func(tx *bolt.Tx) error {
				leafInUse = float64(tx.Bucket(chainInfoBucket).Stats().LeafInuse)
				return nil
			}); err != nil {
				return 0
			}
			return leafInUse
		}),
		prometheus.NewGaugeFunc(prometheus.GaugeOpts{
			Name: "beaconchain_db_block_operations_bucket_number_of_keys",
			Help: "number of keys/value pairs in the block operations bucket",
		}, func() float64 {
			var keys float64
			if err := db.View(func(tx *bolt.Tx) error {
				keys = float64(tx.Bucket(blockOperationsBucket).Stats().KeyN)
				return nil
			}); err != nil {
				return 0
			}
			return keys
		}),
		prometheus.NewGaugeFunc(prometheus.GaugeOpts{
			Name: "beaconchain_db_block_operations_bucket_depth",
			Help: "number of levels in B+tree in the block operations bucket",
		}, func() float64 {
			var depth float64
			if err := db.View(func(tx *bolt.Tx) error {
				depth = float64(tx.Bucket(blockOperationsBucket).Stats().Depth)
				return nil
			}); err != nil {
				return 0
			}
			return depth
		}),
		prometheus.NewGaugeFunc(prometheus.GaugeOpts{
			Name: "beaconchain_db_block_operations_bucket_total_leaf_size",
			Help: "bytes actually used for leaf data in block operations bucket",
		}, func() float64 {
			var leafInUse float64
			if err := db.View(func(tx *bolt.Tx) error {
				leafInUse = float64(tx.Bucket(blockOperationsBucket).Stats().LeafInuse)
				return nil
			}); err != nil {
				return 0
			}
			return leafInUse
		}),
		prometheus.NewGaugeFunc(prometheus.GaugeOpts{
			Name: "beaconchain_db_validator_bucket_number_of_keys",
			Help: "number of keys/value pairs in the validator bucket",
		}, func() float64 {
			var keys float64
			if err := db.View(func(tx *bolt.Tx) error {
				keys = float64(tx.Bucket(validatorBucket).Stats().KeyN)
				return nil
			}); err != nil {
				return 0
			}
			return keys
		}),
		prometheus.NewGaugeFunc(prometheus.GaugeOpts{
			Name: "beaconchain_db_validator_bucket_depth",
			Help: "number of levels in B+tree in the validator bucket",
		}, func() float64 {
			var depth float64
			if err := db.View(func(tx *bolt.Tx) error {
				depth = float64(tx.Bucket(validatorBucket).Stats().Depth)
				return nil
			}); err != nil {
				return 0
			}
			return depth
		}),
		prometheus.NewGaugeFunc(prometheus.GaugeOpts{
			Name: "beaconchain_db_validator_bucket_total_leaf_size",
			Help: "bytes actually used for leaf data in validator bucket",
		}, func() float64 {
			var leafInUse float64
			if err := db.View(func(tx *bolt.Tx) error {
				leafInUse = float64(tx.Bucket(validatorBucket).Stats().LeafInuse)
				return nil
			}); err != nil {
				return 0
			}
			return leafInUse
		}),
	}

	for _, f := range metricFuncs {
		err := prometheus.Register(f)
		if err != nil {
			log.Errorf("Could not register metric: %v", err)
		}
	}
}
