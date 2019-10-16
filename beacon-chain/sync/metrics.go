package sync

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	messageReceivedCounter = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "p2p_message_received_total",
			Help: "Count of messages received.",
		},
		[]string{"topic"},
	)
	messageReceivedBeforeChainStartCounter = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "p2p_message_received_before_chain_start",
			Help: "Count of messages received before chain started.",
		},
		[]string{"topic"},
	)
	messageFailedValidationCounter = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "p2p_message_failed_validation_total",
			Help: "Count of messages that failed validation.",
		},
		[]string{"topic"},
	)
	messageFailedProcessingCounter = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "p2p_message_failed_processing_total",
			Help: "Count of messages that passed validation but failed processing.",
		},
		[]string{"topic"},
	)
)
