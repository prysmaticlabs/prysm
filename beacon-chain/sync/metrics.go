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
	numberOfTimesResyncedCounter = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "number_of_times_resynced",
			Help: "Count the number of times a node resyncs.",
		},
	)
	numberOfBlocksRecoveredFromAtt = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "number_of_blocks_recovered_from_attestation",
			Help: "Count the number of times a missing block recovered from attestation vote.",
		},
	)
	numberOfBlocksNotRecoveredFromAtt = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "number_of_blocks_not_recovered_from_attestation",
			Help: "Count the number of times a missing block not recovered from attestation vote, before pruning",
		},
	)
	numberOfAttsRecovered = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "number_of_atts_recovered",
			Help: "Count the number of times attestation recovered because of missing block",
		},
	)
	numberOfAttsNotRecovered = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "number_of_atts_not_recovered",
			Help: "Count the number of times attestation not recovered because of missing block",
		},
	)
)
