package sync

import (


	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// TODO(3147): Add metrics for RPC & subscription success/error.

var (
	messageSentCounter = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "p2p_message_sent_total",
			Help: "Count of messages sent.",
		},
		[]string{"topic"},
	)
)