package detection

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	doubleVotesDetected = promauto.NewCounter(prometheus.CounterOpts{
		Name: "double_votes_detected_total",
		Help: "The # of double vote slashable events detected",
	})
	surroundingVotesDetected = promauto.NewCounter(prometheus.CounterOpts{
		Name: "surrounding_votes_detected_total",
		Help: "The # of surrounding slashable events detected",
	})
	surroundedVotesDetected = promauto.NewCounter(prometheus.CounterOpts{
		Name: "surrounded_votes_detected_total",
		Help: "The # of surrounded slashable events detected",
	})
)
