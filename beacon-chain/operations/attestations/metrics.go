package attestations

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	attCount = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "attestations_in_pool_total",
			Help: "The number of attestations in the pool.",
		},
	)
	expiredAtts = promauto.NewCounter(prometheus.CounterOpts{
		Name: "expired_atts_total",
		Help: "The number of expired and deleted attestations in the pool.",
	})
	batchForkChoiceAttsT1 = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "aggregate_attestations_t1",
			Help:    "Captures times of attestation aggregation in milliseconds during the first interval per slot",
			Buckets: []float64{10, 20, 50, 100, 200, 300, 500, 1000},
		},
	)
	batchForkChoiceAttsT2 = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "aggregate_attestations_t2",
			Help:    "Captures times of attestation aggregation in milliseconds during the second interval per slot",
			Buckets: []float64{10, 40, 100, 200, 600},
		},
	)
)

func (s *Service) updateMetrics(numExpired uint64) {
	attCount.Set(float64(s.cfg.Cache.Count()))
	expiredAtts.Add(float64(numExpired))
}
