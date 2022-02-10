package protoarray

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	headSlotNumber = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "proto_array_head_slot",
			Help: "The slot number of the current head.",
		},
	)
	nodeCount = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "proto_array_node_count",
			Help: "The number of nodes in the DAG array based store structure.",
		},
	)
	headChangesCount = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "proto_array_head_changed_count",
			Help: "The number of times head changes.",
		},
	)
	calledHeadCount = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "proto_array_head_requested_count",
			Help: "The number of times someone called head.",
		},
	)
	processedBlockCount = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "proto_array_block_processed_count",
			Help: "The number of times a block is processed for fork choice.",
		},
	)
	processedAttestationCount = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "proto_array_attestation_processed_count",
			Help: "The number of times an attestation is processed for fork choice.",
		},
	)
	prunedCount = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "proto_array_pruned_count",
			Help: "The number of times pruning happened.",
		},
	)
	lastSyncedTipSlot = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "proto_array_last_synced_tip_slot",
			Help: "The slot of the last fully validated block added to the proto array.",
		},
	)
	syncedTipsCount = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "proto_array_synced_tips_count",
			Help: "The number of elements in the syncedTips structure.",
		},
	)
)
