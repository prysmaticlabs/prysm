package sync

import (
	"context"

	"github.com/libp2p/go-libp2p-core/peer"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/traceutil"
	"go.opencensus.io/trace"
)

// Clients who receive a proposer slashing on this topic MUST validate the conditions within VerifyProposerSlashing before
// forwarding it across the network.
func (s *Service) validateProposerSlashing(ctx context.Context, pid peer.ID, msg *pubsub.Message) pubsub.ValidationResult {
	// Validation runs on publish (not just subscriptions), so we should approve any message from
	// ourselves.
	if pid == s.cfg.P2P.PeerID() {
		return pubsub.ValidationAccept
	}

	// The head state will be too far away to validate any slashing.
	if s.cfg.InitialSync.Syncing() {
		return pubsub.ValidationIgnore
	}

	ctx, span := trace.StartSpan(ctx, "sync.validateProposerSlashing")
	defer span.End()

	m, err := s.decodePubsubMessage(msg)
	if err != nil {
		log.WithError(err).Debug("Could not decode message")
		traceutil.AnnotateError(span, err)
		return pubsub.ValidationReject
	}

	slashing, ok := m.(*ethpb.ProposerSlashing)
	if !ok {
		return pubsub.ValidationReject
	}

	if slashing.Header_1 == nil || slashing.Header_1.Header == nil {
		return pubsub.ValidationReject
	}
	if s.hasSeenProposerSlashingIndex(slashing.Header_1.Header.ProposerIndex) {
		return pubsub.ValidationIgnore
	}

	headState, err := s.cfg.Chain.HeadState(ctx)
	if err != nil {
		return pubsub.ValidationIgnore
	}
	if err := blocks.VerifyProposerSlashing(headState, slashing); err != nil {
		return pubsub.ValidationReject
	}

	msg.ValidatorData = slashing // Used in downstream subscriber
	return pubsub.ValidationAccept
}

// Returns true if the node has already received a valid proposer slashing received for the proposer with index
func (s *Service) hasSeenProposerSlashingIndex(i types.ValidatorIndex) bool {
	s.seenProposerSlashingLock.RLock()
	defer s.seenProposerSlashingLock.RUnlock()
	_, seen := s.seenProposerSlashingCache.Get(i)
	return seen
}

// Set proposer slashing index in proposer slashing cache.
func (s *Service) setProposerSlashingIndexSeen(i types.ValidatorIndex) {
	s.seenProposerSlashingLock.Lock()
	defer s.seenProposerSlashingLock.Unlock()
	s.seenProposerSlashingCache.Add(i, true)
}
