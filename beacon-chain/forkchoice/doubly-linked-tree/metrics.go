package doublylinkedtree

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	headSlotNumber = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "doublylinkedtree_head_slot",
			Help: "The slot number of the current head.",
		},
	)
	nodeCount = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "doublylinkedtree_node_count",
			Help: "The number of nodes in the DAG array based store structure.",
		},
	)
	headChangesCount = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "doublylinkedtree_head_changed_count",
			Help: "The number of times head changes.",
		},
	)
	calledHeadCount = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "doublylinkedtree_head_requested_count",
			Help: "The number of times someone called head.",
		},
	)
	processedBlockCount = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "doublylinkedtree_block_processed_count",
			Help: "The number of times a block is processed for fork choice.",
		},
	)
	processedAttestationCount = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "doublylinkedtree_attestation_processed_count",
			Help: "The number of times an attestation is processed for fork choice.",
		},
	)
	prunedCount = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "doublylinkedtree_pruned_count",
			Help: "The number of times pruning happened.",
		},
	)
	validatedCount = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "doublylinkedtree_validated_count",
			Help: "The number of blocks that have been fully validated.",
		},
	)
)
