package sync

import (
	"context"

	"github.com/libp2p/go-libp2p-core/peer"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	types "github.com/prysmaticlabs/eth2-types"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/shared/traceutil"
	"go.opencensus.io/trace"
)

// Clients who receive a voluntary exit on this topic MUST validate the conditions within process_voluntary_exit before
// forwarding it across the network.
func (s *Service) validateVoluntaryExit(ctx context.Context, pid peer.ID, msg *pubsub.Message) pubsub.ValidationResult {
	// Validation runs on publish (not just subscriptions), so we should approve any message from
	// ourselves.
	if pid == s.cfg.P2P.PeerID() {
		return pubsub.ValidationAccept
	}

	// The head state will be too far away to validate any voluntary exit.
	if s.cfg.InitialSync.Syncing() {
		return pubsub.ValidationIgnore
	}

	ctx, span := trace.StartSpan(ctx, "sync.validateVoluntaryExit")
	defer span.End()

	m, err := s.decodePubsubMessage(msg)
	if err != nil {
		log.WithError(err).Debug("Could not decode message")
		traceutil.AnnotateError(span, err)
		return pubsub.ValidationReject
	}

	exit, ok := m.(*ethpb.SignedVoluntaryExit)
	if !ok {
		return pubsub.ValidationReject
	}

	if exit.Exit == nil {
		return pubsub.ValidationReject
	}
	if s.hasSeenExitIndex(exit.Exit.ValidatorIndex) {
		return pubsub.ValidationIgnore
	}

	headState, err := s.cfg.Chain.HeadState(ctx)
	if err != nil {
		return pubsub.ValidationIgnore
	}

	if uint64(exit.Exit.ValidatorIndex) >= uint64(headState.NumValidators()) {
		return pubsub.ValidationReject
	}
	val, err := headState.ValidatorAtIndexReadOnly(exit.Exit.ValidatorIndex)
	if err != nil {
		return pubsub.ValidationIgnore
	}
	if err := blocks.VerifyExitAndSignature(val, headState.Slot(), headState.Fork(), exit, headState.GenesisValidatorRoot()); err != nil {
		return pubsub.ValidationReject
	}

	msg.ValidatorData = exit // Used in downstream subscriber

	return pubsub.ValidationAccept
}

// Returns true if the node has already received a valid exit request for the validator with index `i`.
func (s *Service) hasSeenExitIndex(i types.ValidatorIndex) bool {
	s.seenExitLock.RLock()
	defer s.seenExitLock.RUnlock()
	_, seen := s.seenExitCache.Get(i)
	return seen
}

// Set exit request index `i` in seen exit request cache.
func (s *Service) setExitIndexSeen(i types.ValidatorIndex) {
	s.seenExitLock.Lock()
	defer s.seenExitLock.Unlock()
	s.seenExitCache.Add(i, true)
}
