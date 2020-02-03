package sync

import (
	"context"
	"encoding/hex"
	"time"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/runutil"
	"github.com/prysmaticlabs/prysm/shared/traceutil"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
	"golang.org/x/exp/rand"
)

// This defines how often a node cleans up and processes pending attestations in the queue.
var processPendingAttsPeriod = time.Duration(params.BeaconConfig().SecondsPerSlot/2) * time.Second

// This processes pending attestation queues on every `processPendingAttsPeriod`.
func (s *Service) processPendingAttsQueue() {
	ctx := context.Background()
	runutil.RunEvery(s.ctx, processPendingAttsPeriod, func() {
		s.processPendingAtts(ctx)
	})
}

// This defines how pending attestations are processed. It contains features:
// 1. Clean up invalid pending attestations from the queue.
// 2. Check if pending attestations can be processed when the block has arrived.
// 3. Request block from a random peer if unable to proceed step 2.
func (s *Service) processPendingAtts(ctx context.Context) error {
	ctx, span := trace.StartSpan(ctx, "processPendingAtts")
	defer span.End()

	pids := s.p2p.Peers().Connected()

	// Before a node processes pending attestations queue, it verifies
	// the attestations in the queue are still valid. Attestations will
	// be deleted from the queue if invalid (ie. getting staled from falling too many slots behind).
	s.validatePendingAtts(ctx, s.chain.CurrentSlot())

	for bRoot, attestations := range s.blkRootToPendingAtts {
		// Has the pending attestation's missing block arrived yet?
		if s.db.HasBlock(ctx, bRoot) {
			numberOfBlocksRecoveredFromAtt.Inc()
			for _, att := range attestations {
				// The pending attestations can arrive in both aggregated and unaggregated forms,
				// each from has distinct validation steps.
				if helpers.IsAggregated(att.Aggregate) {
					// Save the pending aggregated attestation to the pool if it passes the aggregated
					// validation steps.
					if s.validateAggregatedAtt(ctx, att) {
						if err := s.attPool.SaveAggregatedAttestation(att.Aggregate); err != nil {
							return err
						}
						numberOfAttsRecovered.Inc()

						// Broadcasting the attestation again once a node is able to process it.
						if err := s.p2p.Broadcast(ctx, att); err != nil {
							log.WithError(err).Error("Failed to broadcast")
						}
					}
				} else {
					// Save the pending unaggregated attestation to the pool if the BLS signature is
					// valid.
					if _, err := bls.SignatureFromBytes(att.Aggregate.Signature); err != nil {
						continue
					}
					if err := s.attPool.SaveUnaggregatedAttestation(att.Aggregate); err != nil {
						return err
					}
					numberOfAttsRecovered.Inc()

					// Broadcasting the attestation again once a node is able to process it.
					if err := s.p2p.Broadcast(ctx, att); err != nil {
						log.WithError(err).Error("Failed to broadcast")
					}
				}
			}
			log.WithFields(logrus.Fields{
				"blockRoot":        hex.EncodeToString(bytesutil.Trunc(bRoot[:])),
				"pendingAttsCount": len(attestations),
			}).Info("Verified and saved pending attestations to pool")

			// Delete the missing block root key from pending attestation queue so a node will not request for the block again.
			delete(s.blkRootToPendingAtts, bRoot)
		} else {
			// Pending attestation's missing block has not arrived yet.
			log.WithFields(logrus.Fields{
				"currentSlot": s.chain.CurrentSlot(),
				"attSlot":     attestations[0].Aggregate.Data.Slot,
				"attCount":    len(attestations),
				"blockRoot":   hex.EncodeToString(bytesutil.Trunc(bRoot[:])),
			}).Info("Requesting block for pending attestation")

			// Start with a random peer to query, but choose the first peer in our unsorted list that claims to
			// have a head slot newer or equal to the pending attestation's target boundary slot.
			pid := pids[rand.Int()%len(pids)]
			targetSlot := helpers.SlotToEpoch(attestations[0].Aggregate.Data.Target.Epoch)
			for _, p := range pids {
				if cs, _ := s.p2p.Peers().ChainState(p); cs != nil && cs.HeadSlot >= targetSlot {
					pid = p
					break
				}
			}

			req := [][32]byte{bRoot}
			if err := s.sendRecentBeaconBlocksRequest(ctx, req, pid); err != nil {
				traceutil.AnnotateError(span, err)
				log.Errorf("Could not send recent block request: %v", err)
			}
		}
	}
	return nil
}

// This defines how pending attestations is saved in the map. The key is the
// root of the missing block. The value is the list of pending attestations
// that voted for that block root.
func (s *Service) savePendingAtt(att *ethpb.AggregateAttestationAndProof) {
	s.pendingAttsLock.Lock()
	defer s.pendingAttsLock.Unlock()

	root := bytesutil.ToBytes32(att.Aggregate.Data.BeaconBlockRoot)

	_, ok := s.blkRootToPendingAtts[root]
	if !ok {
		s.blkRootToPendingAtts[root] = []*ethpb.AggregateAttestationAndProof{att}
		return
	}

	s.blkRootToPendingAtts[root] = append(s.blkRootToPendingAtts[root], att)
}

// This validates the pending attestations in the queue are still valid.
// If not valid, a node will remove it in the queue in place. The validity
// check specifies the pending attestation could not fall one epoch behind
// of the current slot.
func (s *Service) validatePendingAtts(ctx context.Context, slot uint64) {
	s.pendingAttsLock.Lock()
	defer s.pendingAttsLock.Unlock()

	ctx, span := trace.StartSpan(ctx, "validatePendingAtts")
	defer span.End()

	for bRoot, atts := range s.blkRootToPendingAtts {
		for i := len(atts) - 1; i >= 0; i-- {
			if slot >= atts[i].Aggregate.Data.Slot+params.BeaconConfig().SlotsPerEpoch {
				// Remove the pending attestation from the list in place.
				atts = append(atts[:i], atts[i+1:]...)
				numberOfAttsNotRecovered.Inc()
			}
		}
		s.blkRootToPendingAtts[bRoot] = atts

		// If the pending attestations list of a given block root is empty,
		// a node will remove the key from the map to avoid dangling keys.
		if len(s.blkRootToPendingAtts[bRoot]) == 0 {
			delete(s.blkRootToPendingAtts, bRoot)
			numberOfBlocksNotRecoveredFromAtt.Inc()
		}
	}
}
