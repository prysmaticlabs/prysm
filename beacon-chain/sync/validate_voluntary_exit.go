package sync

import (
	"context"

	"github.com/libp2p/go-libp2p-core/peer"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/traceutil"
	"go.opencensus.io/trace"
)

// Clients who receive a voluntary exit on this topic MUST validate the conditions within process_voluntary_exit before
// forwarding it across the network.
func (r *Service) validateVoluntaryExit(ctx context.Context, pid peer.ID, msg *pubsub.Message) bool {
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

	exit, ok := m.(*ethpb.VoluntaryExit)
	if !ok {
		return false
	}

	// Retrieve head state, advance state to the epoch slot used specified in exit message.
	s, err := r.chain.HeadState(ctx)
	if err != nil {
		return false
	}

	exitedEpochSlot := exit.Epoch * params.BeaconConfig().SlotsPerEpoch
	if s.Slot < exitedEpochSlot {
		var err error
		s, err = state.ProcessSlots(ctx, s, exitedEpochSlot)
		if err != nil {
			return false
		}
	}

	if err := blocks.VerifyExit(s, exit); err != nil {
		return false
	}

	msg.ValidatorData = exit // Used in downstream subscriber

	return true
}
