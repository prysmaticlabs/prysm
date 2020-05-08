package sync

import (
	"context"

	"github.com/libp2p/go-libp2p-core/peer"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/traceutil"
	"go.opencensus.io/trace"
)

// Clients who receive a voluntary exit on this topic MUST validate the conditions within process_voluntary_exit before
// forwarding it across the network.
func (r *Service) validateVoluntaryExit(ctx context.Context, pid peer.ID, msg *pubsub.Message) bool {
	// Validation runs on publish (not just subscriptions), so we should approve any message from
	// ourselves.
	if pid == r.p2p.PeerID() {
		return true
	}

	// The head state will be too far away to validate any voluntary exit.
	if r.initialSync.Syncing() {
		return false
	}

	ctx, span := trace.StartSpan(ctx, "sync.validateVoluntaryExit")
	defer span.End()

	m, err := r.decodePubsubMessage(msg)
	if err != nil {
		log.WithError(err).Error("Failed to decode message")
		traceutil.AnnotateError(span, err)
		return false
	}

	exit, ok := m.(*ethpb.SignedVoluntaryExit)
	if !ok {
		return false
	}

	if exit.Exit == nil {
		return false
	}
	if r.hasSeenExitIndex(exit.Exit.ValidatorIndex) {
		return false
	}

	s, err := r.chain.HeadState(ctx)
	if err != nil {
		return false
	}

	exitedEpochSlot := exit.Exit.Epoch * params.BeaconConfig().SlotsPerEpoch
	if int(exit.Exit.ValidatorIndex) >= s.NumValidators() {
		return false
	}
	val, err := s.ValidatorAtIndexReadOnly(exit.Exit.ValidatorIndex)
	if err != nil {
		return false
	}
	if err := blocks.VerifyExit(val, exitedEpochSlot, s.Fork(), exit, s.GenesisValidatorRoot()); err != nil {
		return false
	}

	msg.ValidatorData = exit // Used in downstream subscriber

	return true
}

// Returns true if the node has already received a valid exit request for the validator with index `i`.
func (r *Service) hasSeenExitIndex(i uint64) bool {
	r.seenExitLock.RLock()
	defer r.seenExitLock.RUnlock()
	_, seen := r.seenExitCache.Get(i)
	return seen
}

// Set exit request index `i` in seen exit request cache.
func (r *Service) setExitIndexSeen(i uint64) {
	r.seenExitLock.Lock()
	defer r.seenExitLock.Unlock()
	r.seenExitCache.Add(i, true)
}
