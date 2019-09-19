package sync

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// TODO(3147): Add metrics for RPC & subscription success/error.

var (
	messageReceivedCounter = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "p2p_message_recieved_total",
			Help: "Count of messages received.",
		},
		[]string{"topic"},
	)
	messageReceivedBeforeChainStartCounter = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "p2p_message_recieved_before_chain_start",
			Help: "Count of messages received before chain started.",
		},
		[]string{"topic"},
	)
)
