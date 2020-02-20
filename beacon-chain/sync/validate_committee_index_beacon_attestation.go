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
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/traceutil"
	"go.opencensus.io/trace"
)

// Validation
// - The attestation's committee index (attestation.data.index) is for the correct subnet.
// - The attestation is unaggregated -- that is, it has exactly one participating validator (len([bit for bit in attestation.aggregation_bits if bit == 0b1]) == 1).
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

	// The attestation's committee index (attestation.data.index) is for the correct subnet.
	if !strings.HasPrefix(originalTopic, fmt.Sprintf(format, att.Data.CommitteeIndex)) {
		return false
	}

	// Attestation must be unaggregated.
	if att.AggregationBits == nil || att.AggregationBits.Count() != 1 {
		return false
	}

	// Attestation's slot is within ATTESTATION_PROPAGATION_SLOT_RANGE.
	currentSlot := helpers.SlotsSince(s.chain.GenesisTime())
	upper := att.Data.Slot + params.BeaconConfig().AttestationPropagationSlotRange
	lower := att.Data.Slot
	if currentSlot > upper || currentSlot < lower {
		return false
	}

	// Verify the block being voted and the processed state is in DB and. The block should have passed validation if it's in the DB.
	hasState := s.db.HasState(ctx, bytesutil.ToBytes32(att.Data.BeaconBlockRoot))
	hasBlock := s.db.HasBlock(ctx, bytesutil.ToBytes32(att.Data.BeaconBlockRoot))
	if !(hasState && hasBlock) {
		// A node doesn't have the block, it'll request from peer while saving the pending attestation to a queue.
		s.savePendingAtt(&eth.AggregateAttestationAndProof{Aggregate: att})
		return false
	}

	// Attestation's signature is a valid BLS signature and belongs to correct public key..
	if !featureconfig.Get().DisableStrictAttestationPubsubVerification && !s.chain.IsValidAttestation(ctx, att) {
		return false
	}

	msg.ValidatorData = att

	return true
}
