package sync

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/libp2p/go-libp2p-core/peer"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	eth "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p"
	"github.com/prysmaticlabs/prysm/shared/attestationutil"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/traceutil"
	"go.opencensus.io/trace"
)

// Validation
// - The attestation's committee index (attestation.data.index) is for the correct subnet.
// - The attestation is unaggregated -- that is, it has exactly one participating validator (len(get_attesting_indices(state, attestation.data, attestation.aggregation_bits)) == 1).
// - The block being voted for (attestation.data.beacon_block_root) passes validation.
// - attestation.data.slot is within the last ATTESTATION_PROPAGATION_SLOT_RANGE slots (attestation.data.slot + ATTESTATION_PROPAGATION_SLOT_RANGE >= current_slot >= attestation.data.slot).
// - The signature of attestation is valid.
func (s *Service) validateCommitteeIndexBeaconAttestation(ctx context.Context, pid peer.ID, msg *pubsub.Message) bool {
	if pid == s.p2p.PeerID() {
		return true
	}
	// Attestation processing requires the target block to be present in the database, so we'll skip
	// validating or processing attestations until fully synced.
	if s.initialSync.Syncing() {
		return false
	}
	ctx, span := trace.StartSpan(ctx, "sync.validateCommitteeIndexBeaconAttestation")
	defer span.End()

	// Override topic for decoding.
	originalTopic := msg.TopicIDs[0]
	format := p2p.GossipTypeMapping[reflect.TypeOf(&eth.Attestation{})]
	msg.TopicIDs[0] = format

	m, err := s.decodePubsubMessage(msg)
	if err != nil {
		log.WithError(err).Error("Failed to decode message")
		traceutil.AnnotateError(span, err)
		return false
	}
	// Restore topic.
	msg.TopicIDs[0] = originalTopic

	att, ok := m.(*eth.Attestation)
	if !ok {
		return false
	}

	if att.Data == nil {
		return false
	}

	// Verify this the first attestation received for the participating validator for the slot.
	if s.hasSeenCommitteeIndicesSlot(att.Data.Slot, att.Data.CommitteeIndex, att.AggregationBits) {
		return false
	}

	// The attestation's committee index (attestation.data.index) is for the correct subnet.
	digest, err := s.forkDigest()
	if err != nil {
		log.WithError(err).Error("Failed to compute fork digest")
		traceutil.AnnotateError(span, err)
		return false
	}
	if !strings.HasPrefix(originalTopic, fmt.Sprintf(format, digest, att.Data.CommitteeIndex)) {
		return false
	}

	// Attestation aggregation bits must exist.
	if att.AggregationBits == nil {
		return false
	}
	st, err := s.chain.AttestationPreState(ctx, att)
	if err != nil {
		traceutil.AnnotateError(span, err)
		return false
	}
	committee, err := helpers.BeaconCommitteeFromState(st, att.Data.Slot, att.Data.CommitteeIndex)
	if err != nil {
		return false
	}

	// Attestation must be unaggregated.
	if len(attestationutil.AttestingIndices(att.AggregationBits, committee)) != 1 {
		return false
	}

	// Attestation's slot is within ATTESTATION_PROPAGATION_SLOT_RANGE.
	if err := validateAggregateAttTime(att.Data.Slot, uint64(s.chain.GenesisTime().Unix())); err != nil {
		traceutil.AnnotateError(span, err)
		return false
	}

	// Verify the block being voted and the processed state is in DB and. The block should have passed validation if it's in the DB.
	blockRoot := bytesutil.ToBytes32(att.Data.BeaconBlockRoot)
	hasStateSummary := featureconfig.Get().NewStateMgmt && s.db.HasStateSummary(ctx, blockRoot) || s.stateSummaryCache.Has(blockRoot)
	hasState := s.db.HasState(ctx, blockRoot) || hasStateSummary
	hasBlock := s.db.HasBlock(ctx, blockRoot)
	if !(hasState && hasBlock) {
		// A node doesn't have the block, it'll request from peer while saving the pending attestation to a queue.
		s.savePendingAtt(&eth.SignedAggregateAttestationAndProof{Message: &eth.AggregateAttestationAndProof{Aggregate: att}})
		return false
	}

	// Attestation's signature is a valid BLS signature and belongs to correct public key..
	if !featureconfig.Get().DisableStrictAttestationPubsubVerification && !s.chain.IsValidAttestation(ctx, att) {
		return false
	}

	s.setSeenCommitteeIndicesSlot(att.Data.Slot, att.Data.CommitteeIndex, att.AggregationBits)

	msg.ValidatorData = att

	return true
}

// Returns true if the attestation was already seen for the participating validator for the slot.
func (s *Service) hasSeenCommitteeIndicesSlot(slot uint64, committeeID uint64, aggregateBits []byte) bool {
	s.seenAttestationLock.RLock()
	defer s.seenAttestationLock.RUnlock()
	b := append(bytesutil.Bytes32(slot), bytesutil.Bytes32(committeeID)...)
	b = append(b, aggregateBits...)
	_, seen := s.seenAttestationCache.Get(string(b))
	return seen
}

// Set committee's indices and slot as seen for incoming attestations.
func (s *Service) setSeenCommitteeIndicesSlot(slot uint64, committeeID uint64, aggregateBits []byte) {
	s.seenAttestationLock.Lock()
	defer s.seenAttestationLock.Unlock()
	b := append(bytesutil.Bytes32(slot), bytesutil.Bytes32(committeeID)...)
	b = append(b, aggregateBits...)
	s.seenAttestationCache.Add(string(b), true)
}
