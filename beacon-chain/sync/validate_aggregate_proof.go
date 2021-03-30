package sync

import (
	"context"
	"fmt"

	"github.com/libp2p/go-libp2p-core/peer"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/pkg/errors"
	types "github.com/prysmaticlabs/eth2-types"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/feed"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/feed/operation"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	iface "github.com/prysmaticlabs/prysm/beacon-chain/state/interface"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/traceutil"
	"go.opencensus.io/trace"
)

// validateAggregateAndProof verifies the aggregated signature and the selection proof is valid before forwarding to the
// network and downstream services.
func (s *Service) validateAggregateAndProof(ctx context.Context, pid peer.ID, msg *pubsub.Message) pubsub.ValidationResult {
	if pid == s.cfg.P2P.PeerID() {
		return pubsub.ValidationAccept
	}

	ctx, span := trace.StartSpan(ctx, "sync.validateAggregateAndProof")
	defer span.End()

	// To process the following it requires the recent blocks to be present in the database, so we'll skip
	// validating or processing aggregated attestations until fully synced.
	if s.cfg.InitialSync.Syncing() {
		return pubsub.ValidationIgnore
	}

	raw, err := s.decodePubsubMessage(msg)
	if err != nil {
		log.WithError(err).Debug("Could not decode message")
		traceutil.AnnotateError(span, err)
		return pubsub.ValidationReject
	}
	m, ok := raw.(*ethpb.SignedAggregateAttestationAndProof)
	if !ok {
		return pubsub.ValidationReject
	}
	if m.Message == nil {
		return pubsub.ValidationReject
	}
	if err := helpers.ValidateNilAttestation(m.Message.Aggregate); err != nil {
		return pubsub.ValidationReject
	}

	// Broadcast the aggregated attestation on a feed to notify other services in the beacon node
	// of a received aggregated attestation.
	s.cfg.AttestationNotifier.OperationFeed().Send(&feed.Event{
		Type: operation.AggregatedAttReceived,
		Data: &operation.AggregatedAttReceivedData{
			Attestation: m.Message,
		},
	})

	if err := helpers.ValidateSlotTargetEpoch(m.Message.Aggregate.Data); err != nil {
		return pubsub.ValidationReject
	}
	if err := helpers.ValidateAttestationTime(m.Message.Aggregate.Data.Slot, s.cfg.Chain.GenesisTime()); err != nil {
		traceutil.AnnotateError(span, err)
		return pubsub.ValidationIgnore
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
	seen, err := s.cfg.AttPool.HasAggregatedAttestation(m.Message.Aggregate)
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

	// Verify attestation target root is consistent with the head root.
	// This verification is not in the spec, however we guard against it as it opens us up
	// to weird edge cases during verification. The attestation technically could be used to add value to a block,
	// but it's invalid in the spirit of the protocol. Here we choose safety over profit.
	if err := s.cfg.Chain.VerifyLmdFfgConsistency(ctx, signed.Message.Aggregate); err != nil {
		traceutil.AnnotateError(span, err)
		return pubsub.ValidationReject
	}

	// Verify current finalized checkpoint is an ancestor of the block defined by the attestation's beacon block root.
	if err := s.cfg.Chain.VerifyFinalizedConsistency(ctx, signed.Message.Aggregate.Data.BeaconBlockRoot); err != nil {
		traceutil.AnnotateError(span, err)
		return pubsub.ValidationReject
	}

	bs, err := s.cfg.Chain.AttestationPreState(ctx, signed.Message.Aggregate)
	if err != nil {
		traceutil.AnnotateError(span, err)
		return pubsub.ValidationIgnore
	}

	attSlot := signed.Message.Aggregate.Data.Slot
	// Only advance state if different epoch as the committee can only change on an epoch transition.
	if helpers.SlotToEpoch(attSlot) > helpers.SlotToEpoch(bs.Slot()) {
		startSlot, err := helpers.StartSlot(helpers.SlotToEpoch(attSlot))
		if err != nil {
			return pubsub.ValidationIgnore
		}
		bs, err = state.ProcessSlots(ctx, bs, startSlot)
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

	// Verify selection proof reflects to the right validator.
	selectionSigSet, err := validateSelectionIndex(ctx, bs, signed.Message.Aggregate.Data, signed.Message.AggregatorIndex, signed.Message.SelectionProof)
	if err != nil {
		traceutil.AnnotateError(span, errors.Wrapf(err, "Could not validate selection for validator %d", signed.Message.AggregatorIndex))
		return pubsub.ValidationReject
	}

	// Verify selection signature, aggregator signature and attestation signature are valid.
	// We use batch verify here to save compute.
	aggregatorSigSet, err := aggSigSet(bs, signed)
	if err != nil {
		traceutil.AnnotateError(span, errors.Wrapf(err, "Could not get aggregator sig set %d", signed.Message.AggregatorIndex))
		return pubsub.ValidationIgnore
	}
	attSigSet, err := blocks.AttestationSignatureSet(ctx, bs, []*ethpb.Attestation{signed.Message.Aggregate})
	if err != nil {
		traceutil.AnnotateError(span, errors.Wrapf(err, "Could not verify aggregator signature %d", signed.Message.AggregatorIndex))
		return pubsub.ValidationIgnore
	}
	set := bls.NewSet()
	set.Join(selectionSigSet).Join(aggregatorSigSet).Join(attSigSet)
	valid, err := set.Verify()
	if err != nil {
		traceutil.AnnotateError(span, errors.Errorf("Could not join signature set"))
		return pubsub.ValidationIgnore
	}
	if !valid {
		traceutil.AnnotateError(span, errors.Errorf("Could not verify selection or aggregator or attestation signature"))
		return pubsub.ValidationReject
	}

	return pubsub.ValidationAccept
}

func (s *Service) validateBlockInAttestation(ctx context.Context, satt *ethpb.SignedAggregateAttestationAndProof) bool {
	a := satt.Message
	// Verify the block being voted and the processed state is in DB. The block should have passed validation if it's in the DB.
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
	s.seenAttestationLock.RLock()
	defer s.seenAttestationLock.RUnlock()
	b := append(bytesutil.Bytes32(uint64(epoch)), bytesutil.Bytes32(uint64(aggregatorIndex))...)
	_, seen := s.seenAttestationCache.Get(string(b))
	return seen
}

// Set aggregate's aggregator index target epoch as seen.
func (s *Service) setAggregatorIndexEpochSeen(epoch types.Epoch, aggregatorIndex types.ValidatorIndex) {
	s.seenAttestationLock.Lock()
	defer s.seenAttestationLock.Unlock()
	b := append(bytesutil.Bytes32(uint64(epoch)), bytesutil.Bytes32(uint64(aggregatorIndex))...)
	s.seenAttestationCache.Add(string(b), true)
}

// This validates the aggregator's index in state is within the beacon committee.
func validateIndexInCommittee(ctx context.Context, bs iface.ReadOnlyBeaconState, a *ethpb.Attestation, validatorIndex types.ValidatorIndex) error {
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
		return fmt.Errorf("validator index %d is not within the committee: %v",
			validatorIndex, committee)
	}
	return nil
}

// This validates selection proof by validating it's from the correct validator index of the slot.
// It does not verify the selection proof, it returns the signature set of selection proof which can be used for batch verify.
func validateSelectionIndex(ctx context.Context, bs iface.ReadOnlyBeaconState, data *ethpb.AttestationData, validatorIndex types.ValidatorIndex, proof []byte) (*bls.SignatureSet, error) {
	_, span := trace.StartSpan(ctx, "sync.validateSelectionIndex")
	defer span.End()

	committee, err := helpers.BeaconCommitteeFromState(bs, data.Slot, data.CommitteeIndex)
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
	epoch := helpers.SlotToEpoch(data.Slot)

	v, err := bs.ValidatorAtIndex(validatorIndex)
	if err != nil {
		return nil, err
	}
	publicKey, err := bls.PublicKeyFromBytes(v.PublicKey)
	if err != nil {
		return nil, err
	}

	d, err := helpers.Domain(bs.Fork(), epoch, domain, bs.GenesisValidatorRoot())
	if err != nil {
		return nil, err
	}
	sszUint := types.SSZUint64(data.Slot)
	root, err := helpers.ComputeSigningRoot(&sszUint, d)
	if err != nil {
		return nil, err
	}
	return &bls.SignatureSet{
		Signatures: [][]byte{proof},
		PublicKeys: []bls.PublicKey{publicKey},
		Messages:   [][32]byte{root},
	}, nil
}

// This returns aggregator signature set which can be used to batch verify.
func aggSigSet(s iface.ReadOnlyBeaconState, a *ethpb.SignedAggregateAttestationAndProof) (*bls.SignatureSet, error) {
	v, err := s.ValidatorAtIndex(a.Message.AggregatorIndex)
	if err != nil {
		return nil, err
	}
	publicKey, err := bls.PublicKeyFromBytes(v.PublicKey)
	if err != nil {
		return nil, err
	}

	epoch := helpers.SlotToEpoch(a.Message.Aggregate.Data.Slot)
	d, err := helpers.Domain(s.Fork(), epoch, params.BeaconConfig().DomainAggregateAndProof, s.GenesisValidatorRoot())
	if err != nil {
		return nil, err
	}
	root, err := helpers.ComputeSigningRoot(a.Message, d)
	if err != nil {
		return nil, err
	}
	return &bls.SignatureSet{
		Signatures: [][]byte{a.Signature},
		PublicKeys: []bls.PublicKey{publicKey},
		Messages:   [][32]byte{root},
	}, nil
}
