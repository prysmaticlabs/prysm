package rpc

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	TrackedValidatorsCount = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "tracked_validator_count",
		Help: "The total number of validators tracked by trackedValidatorsCache in the beacon node. This is updated at intervals via the push proposer settings API endpoint.",
	})
)
