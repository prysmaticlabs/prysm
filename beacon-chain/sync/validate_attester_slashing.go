package sync

import (
	"context"

	"github.com/libp2p/go-libp2p-core/peer"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/shared/sliceutil"
	"github.com/prysmaticlabs/prysm/shared/traceutil"
	"go.opencensus.io/trace"
)

// Clients who receive an attester slashing on this topic MUST validate the conditions within VerifyAttesterSlashing before
// forwarding it across the network.
func (s *Service) validateAttesterSlashing(ctx context.Context, pid peer.ID, msg *pubsub.Message) pubsub.ValidationResult {
	// Validation runs on publish (not just subscriptions), so we should approve any message from
	// ourselves.
	if pid == s.cfg.P2P.PeerID() {
		return pubsub.ValidationAccept
	}

	// The head state will be too far away to validate any slashing.
	if s.cfg.InitialSync.Syncing() {
		return pubsub.ValidationIgnore
	}

	ctx, span := trace.StartSpan(ctx, "sync.validateAttesterSlashing")
	defer span.End()

	m, err := s.decodePubsubMessage(msg)
	if err != nil {
		log.WithError(err).Debug("Could not decode message")
		traceutil.AnnotateError(span, err)
		return pubsub.ValidationReject
	}
	slashing, ok := m.(*ethpb.AttesterSlashing)
	if !ok {
		return pubsub.ValidationReject
	}

	if slashing == nil || slashing.Attestation_1 == nil || slashing.Attestation_2 == nil {
		return pubsub.ValidationReject
	}
	if s.hasSeenAttesterSlashingIndices(slashing.Attestation_1.AttestingIndices, slashing.Attestation_2.AttestingIndices) {
		return pubsub.ValidationIgnore
	}

	headState, err := s.cfg.Chain.HeadState(ctx)
	if err != nil {
		return pubsub.ValidationIgnore
	}
	if err := blocks.VerifyAttesterSlashing(ctx, headState, slashing); err != nil {
		return pubsub.ValidationReject
	}

	msg.ValidatorData = slashing // Used in downstream subscriber
	return pubsub.ValidationAccept
}

// Returns true if the node has already received a valid attester slashing with the attesting indices.
func (s *Service) hasSeenAttesterSlashingIndices(indices1, indices2 []uint64) bool {
	slashableIndices := sliceutil.IntersectionUint64(indices1, indices2)

	s.seenAttesterSlashingLock.RLock()
	defer s.seenAttesterSlashingLock.RUnlock()

	// Return false if any of the slashing index has not been seen before. (i.e. not in cache)
	for _, index := range slashableIndices {
		if !s.seenAttesterSlashingCache[index] {
			return false
		}
	}

	return true
}

// Set attester slashing indices in attester slashing cache.
func (s *Service) setAttesterSlashingIndicesSeen(indices1, indices2 []uint64) {
	slashableIndices := sliceutil.IntersectionUint64(indices1, indices2)

	s.seenAttesterSlashingLock.Lock()
	defer s.seenAttesterSlashingLock.Unlock()

	for _, index := range slashableIndices {
		s.seenAttesterSlashingCache[index] = true
	}
}
