package sync

import (
	"context"
	"fmt"

	"github.com/libp2p/go-libp2p-core/peer"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/traceutil"
	"go.opencensus.io/trace"
)

// validateAggregateAndProof verifies the aggregated signature and the selection proof is valid before forwarding to the
// network and downstream services.
func (s *Service) validateAggregateAndProof(ctx context.Context, pid peer.ID, msg *pubsub.Message) pubsub.ValidationResult {
	if pid == s.p2p.PeerID() {
		return pubsub.ValidationAccept
	}

	ctx, span := trace.StartSpan(ctx, "sync.validateAggregateAndProof")
	defer span.End()

	// To process the following it requires the recent blocks to be present in the database, so we'll skip
	// validating or processing aggregated attestations until fully synced.
	if s.initialSync.Syncing() {
		return pubsub.ValidationIgnore
	}

	raw, err := s.decodePubsubMessage(msg)
	if err != nil {
		log.WithError(err).Debug("Failed to decode message")
		traceutil.AnnotateError(span, err)
		return pubsub.ValidationReject
	}
	m, ok := raw.(*ethpb.SignedAggregateAttestationAndProof)
	if !ok {
		return pubsub.ValidationReject
	}
	if err := helpers.ValidateAttestationTime(m.Message.Aggregate.Data.Slot, s.chain.GenesisTime()); err != nil {
		traceutil.AnnotateError(span, err)
		return pubsub.ValidationIgnore
	}

	if m.Message == nil || m.Message.Aggregate == nil || m.Message.Aggregate.Data == nil {
		return pubsub.ValidationReject
	}
	// Verify this is the first aggregate received from the aggregator with index and slot.
	if s.hasSeenAggregatorIndexEpoch(m.Message.Aggregate.Data.Target.Epoch, m.Message.AggregatorIndex) {
		return pubsub.ValidationIgnore
	}
	// Check that the block being voted on isn't invalid.
	if s.hasBadBlock(bytesutil.ToBytes32(m.Message.Aggregate.Data.BeaconBlockRoot)) ||
		s.hasBadBlock(bytesutil.ToBytes32(m.Message.Aggregate.Data.Target.Root)) ||
		s.hasBadBlock(bytesutil.ToBytes32(m.Message.Aggregate.Data.Source.Root)) {
		return pubsub.ValidationReject
	}

	// Verify aggregate attestation has not already been seen via aggregate gossip, within a block, or through the creation locally.
	seen, err := s.attPool.HasAggregatedAttestation(m.Message.Aggregate)
	if err != nil {
		traceutil.AnnotateError(span, err)
		return pubsub.ValidationIgnore
	}
	if seen {
		return pubsub.ValidationIgnore
	}
	if !s.validateBlockInAttestation(ctx, m) {
		return pubsub.ValidationIgnore
	}

	validationRes := s.validateAggregatedAtt(ctx, m)
	if validationRes != pubsub.ValidationAccept {
		return validationRes
	}

	s.setAggregatorIndexEpochSeen(m.Message.Aggregate.Data.Target.Epoch, m.Message.AggregatorIndex)

	msg.ValidatorData = m

	return pubsub.ValidationAccept
}

func (s *Service) validateAggregatedAtt(ctx context.Context, signed *ethpb.SignedAggregateAttestationAndProof) pubsub.ValidationResult {
	ctx, span := trace.StartSpan(ctx, "sync.validateAggregatedAtt")
	defer span.End()

	attSlot := signed.Message.Aggregate.Data.Slot

	bs, err := s.chain.AttestationPreState(ctx, signed.Message.Aggregate)
	if err != nil {
		traceutil.AnnotateError(span, err)
		return pubsub.ValidationIgnore
	}

	// Only advance state if different epoch as the committee can only change on an epoch transition.
	if helpers.SlotToEpoch(attSlot) > helpers.SlotToEpoch(bs.Slot()) {
		bs, err = state.ProcessSlots(ctx, bs, helpers.StartSlot(helpers.SlotToEpoch(attSlot)))
		if err != nil {
			traceutil.AnnotateError(span, err)
			return pubsub.ValidationIgnore
		}
	}

	// Verify validator index is within the beacon committee.
	if err := validateIndexInCommittee(ctx, bs, signed.Message.Aggregate, signed.Message.AggregatorIndex); err != nil {
		traceutil.AnnotateError(span, errors.Wrapf(err, "Could not validate index in committee"))
		return pubsub.ValidationReject
	}

	// Verify selection proof reflects to the right validator and signature is valid.
	if err := validateSelection(ctx, bs, signed.Message.Aggregate.Data, signed.Message.AggregatorIndex, signed.Message.SelectionProof); err != nil {
		traceutil.AnnotateError(span, errors.Wrapf(err, "Could not validate selection for validator %d", signed.Message.AggregatorIndex))
		return pubsub.ValidationReject
	}

	// Verify the aggregator's signature is valid.
	if err := validateAggregatorSignature(bs, signed); err != nil {
		traceutil.AnnotateError(span, errors.Wrapf(err, "Could not verify aggregator signature %d", signed.Message.AggregatorIndex))
		return pubsub.ValidationReject
	}

	// Verify aggregated attestation has a valid signature.
	if !featureconfig.Get().DisableStrictAttestationPubsubVerification {
		if err := blocks.VerifyAttestation(ctx, bs, signed.Message.Aggregate); err != nil {
			traceutil.AnnotateError(span, err)
			return pubsub.ValidationReject
		}
	}

	return pubsub.ValidationAccept
}

func (s *Service) validateBlockInAttestation(ctx context.Context, satt *ethpb.SignedAggregateAttestationAndProof) bool {
	a := satt.Message
	// Verify the block being voted and the processed state is in DB. The block should have passed validation if it's in the DB.
	blockRoot := bytesutil.ToBytes32(a.Aggregate.Data.BeaconBlockRoot)
	hasStateSummary := featureconfig.Get().NewStateMgmt && s.db.HasStateSummary(ctx, blockRoot) || s.stateSummaryCache.Has(blockRoot)
	hasState := s.db.HasState(ctx, blockRoot) || hasStateSummary
	hasBlock := s.db.HasBlock(ctx, blockRoot) || s.chain.HasInitSyncBlock(blockRoot)
	if !(hasState && hasBlock) {
		// A node doesn't have the block, it'll request from peer while saving the pending attestation to a queue.
		s.savePendingAtt(satt)
		return false
	}
	return true
}

// Returns true if the node has received aggregate for the aggregator with index and target epoch.
func (s *Service) hasSeenAggregatorIndexEpoch(epoch uint64, aggregatorIndex uint64) bool {
	s.seenAttestationLock.RLock()
	defer s.seenAttestationLock.RUnlock()
	b := append(bytesutil.Bytes32(epoch), bytesutil.Bytes32(aggregatorIndex)...)
	_, seen := s.seenAttestationCache.Get(string(b))
	return seen
}

// Set aggregate's aggregator index target epoch as seen.
func (s *Service) setAggregatorIndexEpochSeen(epoch uint64, aggregatorIndex uint64) {
	s.seenAttestationLock.Lock()
	defer s.seenAttestationLock.Unlock()
	b := append(bytesutil.Bytes32(epoch), bytesutil.Bytes32(aggregatorIndex)...)
	s.seenAttestationCache.Add(string(b), true)
}

// This validates the aggregator's index in state is within the beacon committee.
func validateIndexInCommittee(ctx context.Context, bs *stateTrie.BeaconState, a *ethpb.Attestation, validatorIndex uint64) error {
	ctx, span := trace.StartSpan(ctx, "sync.validateIndexInCommittee")
	defer span.End()

	committee, err := helpers.BeaconCommitteeFromState(bs, a.Data.Slot, a.Data.CommitteeIndex)
	if err != nil {
		return err
	}
	var withinCommittee bool
	for _, i := range committee {
		if validatorIndex == i {
			withinCommittee = true
			break
		}
	}
	if !withinCommittee {
		return fmt.Debugf("validator index %d is not within the committee: %v",
			validatorIndex, committee)
	}
	return nil
}

// This validates selection proof by validating it's from the correct validator index of the slot and selection
// proof is a valid signature.
func validateSelection(ctx context.Context, bs *stateTrie.BeaconState, data *ethpb.AttestationData, validatorIndex uint64, proof []byte) error {
	_, span := trace.StartSpan(ctx, "sync.validateSelection")
	defer span.End()

	committee, err := helpers.BeaconCommitteeFromState(bs, data.Slot, data.CommitteeIndex)
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

	if err := helpers.ComputeDomainVerifySigningRoot(bs, validatorIndex,
		helpers.SlotToEpoch(data.Slot), data.Slot, params.BeaconConfig().DomainSelectionProof, proof); err != nil {
		return err
	}

	return nil
}

// This verifies aggregator signature over the signed aggregate and proof object.
func validateAggregatorSignature(s *stateTrie.BeaconState, a *ethpb.SignedAggregateAttestationAndProof) error {
	return helpers.ComputeDomainVerifySigningRoot(s, a.Message.AggregatorIndex,
		helpers.SlotToEpoch(a.Message.Aggregate.Data.Slot), a.Message, params.BeaconConfig().DomainAggregateAndProof, a.Signature)

}
