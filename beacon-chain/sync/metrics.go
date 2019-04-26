package sync

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (

	// Metrics
	batchedBlockReq = promauto.NewCounter(prometheus.CounterOpts{
		Name: "regsync_batched_block_req",
		Help: "The number of received batch block requests",
	})
	blockReqSlot = promauto.NewCounter(prometheus.CounterOpts{
		Name: "regsync_block_req_by_slot",
		Help: "The number of received block requests by slot",
	})
	blockReqHash = promauto.NewCounter(prometheus.CounterOpts{
		Name: "regsync_block_req_by_hash",
		Help: "The number of received block requests by hash",
	})
	recBlock = promauto.NewCounter(prometheus.CounterOpts{
		Name: "regsync_received_blocks",
		Help: "The number of received blocks",
	})
	forkedBlock = promauto.NewCounter(prometheus.CounterOpts{
		Name: "regsync_received_forked_blocks",
		Help: "The number of received forked blocks",
	})
	recBlockAnnounce = promauto.NewCounter(prometheus.CounterOpts{
		Name: "regsync_received_block_announce",
		Help: "The number of received block announcements",
	})
	sentBlockAnnounce = promauto.NewCounter(prometheus.CounterOpts{
		Name: "regsync_sent_block_announce",
		Help: "The number of sent block announcements",
	})
	sentBlockReq = promauto.NewCounter(prometheus.CounterOpts{
		Name: "regsync_sent_block_request",
		Help: "The number of sent block request",
	})
	sentBlocks = promauto.NewCounter(prometheus.CounterOpts{
		Name: "regsync_sent_blocks",
		Help: "The number of sent blocks",
	})
	sentBatchedBlocks = promauto.NewCounter(prometheus.CounterOpts{
		Name: "regsync_sent_batched_blocks",
		Help: "The number of sent batched blocks",
	})
	stateReq = promauto.NewCounter(prometheus.CounterOpts{
		Name: "regsync_state_req",
		Help: "The number of state requests",
	})
	sentState = promauto.NewCounter(prometheus.CounterOpts{
		Name: "regsync_sent_state",
		Help: "The number of sent state",
	})
	attestationReq = promauto.NewCounter(prometheus.CounterOpts{
		Name: "regsync_attestation_req",
		Help: "The number of received attestation requests",
	})
	recAttestation = promauto.NewCounter(prometheus.CounterOpts{
		Name: "regsync_received_attestation",
		Help: "The number of received attestations",
	})
	sentAttestation = promauto.NewCounter(prometheus.CounterOpts{
		Name: "regsync_sent_attestation",
		Help: "The number of sent attestations",
	})
	recExit = promauto.NewCounter(prometheus.CounterOpts{
		Name: "regsync_received_exits",
		Help: "The number of received exits",
	})
	sentExit = promauto.NewCounter(prometheus.CounterOpts{
		Name: "regsync_sent_exits",
		Help: "The number of sent exits",
	})
	chainHeadReq = promauto.NewCounter(prometheus.CounterOpts{
		Name: "regsync_chain_head_req",
		Help: "The number of sent attestation requests",
	})
	sentChainHead = promauto.NewCounter(prometheus.CounterOpts{
		Name: "regsync_chain_head_sent",
		Help: "The number of sent chain head responses",
	})
)
