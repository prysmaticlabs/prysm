package internal

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	signRequestTimeElapse = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "client_request_duration_seconds",
			Help:    "Time (in seconds) spent doing client HTTP requests",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "status_code"},
	)
)
