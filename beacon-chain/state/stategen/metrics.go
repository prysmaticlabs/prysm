package stategen

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	replayBlockCount = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "replay_blocks_count",
			Help:    "The number of blocks to replay to generate a state",
			Buckets: []float64{64, 256, 1024, 2048, 4096},
		},
	)
)
