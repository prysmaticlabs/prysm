package sync

import (
	"context"
	"time"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/roughtime"
	"github.com/prysmaticlabs/prysm/shared/runutil"
	"github.com/prysmaticlabs/prysm/shared/traceutil"
	"go.opencensus.io/trace"
)

var processPendingAttsPeriod = time.Duration(params.BeaconConfig().SecondsPerSlot/2) * time.Second

// processes pending attestation queue on every processPendingBlocksPeriod.
func (r *Service) processPendingAttsQueue() {
	ctx := context.Background()
	runutil.RunEvery(r.ctx, processPendingAttsPeriod, func() {
		r.processPendingAtts(ctx)
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
			for _, att := range attestations {
				if helpers.IsAggregated(att.Aggregate) {
					if r.validateAtt(ctx, att) {
						if err := r.attPool.SaveAggregatedAttestation(att.Aggregate); err != nil {
							return err
						}
					}
				} else {
					if _, err := bls.SignatureFromBytes(att.Aggregate.Signature); err != nil {
						continue
					}
					if err := r.attPool.SaveUnaggregatedAttestation(att.Aggregate); err != nil {
						return err
					}
				}
			}
			delete(r.blkRootToPendingAtts, bRoot)
		} else {
			req := [][32]byte{bRoot}
			if err := r.sendRecentBeaconBlocksRequest(ctx, req, pids[0]); err != nil {
				traceutil.AnnotateError(span, err)
				log.Errorf("Could not send recent block request: %v", err)
			}
		}
	}

	return nil
}

func (r *Service) savePendingAtt(att *ethpb.AggregateAttestationAndProof) {
	r.pendingAttsLock.Lock()
	defer r.pendingAttsLock.Unlock()

	r32 := bytesutil.ToBytes32(att.Aggregate.Data.BeaconBlockRoot)

	_, ok := r.blkRootToPendingAtts[r32]
	if !ok {
		r.blkRootToPendingAtts[r32] = []*ethpb.AggregateAttestationAndProof{att}
	} else {
		r.blkRootToPendingAtts[r32] = append(r.blkRootToPendingAtts[r32], att)
	}
}

func (r *Service) validatePendingAtts(slot uint64) {
	r.pendingAttsLock.Lock()
	defer r.pendingAttsLock.Unlock()

	for bRoot, atts := range r.blkRootToPendingAtts {
		for i, att := range atts {
			if slot >= att.Aggregate.Data.Slot+params.BeaconConfig().SlotsPerEpoch {
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
