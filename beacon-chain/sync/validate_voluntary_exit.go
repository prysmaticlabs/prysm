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

	s, err := r.chain.HeadState(ctx)
	if err != nil {
		return false
	}

	exitedEpochSlot := exit.Exit.Epoch * params.BeaconConfig().SlotsPerEpoch
	if int(exit.Exit.ValidatorIndex) >= s.NumValidators() {
		return false
	}
	val, err := s.ValidatorAtIndex(exit.Exit.ValidatorIndex)
	if err != nil {
		return false
	}
	if err := blocks.VerifyExit(val, exitedEpochSlot, s.Fork(), exit); err != nil {
		return false
	}

	msg.ValidatorData = exit // Used in downstream subscriber

	return true
}
