package initialsync

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// Metrics
	sentBatchedBlockReq = promauto.NewCounter(prometheus.CounterOpts{
		Name: "initsync_sent_batched_block_req",
		Help: "The number of sent batched block req",
	})
	batchedBlockReq = promauto.NewCounter(prometheus.CounterOpts{
		Name: "initsync_batched_block_req",
		Help: "The number of received batch blocks responses",
	})
	recBlock = promauto.NewCounter(prometheus.CounterOpts{
		Name: "initsync_received_blocks",
		Help: "The number of received blocks",
	})
	stateReq = promauto.NewCounter(prometheus.CounterOpts{
		Name: "initsync_state_req",
		Help: "The number of sent state requests",
	})
	recState = promauto.NewCounter(prometheus.CounterOpts{
		Name: "initsync_received_state",
		Help: "The number of received state",
	})
)
