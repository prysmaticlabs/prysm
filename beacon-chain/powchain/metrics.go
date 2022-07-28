package powchain

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	totalTerminalDifficulty = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "total_terminal_difficulty",
		Help: "The total terminal difficulty of the execution chain before merge",
	})
	newPayloadLatency = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "new_payload_v1_latency_milliseconds",
			Help:    "Captures RPC latency for newPayloadV1 in milliseconds",
			Buckets: []float64{25, 50, 100, 200, 500, 1000, 2000, 4000},
		},
	)
	getPayloadLatency = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "get_payload_v1_latency_milliseconds",
			Help:    "Captures RPC latency for getPayloadV1 in milliseconds",
			Buckets: []float64{25, 50, 100, 200, 500, 1000, 2000, 4000},
		},
	)
	forkchoiceUpdatedLatency = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "forkchoice_updated_v1_latency_milliseconds",
			Help:    "Captures RPC latency for forkchoiceUpdatedV1 in milliseconds",
			Buckets: []float64{25, 50, 100, 200, 500, 1000, 2000, 4000},
		},
	)
	reconstructedExecutionPayloadCount = promauto.NewCounter(prometheus.CounterOpts{
		Name: "reconstructed_execution_payload_count",
		Help: "Count the number of execution payloads that are reconstructed using JSON-RPC from payload headers",
	})
)
