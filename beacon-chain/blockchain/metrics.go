package blockchain

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
)

var (
	beaconSlot = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "beacon_slot",
		Help: "Latest slot of the beacon chain state",
	})
	beaconHeadSlot = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "beacon_head_slot",
		Help: "Slot of the head block of the beacon chain",
	})
	beaconHeadRoot = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "beacon_head_root",
		Help: "Root of the head block of the beacon chain, it returns the lowest 8 bytes interpreted as little endian",
	})
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
	headFinalizedEpoch = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "head_finalized_epoch",
		Help: "Last finalized epoch of the head state",
	})
	headFinalizedRoot = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "head_finalized_root",
		Help: "Last finalized root of the head state",
	})
)

func (s *Service) reportSlotMetrics(currentSlot uint64) {
	beaconSlot.Set(float64(currentSlot))
	beaconHeadSlot.Set(float64(s.HeadSlot()))
	beaconHeadRoot.Set(float64(bytesutil.ToLowInt64(s.HeadRoot())))
	if s.headState != nil {
		headFinalizedEpoch.Set(float64(s.headState.FinalizedCheckpoint.Epoch))
		headFinalizedRoot.Set(float64(bytesutil.ToLowInt64(s.headState.FinalizedCheckpoint.Root)))
	}
}
