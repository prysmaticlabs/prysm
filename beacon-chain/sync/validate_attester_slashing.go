package sync

import (
	"context"
	"sort"

	"github.com/libp2p/go-libp2p-core/peer"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/sliceutil"
	"github.com/prysmaticlabs/prysm/shared/traceutil"
	"go.opencensus.io/trace"
)

// Clients who receive an attester slashing on this topic MUST validate the conditions within VerifyAttesterSlashing before
// forwarding it across the network.
func (r *Service) validateAttesterSlashing(ctx context.Context, pid peer.ID, msg *pubsub.Message) pubsub.ValidationResult {
	// Validation runs on publish (not just subscriptions), so we should approve any message from
	// ourselves.
	if pid == r.p2p.PeerID() {
		return pubsub.ValidationAccept
	}

	// The head state will be too far away to validate any slashing.
	if r.initialSync.Syncing() {
		return pubsub.ValidationIgnore
	}

	ctx, span := trace.StartSpan(ctx, "sync.validateAttesterSlashing")
	defer span.End()

	m, err := r.decodePubsubMessage(msg)
	if err != nil {
		log.WithError(err).Error("Failed to decode message")
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
	if r.hasSeenAttesterSlashingIndices(slashing.Attestation_1.AttestingIndices, slashing.Attestation_2.AttestingIndices) {
		return pubsub.ValidationIgnore
	}

	// Retrieve head state, advance state to the epoch slot used specified in slashing message.
	s, err := r.chain.HeadState(ctx)
	if err != nil {
		return pubsub.ValidationIgnore
	}
	slashSlot := slashing.Attestation_1.Data.Target.Epoch * params.BeaconConfig().SlotsPerEpoch
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

	if err := blocks.VerifyAttesterSlashing(ctx, s, slashing); err != nil {
		return pubsub.ValidationReject
	}

	msg.ValidatorData = slashing // Used in downstream subscriber
	return pubsub.ValidationAccept
}

// Returns true if the node has already received a valid attester slashing with the attesting indices.
func (r *Service) hasSeenAttesterSlashingIndices(indices1 []uint64, indices2 []uint64) bool {
	r.seenAttesterSlashingLock.RLock()
	defer r.seenAttesterSlashingLock.RUnlock()

	slashableIndices := sliceutil.IntersectionUint64(indices1, indices2)
	sort.SliceStable(slashableIndices, func(i, j int) bool {
		return slashableIndices[i] < slashableIndices[j]
	})
	IndicesInBytes := make([]byte, 0, len(slashableIndices))
	for _, i := range slashableIndices {
		IndicesInBytes = append(IndicesInBytes, byte(i))
	}
	b := hashutil.FastSum256(IndicesInBytes)

	_, seen := r.seenAttesterSlashingCache.Get(b)
	return seen
}

// Set attester slashing indices in attester slashing cache.
func (r *Service) setAttesterSlashingIndicesSeen(indices1 []uint64, indices2 []uint64) {
	r.seenAttesterSlashingLock.Lock()
	defer r.seenAttesterSlashingLock.Unlock()

	slashableIndices := sliceutil.IntersectionUint64(indices1, indices2)
	sort.SliceStable(slashableIndices, func(i, j int) bool {
		return slashableIndices[i] < slashableIndices[j]
	})
	IndicesInBytes := make([]byte, 0, len(slashableIndices))
	for _, i := range slashableIndices {
		IndicesInBytes = append(IndicesInBytes, byte(i))
	}
	b := hashutil.FastSum256(IndicesInBytes)

	r.seenAttesterSlashingCache.Add(b, true)
}
