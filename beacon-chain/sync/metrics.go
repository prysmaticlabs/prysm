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
			Name: "beacon_blocks_recovered_from_attestation_total",
			Help: "Count the number of times a missing block recovered from attestation vote.",
		},
	)
	numberOfBlocksNotRecoveredFromAtt = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "beacon_blocks_not_recovered_from_attestation_total",
			Help: "Count the number of times a missing block not recovered and pruned from attestation vote.",
		},
	)
	numberOfAttsRecovered = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "beacon_attestations_recovered_total",
			Help: "Count the number of times attestation recovered because of missing block",
		},
	)
	numberOfAttsNotRecovered = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "beacon_attestations_not_recovered_total",
			Help: "Count the number of times attestation not recovered and pruned because of missing block",
		},
	)
)
