package sync

import (
	"context"
	"encoding/hex"
	"sync"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/runutil"
	"github.com/prysmaticlabs/prysm/shared/slotutil"
	"github.com/prysmaticlabs/prysm/shared/traceutil"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
	"golang.org/x/exp/rand"
)

// This defines how often a node cleans up and processes pending attestations in the queue.
var processPendingAttsPeriod = slotutil.DivideSlotBy(2 /* twice per slot */)

// This processes pending attestation queues on every `processPendingAttsPeriod`.
func (xx *Service) processPendingAttsQueue() {
	ctx := context.Background()
	mutex := new(sync.Mutex)
	runutil.RunEvery(xx.ctx, processPendingAttsPeriod, func() {
		mutex.Lock()
		if err := xx.processPendingAtts(ctx); err != nil {
			log.WithError(err).Errorf("Could not process pending attestation: %v", err)
		}
		mutex.Unlock()
	})
}

// This defines how pending attestations are processed. It contains features:
// 1. Clean up invalid pending attestations from the queue.
// 2. Check if pending attestations can be processed when the block has arrived.
// 3. Request block from a random peer if unable to proceed step 2.
func (xx *Service) processPendingAtts(ctx context.Context) error {
	ctx, span := trace.StartSpan(ctx, "processPendingAtts")
	defer span.End()

	pids := xx.p2p.Peers().Connected()

	// Before a node processes pending attestations queue, it verifies
	// the attestations in the queue are still valid. Attestations will
	// be deleted from the queue if invalid (ie. getting staled from falling too many slots behind).
	xx.validatePendingAtts(ctx, xx.chain.CurrentSlot())

	roots := make([][32]byte, 0, len(xx.blkRootToPendingAtts))
	xx.pendingAttsLock.RLock()
	for br := range xx.blkRootToPendingAtts {
		roots = append(roots, br)
	}
	xx.pendingAttsLock.RUnlock()

	for _, bRoot := range roots {
		xx.pendingAttsLock.RLock()
		attestations := xx.blkRootToPendingAtts[bRoot]
		xx.pendingAttsLock.RUnlock()
		// Has the pending attestation's missing block arrived and the node processed block yet?
		hasStateSummary := featureconfig.Get().NewStateMgmt && xx.db.HasStateSummary(ctx, bRoot) || xx.stateSummaryCache.Has(bRoot)
		if xx.db.HasBlock(ctx, bRoot) && (xx.db.HasState(ctx, bRoot) || hasStateSummary) {
			numberOfBlocksRecoveredFromAtt.Inc()
			for _, signedAtt := range attestations {
				att := signedAtt.Message
				// The pending attestations can arrive in both aggregated and unaggregated forms,
				// each from has distinct validation steps.
				if helpers.IsAggregated(att.Aggregate) {
					// Save the pending aggregated attestation to the pool if it passes the aggregated
					// validation steps.
					aggValid := xx.validateAggregatedAtt(ctx, signedAtt) == pubsub.ValidationAccept
					if xx.validateBlockInAttestation(ctx, signedAtt) && aggValid {
						if err := xx.attPool.SaveAggregatedAttestation(att.Aggregate); err != nil {
							return err
						}
						numberOfAttsRecovered.Inc()

						// Broadcasting the signed attestation again once a node is able to process it.
						if err := xx.p2p.Broadcast(ctx, signedAtt); err != nil {
							log.WithError(err).Error("Failed to broadcast")
						}
					}
				} else {
					// Save the pending unaggregated attestation to the pool if the BLS signature is
					// valid.
					if _, err := bls.SignatureFromBytes(att.Aggregate.Signature); err != nil {
						continue
					}
					if err := xx.attPool.SaveUnaggregatedAttestation(att.Aggregate); err != nil {
						return err
					}
					numberOfAttsRecovered.Inc()

					// Broadcasting the signed attestation again once a node is able to process it.
					if err := xx.p2p.Broadcast(ctx, signedAtt); err != nil {
						log.WithError(err).Error("Failed to broadcast")
					}
				}
			}
			log.WithFields(logrus.Fields{
				"blockRoot":        hex.EncodeToString(bytesutil.Trunc(bRoot[:])),
				"pendingAttsCount": len(attestations),
			}).Info("Verified and saved pending attestations to pool")

			// Delete the missing block root key from pending attestation queue so a node will not request for the block again.
			xx.pendingAttsLock.Lock()
			delete(xx.blkRootToPendingAtts, bRoot)
			xx.pendingAttsLock.Unlock()
		} else {
			// Pending attestation's missing block has not arrived yet.
			log.WithFields(logrus.Fields{
				"currentSlot": xx.chain.CurrentSlot(),
				"attSlot":     attestations[0].Message.Aggregate.Data.Slot,
				"attCount":    len(attestations),
				"blockRoot":   hex.EncodeToString(bytesutil.Trunc(bRoot[:])),
			}).Debug("Requesting block for pending attestation")

			// Start with a random peer to query, but choose the first peer in our unsorted list that claims to
			// have a head slot newer or equal to the pending attestation's target boundary slot.
			if len(pids) == 0 {
				return nil
			}
			pid := pids[rand.Int()%len(pids)]
			targetSlot := helpers.SlotToEpoch(attestations[0].Message.Aggregate.Data.Target.Epoch)
			for _, p := range pids {
				cs, err := xx.p2p.Peers().ChainState(p)
				if err != nil {
					return errors.Wrap(err, "could not get chain state for peer")
				}
				if cs != nil && cs.HeadSlot >= targetSlot {
					pid = p
					break
				}
			}

			req := [][]byte{bRoot[:]}
			if err := xx.sendRecentBeaconBlocksRequest(ctx, req, pid); err != nil {
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
func (xx *Service) savePendingAtt(att *ethpb.SignedAggregateAttestationAndProof) {
	root := bytesutil.ToBytes32(att.Message.Aggregate.Data.BeaconBlockRoot)

	xx.pendingAttsLock.Lock()
	defer xx.pendingAttsLock.Unlock()
	_, ok := xx.blkRootToPendingAtts[root]
	if !ok {
		xx.blkRootToPendingAtts[root] = []*ethpb.SignedAggregateAttestationAndProof{att}
		return
	}

	xx.blkRootToPendingAtts[root] = append(xx.blkRootToPendingAtts[root], att)
}

// This validates the pending attestations in the queue are still valid.
// If not valid, a node will remove it in the queue in place. The validity
// check specifies the pending attestation could not fall one epoch behind
// of the current slot.
func (xx *Service) validatePendingAtts(ctx context.Context, slot uint64) {
	ctx, span := trace.StartSpan(ctx, "validatePendingAtts")
	defer span.End()

	xx.pendingAttsLock.Lock()
	defer xx.pendingAttsLock.Unlock()

	for bRoot, atts := range xx.blkRootToPendingAtts {
		for i := len(atts) - 1; i >= 0; i-- {
			if slot >= atts[i].Message.Aggregate.Data.Slot+params.BeaconConfig().SlotsPerEpoch {
				// Remove the pending attestation from the list in place.
				atts = append(atts[:i], atts[i+1:]...)
				numberOfAttsNotRecovered.Inc()
			}
		}
		xx.blkRootToPendingAtts[bRoot] = atts

		// If the pending attestations list of a given block root is empty,
		// a node will remove the key from the map to avoid dangling keys.
		if len(xx.blkRootToPendingAtts[bRoot]) == 0 {
			delete(xx.blkRootToPendingAtts, bRoot)
			numberOfBlocksNotRecoveredFromAtt.Inc()
		}
	}
}
