package sync

import (
	"context"
	"time"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/prysmaticlabs/prysm/v5/config/params"
	"github.com/prysmaticlabs/prysm/v5/monitoring/tracing"
	"github.com/prysmaticlabs/prysm/v5/proto/eth/v2"
	"github.com/prysmaticlabs/prysm/v5/time/slots"
	"go.opencensus.io/trace"
)

func (s *Service) validateLightClientFinalityUpdate(ctx context.Context, pid peer.ID, msg *pubsub.Message) (pubsub.ValidationResult, error) {
	// Validation runs on publish (not just subscriptions), so we should approve any message from
	// ourselves.
	if pid == s.cfg.p2p.PeerID() {
		return pubsub.ValidationAccept, nil
	}

	// TODO keep?
	// The head state will be too far away to validate any execution change.
	if s.cfg.initialSync.Syncing() {
		return pubsub.ValidationIgnore, nil
	}

	ctx, span := trace.StartSpan(ctx, "sync.validateLightClientFinalityUpdate")
	defer span.End()

	m, err := s.decodePubsubMessage(msg)
	if err != nil {
		tracing.AnnotateError(span, err)
		return pubsub.ValidationReject, err
	}

	update, ok := m.(*eth.LightClientFinalityUpdate)
	if !ok {
		return pubsub.ValidationReject, errWrongMessage
	}

	s.lightClientFinalityUpdateLock.Lock()

	best := s.bestLightClientFinalityUpdate
	if best != nil {
		// [IGNORE] The finalized_header.beacon.slot is greater than that of all previously forwarded finality_updates,
		// or it matches the highest previously forwarded slot and also has a sync_aggregate indicating supermajority (> 2/3)
		// sync committee participation while the previously forwarded finality_update for that slot did not indicate supermajority
		if update.FinalizedHeader.Slot < best.slot {
			s.lightClientFinalityUpdateLock.Unlock()
			return pubsub.ValidationIgnore, nil
		}
		if update.FinalizedHeader.Slot == best.slot && (best.hasSupermajority || !update.HasSupermajority()) {
			s.lightClientFinalityUpdateLock.Unlock()
			return pubsub.ValidationIgnore, nil
		}
	}
	// [IGNORE] The finality_update is received after the block at signature_slot was given enough time
	// to propagate through the network -- i.e. validate that one-third of finality_update.signature_slot
	// has transpired (SECONDS_PER_SLOT / INTERVALS_PER_SLOT seconds after the start of the slot,
	// with a MAXIMUM_GOSSIP_CLOCK_DISPARITY allowance)
	earliestValidTime := slots.StartTime(uint64(s.cfg.clock.GenesisTime().Unix()), update.FinalizedHeader.Slot).
		Add(time.Second * time.Duration(params.BeaconConfig().SecondsPerSlot/params.BeaconConfig().IntervalsPerSlot)).
		Add(-params.BeaconConfig().MaximumGossipClockDisparityDuration())
	if s.cfg.clock.Now().Before(earliestValidTime) {
		s.lightClientFinalityUpdateLock.Unlock()
		return pubsub.ValidationIgnore, nil
	}

	// currentTimeWithDisparity := receivedTime.Add(params.BeaconConfig().MaximumGossipClockDisparityDuration())
	//	return currentTimeWithDisparity.Unix() < slotTime.Unix()
	s.lightClientFinalityUpdateLock.Unlock()
}

func (s *Service) validateLightClientOptimisticUpdate(ctx context.Context, pid peer.ID, msg *pubsub.Message) (pubsub.ValidationResult, error) {

}
