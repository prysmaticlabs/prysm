package sync

import (
	"context"
	"fmt"

	"github.com/libp2p/go-libp2p-core/peer"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/shared/attestationutil"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/roughtime"
	"github.com/prysmaticlabs/prysm/shared/traceutil"
	"go.opencensus.io/trace"
)

// validateAggregateAndProof verifies the aggregated signature and the selection proof is valid before forwarding to the
// network and downstream services.
func (r *Service) validateAggregateAndProof(ctx context.Context, pid peer.ID, msg *pubsub.Message) bool {
	if pid == r.p2p.PeerID() {
		return true
	}

	ctx, span := trace.StartSpan(ctx, "sync.validateAggregateAndProof")
	defer span.End()

	// To process the following it requires the recent blocks to be present in the database, so we'll skip
	// validating or processing aggregated attestations until fully synced.
	if r.initialSync.Syncing() {
		return false
	}

	raw, err := r.decodePubsubMessage(msg)
	if err != nil {
		log.WithError(err).Error("Failed to decode message")
		traceutil.AnnotateError(span, err)
		return false
	}
	m, ok := raw.(*ethpb.AggregateAttestationAndProof)
	if !ok {
		return false
	}

	// Verify aggregate attestation has not already been seen via aggregate gossip, within a block, or through the creation locally.
	seen, err := r.attPool.HasAggregatedAttestation(m.Aggregate)
	if err != nil {
		traceutil.AnnotateError(span, err)
		return false
	}
	if seen {
		return false
	}
	if !r.validateBlockInAttestation(ctx, m) {
		return false
	}

	if !r.validateAggregatedAtt(ctx, m) {
		return false
	}

	if !featureconfig.Get().DisableStrictAttestationPubsubVerification && !r.chain.IsValidAttestation(ctx, m.Aggregate) {
		return false
	}

	msg.ValidatorData = m

	return true
}

func (r *Service) validateAggregatedAtt(ctx context.Context, a *ethpb.AggregateAttestationAndProof) bool {
	ctx, span := trace.StartSpan(ctx, "sync.validateAggregatedAtt")
	defer span.End()

	attSlot := a.Aggregate.Data.Slot

	// Verify attestation slot is within the last ATTESTATION_PROPAGATION_SLOT_RANGE slots.
	currentSlot := uint64(roughtime.Now().Unix()-r.chain.GenesisTime().Unix()) / params.BeaconConfig().SecondsPerSlot
	if attSlot > currentSlot || currentSlot > attSlot+params.BeaconConfig().AttestationPropagationSlotRange {
		traceutil.AnnotateError(span, fmt.Errorf("attestation slot out of range %d <= %d <= %d", attSlot, currentSlot, attSlot+params.BeaconConfig().AttestationPropagationSlotRange))
		return false

	}

	s, err := r.chain.HeadState(ctx)
	if err != nil {
		traceutil.AnnotateError(span, err)
		return false
	}

	// Only advance state if different epoch as the committee can only change on an epoch transition.
	if helpers.SlotToEpoch(attSlot) > helpers.SlotToEpoch(s.Slot()) {
		s, err = state.ProcessSlots(ctx, s, helpers.StartSlot(helpers.SlotToEpoch(attSlot)))
		if err != nil {
			traceutil.AnnotateError(span, err)
			return false
		}
	}

	// Verify validator index is within the aggregate's committee.
	if err := validateIndexInCommittee(ctx, s, a.Aggregate, a.AggregatorIndex); err != nil {
		traceutil.AnnotateError(span, errors.Wrapf(err, "Could not validate index in committee"))
		return false
	}

	// Verify selection proof reflects to the right validator and signature is valid.
	if err := validateSelection(ctx, s, a.Aggregate.Data, a.AggregatorIndex, a.SelectionProof); err != nil {
		traceutil.AnnotateError(span, errors.Wrapf(err, "Could not validate selection for validator %d", a.AggregatorIndex))
		return false
	}

	// Verify aggregated attestation has a valid signature.
	if err := blocks.VerifyAttestation(ctx, s, a.Aggregate); err != nil {
		traceutil.AnnotateError(span, err)
		return false
	}

	return true
}

func (r *Service) validateBlockInAttestation(ctx context.Context, a *ethpb.AggregateAttestationAndProof) bool {
	// Verify the block being voted and the processed state is in DB. The block should have passed validation if it's in the DB.
	hasState := r.db.HasState(ctx, bytesutil.ToBytes32(a.Aggregate.Data.BeaconBlockRoot))
	hasBlock := r.db.HasBlock(ctx, bytesutil.ToBytes32(a.Aggregate.Data.BeaconBlockRoot))
	if !(hasState && hasBlock) {
		// A node doesn't have the block, it'll request from peer while saving the pending attestation to a queue.
		r.savePendingAtt(a)
		return false
	}
	return true
}

// This validates the aggregator's index in state is within the attesting indices of the attestation.
func validateIndexInCommittee(ctx context.Context, s *stateTrie.BeaconState, a *ethpb.Attestation, validatorIndex uint64) error {
	ctx, span := trace.StartSpan(ctx, "sync.validateIndexInCommittee")
	defer span.End()

	committee, err := helpers.BeaconCommitteeFromState(s, a.Data.Slot, a.Data.CommitteeIndex)
	if err != nil {
		return err
	}
	attestingIndices, err := attestationutil.AttestingIndices(a.AggregationBits, committee)
	if err != nil {
		return err
	}
	var withinCommittee bool
	for _, i := range attestingIndices {
		if validatorIndex == i {
			withinCommittee = true
			break
		}
	}
	if !withinCommittee {
		return fmt.Errorf("validator index %d is not within the committee: %v",
			validatorIndex, attestingIndices)
	}
	return nil
}

// This validates selection proof by validating it's from the correct validator index of the slot and selection
// proof is a valid signature.
func validateSelection(ctx context.Context, s *stateTrie.BeaconState, data *ethpb.AttestationData, validatorIndex uint64, proof []byte) error {
	_, span := trace.StartSpan(ctx, "sync.validateSelection")
	defer span.End()

	committee, err := helpers.BeaconCommitteeFromState(s, data.Slot, data.CommitteeIndex)
	if err != nil {
		return err
	}
	aggregator, err := helpers.IsAggregator(uint64(len(committee)), proof)
	if err != nil {
		return err
	}
	if !aggregator {
		return fmt.Errorf("validator is not an aggregator for slot %d", data.Slot)
	}

	domain := helpers.Domain(s.Fork(), helpers.SlotToEpoch(data.Slot), params.BeaconConfig().DomainBeaconAttester)
	slotMsg, err := ssz.HashTreeRoot(data.Slot)
	if err != nil {
		return err
	}
	pubkeyState := s.PubkeyAtIndex(validatorIndex)
	pubKey, err := bls.PublicKeyFromBytes(pubkeyState[:])
	if err != nil {
		return err
	}
	slotSig, err := bls.SignatureFromBytes(proof)
	if err != nil {
		return err
	}
	if !slotSig.Verify(slotMsg[:], pubKey, domain) {
		return errors.New("could not validate slot signature")
	}

	return nil
}
