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
	hotSummarySaved = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "hot_summary_count",
			Help: "The number of state summary saved in the hot section of the DB.",
		},
	)
	coldSummarySaved = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "cold_summary_count",
			Help: "The number of state summary saved in the cold section of the DB.",
		},
	)
	archivePointSaved = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "archive_point_count",
			Help: "The number of archive point saved in the DB.",
		},
	)
)
