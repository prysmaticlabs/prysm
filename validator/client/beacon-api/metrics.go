package beacon_api

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	httpActionLatency = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "validator",
			Name:      "http_action_latency_milliseconds",
			Help:      "Latency of HTTP actions performed against the beacon node",
			Buckets:   []float64{1, 10, 25, 100, 250, 1000, 2500, 10000},
		},
		[]string{"action"},
	)
	httpActionCount = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "validator",
			Name:      "http_action_count",
			Help:      "Number of all HTTP actions performed against the beacon node",
		},
		[]string{"action"},
	)
	failedHTTPActionCount = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "validator",
			Name:      "failed_http_action_count",
			Help:      "Number of failed HTTP actions performed against the beacon node",
		},
		[]string{"action"},
	)
)
