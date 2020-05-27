package sync

import (
	"context"

	"github.com/libp2p/go-libp2p-core/peer"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	"github.com/prysmaticlabs/prysm/shared/traceutil"
	"go.opencensus.io/trace"
)

// Clients who receive a proposer slashing on this topic MUST validate the conditions within VerifyProposerSlashing before
// forwarding it across the network.
func (r *Service) validateProposerSlashing(ctx context.Context, pid peer.ID, msg *pubsub.Message) pubsub.ValidationResult {
	// Validation runs on publish (not just subscriptions), so we should approve any message from
	// ourselves.
	if pid == r.p2p.PeerID() {
		return pubsub.ValidationAccept
	}

	// The head state will be too far away to validate any slashing.
	if r.initialSync.Syncing() {
		return pubsub.ValidationIgnore
	}

	ctx, span := trace.StartSpan(ctx, "sync.validateProposerSlashing")
	defer span.End()

	m, err := r.decodePubsubMessage(msg)
	if err != nil {
		log.WithError(err).Error("Failed to decode message")
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
	if r.hasSeenProposerSlashingIndex(slashing.Header_1.Header.ProposerIndex) {
		return pubsub.ValidationIgnore
	}

	// Retrieve head state, advance state to the epoch slot used specified in slashing message.
	s, err := r.chain.HeadState(ctx)
	if err != nil {
		return pubsub.ValidationIgnore
	}
	slashSlot := slashing.Header_1.Header.Slot
	if s.Slot() < slashSlot {
		if ctx.Err() != nil {
			return pubsub.ValidationIgnore
		}
		var err error
		s, err = state.ProcessSlots(ctx, s, slashSlot)
		if err != nil {
			return pubsub.ValidationIgnore
		}
	}

	if err := blocks.VerifyProposerSlashing(s, slashing); err != nil {
		return pubsub.ValidationReject
	}

	msg.ValidatorData = slashing // Used in downstream subscriber
	return pubsub.ValidationAccept
}

// Returns true if the node has already received a valid proposer slashing received for the proposer with index
func (r *Service) hasSeenProposerSlashingIndex(i uint64) bool {
	r.seenProposerSlashingLock.RLock()
	defer r.seenProposerSlashingLock.RUnlock()
	_, seen := r.seenProposerSlashingCache.Get(i)
	return seen
}

// Set proposer slashing index in proposer slashing cache.
func (r *Service) setProposerSlashingIndexSeen(i uint64) {
	r.seenProposerSlashingLock.Lock()
	defer r.seenProposerSlashingLock.Unlock()
	r.seenProposerSlashingCache.Add(i, true)
}
