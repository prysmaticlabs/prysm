package blockchain

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	competingAtts = promauto.NewCounter(prometheus.CounterOpts{
		Name: "competing_attestations",
		Help: "The # of attestations received and processed from a competing chain",
	})
	competingBlks = promauto.NewCounter(prometheus.CounterOpts{
		Name: "competing_blocks",
		Help: "The # of blocks received and processed from a competing chain",
	})
	processedBlkNoPubsub = promauto.NewCounter(prometheus.CounterOpts{
		Name: "processed_no_pubsub_block_counter",
		Help: "The # of processed block without pubsub, this usually means the blocks from sync",
	})
	processedBlkNoPubsubForkchoice = promauto.NewCounter(prometheus.CounterOpts{
		Name: "processed_no_pubsub_forkchoice_block_counter",
		Help: "The # of processed block without pubsub and forkchoice, this means indicate blocks from initial sync",
	})
	processedBlk = promauto.NewCounter(prometheus.CounterOpts{
		Name: "processed_block_counter",
		Help: "The # of total processed in block chain service, with fork choice and pubsub",
	})
	processedAttNoPubsub = promauto.NewCounter(prometheus.CounterOpts{
		Name: "processed_no_pubsub_attestation_counter",
		Help: "The # of processed attestation without pubsub, this usually means the attestations from sync",
	})
	processedAtt = promauto.NewCounter(prometheus.CounterOpts{
		Name: "processed_attestation_counter",
		Help: "The # of processed attestation with pubsub and fork choice, this ususally means attestations from rpc",
	})
)
