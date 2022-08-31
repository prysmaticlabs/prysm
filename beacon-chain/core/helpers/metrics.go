package helpers

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	attReceivedTooEarlyCount = promauto.NewCounter(prometheus.CounterOpts{
		Name: "gossip_attestation_too_early_ignored_total",
		Help: "Increased when a gossip attestation fails decoding",
	})
	attReceivedTooLateCount = promauto.NewCounter(prometheus.CounterOpts{
		Name: "gossip_attestation_too_late_ignored_total",
		Help: "Increased when a gossip attestation fails decoding",
	})
)
