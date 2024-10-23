package peerdas

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var dataColumnComputationTime = promauto.NewHistogram(
	prometheus.HistogramOpts{
		Name:    "beacon_data_column_sidecar_computation_seconds",
		Help:    "Captures the time taken to compute data column sidecars from blobs.",
		Buckets: []float64{0.1, 0.25, 0.5, 0.75, 1, 1.5, 2, 4, 8, 12, 16},
	},
)
