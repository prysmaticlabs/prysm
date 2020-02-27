package stategen

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	hotStateSaved = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "hot_state_count",
			Help: "The number of state saved in the hot section of the DB.",
		},
	)
	stateSummarySaved = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "state_summary_count",
			Help: "The number of state summary saved in the DB.",
		},
	)
	archivePointSaved = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "archive_point_count",
			Help: "The number of archive point saved in the DB.",
		},
	)
)
