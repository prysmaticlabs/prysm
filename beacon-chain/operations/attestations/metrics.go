package attestations

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	aggregatedAttsCount = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "aggregated_attestations_in_pool_count",
			Help: "The number of aggregated attestations in the pool.",
		},
	)
	unaggregatedAttsCount = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "unaggregated_attestations_in_pool_count",
			Help: "The number of unaggregated attestations in the pool.",
		},
	)
)

func (s *Service) updateMetrics() {
	aggregatedAttsCount.Set(float64(s.pool.AggregatedAttestationCount()))
	unaggregatedAttsCount.Set(float64(s.pool.UnaggregatedAttestationCount()))
}
