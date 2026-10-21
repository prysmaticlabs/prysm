package blockchain

import (
	"context"

	"github.com/OffchainLabs/prysm/v7/config/params"
	"github.com/OffchainLabs/prysm/v7/runtime/version"
	"github.com/OffchainLabs/prysm/v7/time/slots"
)

// runInclusionListRefresh calls refreshInclusionListPayload at the inclusion list deadline of every slot.
func (s *Service) runInclusionListRefresh() {
	if err := s.waitForSync(); err != nil {
		log.WithError(err).Error("Failed to wait for initial sync")
		return
	}

	ticker := slots.NewSlotTickerWithOffsetFunc(s.genesisTime, slots.ComponentInterval(params.InclusionListDue))
	defer ticker.Done()

	for {
		select {
		case <-ticker.C():
			s.refreshInclusionListPayload(s.ctx)
		case <-s.ctx.Done():
			log.Debug("Context closed, exiting routine")
			return
		}
	}
}

// refreshInclusionListPayload restarts the payload build for the next slot proposer with the inclusion lists observed by the deadline.
// The first build starts when the head is set, before most inclusion lists for the slot arrive.
func (s *Service) refreshInclusionListPayload(ctx context.Context) {
	nextSlot := s.CurrentSlot() + 1
	if slots.ToEpoch(nextSlot) < params.BeaconConfig().HezeForkEpoch || !s.inRegularSync() {
		return
	}

	s.headLock.RLock()
	headRoot := s.headRoot()
	headState := s.headState(ctx)
	full := s.head.full
	s.headLock.RUnlock()
	if headState == nil || headState.IsNil() || headState.Version() < version.Gloas {
		return
	}

	attribute := s.getPayloadAttribute(ctx, headState, nextSlot, headRoot[:], full)
	// Return early if we are not proposing next slot.
	if attribute.IsEmpty() {
		return
	}
	s.updateForkchoiceWithAttributesGloas(ctx, headState, headRoot, full, nextSlot, attribute)
}
