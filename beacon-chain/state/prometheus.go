package state

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	StateCount = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "beacon_state_count",
		Help: "Count the number of active beacon state objects.",
	})
)
