package sync

import (
	"context"
	"fmt"

	"github.com/libp2p/go-libp2p-core/peer"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/feed"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/feed/operation"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/signing"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/transition"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v3/crypto/bls"
	"github.com/prysmaticlabs/prysm/v3/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/v3/monitoring/tracing"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/time/slots"
	"go.opencensus.io/trace"
)

// validateAggregateAndProof verifies the aggregated signature and the selection proof is valid before forwarding to the
// network and downstream services.
func (s *Service) validateAggregateAndProof(ctx context.Context, pid peer.ID, msg *pubsub.Message) (pubsub.ValidationResult, error) {
	if pid == s.cfg.p2p.PeerID() {
		return pubsub.ValidationAccept, nil
	}

	ctx, span := trace.StartSpan(ctx, "sync.validateAggregateAndProof")
	defer span.End()

	// To process the following it requires the recent blocks to be present in the database, so we'll skip
	// validating or processing aggregated attestations until fully synced.
	if s.cfg.initialSync.Syncing() {
		return pubsub.ValidationIgnore, nil
	}

	raw, err := s.decodePubsubMessage(msg)
	if err != nil {
		tracing.AnnotateError(span, err)
		return pubsub.ValidationReject, err
	}
	m, ok := raw.(*ethpb.SignedAggregateAttestationAndProof)
	if !ok {
		return pubsub.ValidationReject, errors.Errorf("invalid message type: %T", raw)
	}
	if m.Message == nil {
		return pubsub.ValidationReject, errNilMessage
	}
	if err := helpers.ValidateNilAttestation(m.Message.Aggregate); err != nil {
		return pubsub.ValidationReject, err
	}
	// Do not process slot 0 aggregates.
	if m.Message.Aggregate.Data.Slot == 0 {
		return pubsub.ValidationIgnore, nil
	}

	// Broadcast the aggregated attestation on a feed to notify other services in the beacon node
	// of a received aggregated attestation.
	s.cfg.attestationNotifier.OperationFeed().Send(&feed.Event{
		Type: operation.AggregatedAttReceived,
		Data: &operation.AggregatedAttReceivedData{
			Attestation: m.Message,
		},
	})

	if err := helpers.ValidateSlotTargetEpoch(m.Message.Aggregate.Data); err != nil {
		return pubsub.ValidationReject, err
	}

	// Attestation's slot is within ATTESTATION_PROPAGATION_SLOT_RANGE and early attestation
	// processing tolerance.
	if err := helpers.ValidateAttestationTime(m.Message.Aggregate.Data.Slot, s.cfg.chain.GenesisTime(),
		earlyAttestationProcessingTolerance); err != nil {
		tracing.AnnotateError(span, err)
		return pubsub.ValidationIgnore, err
	}

	// Verify this is the first aggregate received from the aggregator with index and slot.
	if s.hasSeenAggregatorIndexEpoch(m.Message.Aggregate.Data.Target.Epoch, m.Message.AggregatorIndex) {
		return pubsub.ValidationIgnore, nil
	}
	// Check that the block being voted on isn't invalid.
	if s.hasBadBlock(bytesutil.ToBytes32(m.Message.Aggregate.Data.BeaconBlockRoot)) ||
		s.hasBadBlock(bytesutil.ToBytes32(m.Message.Aggregate.Data.Target.Root)) ||
		s.hasBadBlock(bytesutil.ToBytes32(m.Message.Aggregate.Data.Source.Root)) {
		return pubsub.ValidationReject, errors.New("bad block referenced in attestation data")
	}

	// Verify aggregate attestation has not already been seen via aggregate gossip, within a block, or through the creation locally.
	seen, err := s.cfg.attPool.HasAggregatedAttestation(m.Message.Aggregate)
	if err != nil {
		tracing.AnnotateError(span, err)
		return pubsub.ValidationIgnore, err
	}
	if seen {
		return pubsub.ValidationIgnore, nil
	}
	if !s.validateBlockInAttestation(ctx, m) {
		return pubsub.ValidationIgnore, nil
	}

	validationRes, err := s.validateAggregatedAtt(ctx, m)
	if validationRes != pubsub.ValidationAccept {
		return validationRes, err
	}

	s.setAggregatorIndexEpochSeen(m.Message.Aggregate.Data.Target.Epoch, m.Message.AggregatorIndex)

	msg.ValidatorData = m

	return pubsub.ValidationAccept, nil
}

func (s *Service) validateAggregatedAtt(ctx context.Context, signed *ethpb.SignedAggregateAttestationAndProof) (pubsub.ValidationResult, error) {
	ctx, span := trace.StartSpan(ctx, "sync.validateAggregatedAtt")
	defer span.End()

	// Verify attestation target root is consistent with the head root.
	// This verification is not in the spec, however we guard against it as it opens us up
	// to weird edge cases during verification. The attestation technically could be used to add value to a block,
	// but it's invalid in the spirit of the protocol. Here we choose safety over profit.
	if err := s.cfg.chain.VerifyLmdFfgConsistency(ctx, signed.Message.Aggregate); err != nil {
		tracing.AnnotateError(span, err)
		return pubsub.ValidationReject, err
	}

	// Verify current finalized checkpoint is an ancestor of the block defined by the attestation's beacon block root.
	if err := s.cfg.chain.VerifyFinalizedConsistency(ctx, signed.Message.Aggregate.Data.BeaconBlockRoot); err != nil {
		tracing.AnnotateError(span, err)
		return pubsub.ValidationIgnore, err
	}

	bs, err := s.cfg.chain.AttestationTargetState(ctx, signed.Message.Aggregate.Data.Target)
	if err != nil {
		tracing.AnnotateError(span, err)
		return pubsub.ValidationIgnore, err
	}

	attSlot := signed.Message.Aggregate.Data.Slot
	// Only advance state if different epoch as the committee can only change on an epoch transition.
	if slots.ToEpoch(attSlot) > slots.ToEpoch(bs.Slot()) {
		startSlot, err := slots.EpochStart(slots.ToEpoch(attSlot))
		if err != nil {
			return pubsub.ValidationIgnore, err
		}
		bs, err = transition.ProcessSlots(ctx, bs, startSlot)
		if err != nil {
			tracing.AnnotateError(span, err)
			return pubsub.ValidationIgnore, err
		}
	}

	// Verify validator index is within the beacon committee.
	if err := validateIndexInCommittee(ctx, bs, signed.Message.Aggregate, signed.Message.AggregatorIndex); err != nil {
		wrappedErr := errors.Wrapf(err, "Could not validate index in committee")
		tracing.AnnotateError(span, wrappedErr)
		return pubsub.ValidationReject, wrappedErr
	}

	// Verify selection proof reflects to the right validator.
	selectionSigSet, err := validateSelectionIndex(ctx, bs, signed.Message.Aggregate.Data, signed.Message.AggregatorIndex, signed.Message.SelectionProof)
	if err != nil {
		wrappedErr := errors.Wrapf(err, "Could not validate selection for validator %d", signed.Message.AggregatorIndex)
		tracing.AnnotateError(span, wrappedErr)
		return pubsub.ValidationReject, wrappedErr
	}

	// Verify selection signature, aggregator signature and attestation signature are valid.
	// We use batch verify here to save compute.
	aggregatorSigSet, err := aggSigSet(bs, signed)
	if err != nil {
		wrappedErr := errors.Wrapf(err, "Could not get aggregator sig set %d", signed.Message.AggregatorIndex)
		tracing.AnnotateError(span, wrappedErr)
		return pubsub.ValidationIgnore, wrappedErr
	}
	attSigSet, err := blocks.AttestationSignatureBatch(ctx, bs, []*ethpb.Attestation{signed.Message.Aggregate})
	if err != nil {
		wrappedErr := errors.Wrapf(err, "Could not verify aggregator signature %d", signed.Message.AggregatorIndex)
		tracing.AnnotateError(span, wrappedErr)
		return pubsub.ValidationIgnore, wrappedErr
	}
	set := bls.NewSet()
	set.Join(selectionSigSet).Join(aggregatorSigSet).Join(attSigSet)

	return s.validateWithBatchVerifier(ctx, "aggregate", set)
}

func (s *Service) validateBlockInAttestation(ctx context.Context, satt *ethpb.SignedAggregateAttestationAndProof) bool {
	a := satt.Message
	// Verify the block being voted and the processed state is in beaconDB. The block should have passed validation if it's in the beaconDB.
	blockRoot := bytesutil.ToBytes32(a.Aggregate.Data.BeaconBlockRoot)
	if !s.hasBlockAndState(ctx, blockRoot) {
		// A node doesn't have the block, it'll request from peer while saving the pending attestation to a queue.
		s.savePendingAtt(satt)
		return false
	}
	return true
}

// Returns true if the node has received aggregate for the aggregator with index and target epoch.
func (s *Service) hasSeenAggregatorIndexEpoch(epoch types.Epoch, aggregatorIndex types.ValidatorIndex) bool {
	s.seenAggregatedAttestationLock.RLock()
	defer s.seenAggregatedAttestationLock.RUnlock()
	b := append(bytesutil.Bytes32(uint64(epoch)), bytesutil.Bytes32(uint64(aggregatorIndex))...)
	_, seen := s.seenAggregatedAttestationCache.Get(string(b))
	return seen
}

// Set aggregate's aggregator index target epoch as seen.
func (s *Service) setAggregatorIndexEpochSeen(epoch types.Epoch, aggregatorIndex types.ValidatorIndex) {
	s.seenAggregatedAttestationLock.Lock()
	defer s.seenAggregatedAttestationLock.Unlock()
	b := append(bytesutil.Bytes32(uint64(epoch)), bytesutil.Bytes32(uint64(aggregatorIndex))...)
	s.seenAggregatedAttestationCache.Add(string(b), true)
}

// This validates the aggregator's index in state is within the beacon committee.
func validateIndexInCommittee(ctx context.Context, bs state.ReadOnlyBeaconState, a *ethpb.Attestation, validatorIndex types.ValidatorIndex) error {
	ctx, span := trace.StartSpan(ctx, "sync.validateIndexInCommittee")
	defer span.End()

	committee, err := helpers.BeaconCommitteeFromState(ctx, bs, a.Data.Slot, a.Data.CommitteeIndex)
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
		return fmt.Errorf("validator index %d is not within the committee: %v",
			validatorIndex, committee)
	}
	return nil
}

// This validates selection proof by validating it's from the correct validator index of the slot.
// It does not verify the selection proof, it returns the signature set of selection proof which can be used for batch verify.
func validateSelectionIndex(
	ctx context.Context,
	bs state.ReadOnlyBeaconState,
	data *ethpb.AttestationData,
	validatorIndex types.ValidatorIndex,
	proof []byte,
) (*bls.SignatureBatch, error) {
	ctx, span := trace.StartSpan(ctx, "sync.validateSelectionIndex")
	defer span.End()

	committee, err := helpers.BeaconCommitteeFromState(ctx, bs, data.Slot, data.CommitteeIndex)
	if err != nil {
		return nil, err
	}
	aggregator, err := helpers.IsAggregator(uint64(len(committee)), proof)
	if err != nil {
		return nil, err
	}
	if !aggregator {
		return nil, fmt.Errorf("validator is not an aggregator for slot %d", data.Slot)
	}

	domain := params.BeaconConfig().DomainSelectionProof
	epoch := slots.ToEpoch(data.Slot)

	v, err := bs.ValidatorAtIndex(validatorIndex)
	if err != nil {
		return nil, err
	}
	publicKey, err := bls.PublicKeyFromBytes(v.PublicKey)
	if err != nil {
		return nil, err
	}

	d, err := signing.Domain(bs.Fork(), epoch, domain, bs.GenesisValidatorsRoot())
	if err != nil {
		return nil, err
	}
	sszUint := types.SSZUint64(data.Slot)
	root, err := signing.ComputeSigningRoot(&sszUint, d)
	if err != nil {
		return nil, err
	}
	return &bls.SignatureBatch{
		Signatures: [][]byte{proof},
		PublicKeys: []bls.PublicKey{publicKey},
		Messages:   [][32]byte{root},
	}, nil
}

// This returns aggregator signature set which can be used to batch verify.
func aggSigSet(s state.ReadOnlyBeaconState, a *ethpb.SignedAggregateAttestationAndProof) (*bls.SignatureBatch, error) {
	v, err := s.ValidatorAtIndex(a.Message.AggregatorIndex)
	if err != nil {
		return nil, err
	}
	publicKey, err := bls.PublicKeyFromBytes(v.PublicKey)
	if err != nil {
		return nil, err
	}

	epoch := slots.ToEpoch(a.Message.Aggregate.Data.Slot)
	d, err := signing.Domain(s.Fork(), epoch, params.BeaconConfig().DomainAggregateAndProof, s.GenesisValidatorsRoot())
	if err != nil {
		return nil, err
	}
	root, err := signing.ComputeSigningRoot(a.Message, d)
	if err != nil {
		return nil, err
	}
	return &bls.SignatureBatch{
		Signatures: [][]byte{a.Signature},
		PublicKeys: []bls.PublicKey{publicKey},
		Messages:   [][32]byte{root},
	}, nil
}
