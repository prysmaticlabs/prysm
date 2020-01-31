package sync

import (
	"context"
	"time"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/roughtime"
	"github.com/prysmaticlabs/prysm/shared/runutil"
	"go.opencensus.io/trace"
)

var processPendingAttsPeriod = time.Duration(params.BeaconConfig().SecondsPerSlot/3) * time.Second

// processes pending attestation queue on every processPendingBlocksPeriod.
func (r *Service) processPendingAttsQueue() {
	ctx := context.Background()
	runutil.RunEvery(r.ctx, processPendingAttsPeriod, func() {
		r.processPendingBlocks(ctx)
	})
}

// processes the block tree inside the queue
func (r *Service) processPendingAtts(ctx context.Context) error {
	ctx, span := trace.StartSpan(ctx, "processPendingAtts")
	defer span.End()

	pids := r.p2p.Peers().Connected()
	r.validatePendingAtts(r.currentSlot())

	for bRoot, attestations := range r.blkRootToPendingAtts {
		if r.db.HasBlock(ctx, bRoot) {

		} else {
			req := [][32]byte{bRoot}
		}
	}
}

func (r *Service) savePendingAtt(att *ethpb.Attestation) {
	r.pendingAttsLock.Lock()
	defer r.pendingAttsLock.Unlock()

	r32 := bytesutil.ToBytes32(att.Data.BeaconBlockRoot)

	_, ok := r.blkRootToPendingAtts[r32]
	if !ok {
		r.blkRootToPendingAtts[r32] = []*ethpb.Attestation{att}
	} else {
		r.blkRootToPendingAtts[r32] = append(r.blkRootToPendingAtts[r32], att)
	}
}

func (r *Service) validatePendingAtts(slot uint64) {
	r.pendingAttsLock.Lock()
	defer r.pendingAttsLock.Unlock()

	for bRoot, atts := range r.blkRootToPendingAtts {
		for i, att := range atts {
			if slot >= att.Data.Slot+params.BeaconConfig().SlotsPerEpoch {
				atts[len(atts)-1], atts[i] = atts[i], atts[len(atts)-1]
				atts = atts[:len(atts)-1]
			}
		}

		r.blkRootToPendingAtts[bRoot] = atts

		if len(r.blkRootToPendingAtts[bRoot]) == 0 {
			delete(r.blkRootToPendingAtts, bRoot)
		}
	}
}

// currentSlot returns the current slot based on time.
func (s *Service) currentSlot() uint64 {
	return uint64(roughtime.Now().Unix()-s.chain.GenesisTime().Unix()) / params.BeaconConfig().SecondsPerSlot
}
