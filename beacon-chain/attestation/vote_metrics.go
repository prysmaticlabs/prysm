package attestation

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	totalAttestationSeen = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "total_seen_attestations",
		Help: "Total number of attestations seen by the validators",
	})

	attestationPoolLimit = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "attestation_pool_limit",
		Help: "The limit of the attestation pool",
	})
	attestationPoolSize = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "attestation_pool_size",
		Help: "The current size of the attestation pool",
	})
)
