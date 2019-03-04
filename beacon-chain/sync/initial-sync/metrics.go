package initialsync

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// Metrics
	sentBatchedBlockReq = promauto.NewCounter(prometheus.CounterOpts{
		Name: "initSync_sent_batched_block_req",
		Help: "The number of sent batched block req",
	})
	batchedBlockReq = promauto.NewCounter(prometheus.CounterOpts{
		Name: "initSync_batched_block_req",
		Help: "The number of received batch blocks responses",
	})
	blockReqSlot = promauto.NewCounter(prometheus.CounterOpts{
		Name: "initSync_block_req_by_slot",
		Help: "The number of sent block requests by slot",
	})
	recBlock = promauto.NewCounter(prometheus.CounterOpts{
		Name: "initSync_received_blocks",
		Help: "The number of received blocks",
	})
	recBlockAnnounce = promauto.NewCounter(prometheus.CounterOpts{
		Name: "initSync_received_blocks",
		Help: "The number of received block announce",
	})
	stateReq = promauto.NewCounter(prometheus.CounterOpts{
		Name: "initSync_state_req",
		Help: "The number of sent state requests",
	})
	recState = promauto.NewCounter(prometheus.CounterOpts{
		Name: "initSync_received_state",
		Help: "The number of received state",
	})
)
