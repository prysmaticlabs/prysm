package sync

import (
	"context"
	"strings"

	"github.com/gogo/protobuf/proto"
	"github.com/libp2p/go-libp2p-core/peer"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/traceutil"
	"go.opencensus.io/trace"
)

// Clients who receive an attester slashing on this topic MUST validate the conditions within VerifyAttesterSlashing before
// forwarding it across the network.
func (r *Service) validateAttesterSlashing(ctx context.Context, pid peer.ID, msg *pubsub.Message) bool {
	// The head state will be too far away to validate any slashing.
	if r.initialSync.Syncing() {
		return false
	}

	ctx, span := trace.StartSpan(ctx, "sync.validateAttesterSlashing")
	defer span.End()

	if msg == nil || msg.TopicIDs == nil || len(msg.TopicIDs) == 0 {
		return false
	}
	topic := msg.TopicIDs[0]
	topic = strings.TrimSuffix(topic, r.p2p.Encoding().ProtocolSuffix())
	base, ok := p2p.GossipTopicMappings[topic]
	if !ok {
		return false
	}
	m := proto.Clone(base)
	if err := r.p2p.Encoding().Decode(msg.Data, m); err != nil {
		traceutil.AnnotateError(span, err)
		log.WithError(err).Warn("Failed to decode pubsub message")
		return false
	}

	slashing, ok := m.(*ethpb.AttesterSlashing)
	if !ok {
		return false
	}

	// Retrieve head state, advance state to the epoch slot used specified in slashing message.
	s, err := r.chain.HeadState(ctx)
	if err != nil {
		return false
	}
	slashSlot := slashing.Attestation_1.Data.Target.Epoch * params.BeaconConfig().SlotsPerEpoch
	if s.Slot < slashSlot {
		if ctx.Err() != nil {
			return false
		}

		var err error
		s, err = state.ProcessSlots(ctx, s, slashSlot)
		if err != nil {
			return false
		}
	}

	if err := blocks.VerifyAttesterSlashing(ctx, s, slashing); err != nil {
		return false
	}

	if pid == r.p2p.PeerID() {
		return false
	}

	msg.VaidatorData = slashing // Used in downstream subscriber
	return true
}
