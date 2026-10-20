package sync

import (
	"context"

	"github.com/OffchainLabs/prysm/v7/beacon-chain/blockchain"
	"github.com/OffchainLabs/prysm/v7/config/params"
	"github.com/OffchainLabs/prysm/v7/consensus-types/interfaces"
	"github.com/OffchainLabs/prysm/v7/time/slots"
)

// maxPendingHezeEnvelopeRoots bounds the map of reconstructed Heze envelopes
// awaiting their beacon block. Chunk admission is already gated by known bid
// commitments, so a small cap suffices.
const maxPendingHezeEnvelopeRoots = 8

// queuePendingPayloadEnvelopeHeze parks a Heze envelope already bound to its
// beacon block root until the block is processed. Signature or chunk
// authentication is not verified here; import revalidates fully and the slot
// bound plus the cap bound memory.
func (s *Service) queuePendingPayloadEnvelopeHeze(signed interfaces.ROSignedExecutionPayloadEnvelope, root [32]byte) {
	env, err := signed.Envelope()
	if err != nil {
		return
	}
	// A future slot would never fall below the finalized epoch, defeating
	// pruning and pinning memory.
	if env.Slot() > s.cfg.clock.CurrentSlot() {
		log.WithField("envelopeSlot", env.Slot()).Debug("Ignoring pending Heze payload envelope with future slot")
		return
	}

	s.pendingHezeEnvelopesLock.Lock()
	defer s.pendingHezeEnvelopesLock.Unlock()
	if _, ok := s.pendingHezeEnvelopes[root]; ok {
		return
	}
	if len(s.pendingHezeEnvelopes) >= maxPendingHezeEnvelopeRoots {
		log.Debug("Too many pending Heze payload envelopes, dropping envelope")
		return
	}
	s.pendingHezeEnvelopes[root] = signed
}

// processPendingPayloadEnvelopeHeze imports the queued Heze envelope for the
// given block root, if any. Unlike the Gloas pending path there is no
// re-broadcast: Heze envelopes travel as payload chunks, which already
// propagated on gossip.
func (s *Service) processPendingPayloadEnvelopeHeze(ctx context.Context, root [32]byte) {
	s.pendingHezeEnvelopesLock.Lock()
	signed, ok := s.pendingHezeEnvelopes[root]
	if ok {
		delete(s.pendingHezeEnvelopes, root)
	}
	s.pendingHezeEnvelopesLock.Unlock()
	if !ok {
		return
	}

	env, err := signed.Envelope()
	if err != nil {
		log.WithError(err).Debug("Could not get pending Heze execution payload envelope")
		return
	}
	if s.hasSeenPayloadEnvelope(root, env.BuilderIndex()) {
		return
	}
	if s.hasBadBlock(root) {
		s.setSeenPayloadEnvelope(root, env.BuilderIndex())
		return
	}

	receiveCtx, cancel := context.WithTimeout(ctx, params.BeaconConfig().SlotDuration())
	defer cancel()
	if err := s.cfg.chain.ReceiveExecutionPayloadEnvelope(receiveCtx, signed); err != nil {
		if blockchain.IsInvalidBlock(err) {
			s.setBadPayload(ctx, root)
		}
		log.WithError(err).Debug("Could not process pending Heze payload envelope")
		return
	}
	s.setSeenPayloadEnvelope(root, env.BuilderIndex())
}

// processPendingPayloadEnvelopesHeze iterates the pending Heze envelope map and
// processes any entry whose beacon block is now in forkchoice.
func (s *Service) processPendingPayloadEnvelopesHeze(ctx context.Context) {
	s.pendingHezeEnvelopesLock.Lock()
	roots := make([][32]byte, 0, len(s.pendingHezeEnvelopes))
	for root := range s.pendingHezeEnvelopes {
		roots = append(roots, root)
	}
	s.pendingHezeEnvelopesLock.Unlock()

	for _, root := range roots {
		if !s.cfg.chain.InForkchoice(root) {
			continue
		}
		s.processPendingPayloadEnvelopeHeze(ctx, root)
	}
}

// prunePendingPayloadEnvelopesHeze removes entries whose slot is below the
// finalized epoch start slot, mirroring prunePendingPayloadEnvelopes.
func (s *Service) prunePendingPayloadEnvelopesHeze() {
	s.pendingHezeEnvelopesLock.Lock()
	defer s.pendingHezeEnvelopesLock.Unlock()

	finalizedEpoch := s.cfg.chain.FinalizedCheckpt().Epoch
	for root, signed := range s.pendingHezeEnvelopes {
		env, err := signed.Envelope()
		if err != nil || slots.ToEpoch(env.Slot()) < finalizedEpoch {
			delete(s.pendingHezeEnvelopes, root)
		}
	}
}
